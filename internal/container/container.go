package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/exf1kzz/go-labs-avito/internal/config"
	pushgateway "github.com/exf1kzz/go-labs-avito/internal/gateway/push"
	transport "github.com/exf1kzz/go-labs-avito/internal/http"
	createhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/create"
	finishhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/finish"
	gethttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/get"
	positioncreatehttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/position/create"
	positionlisthttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/position/list"
	"github.com/exf1kzz/go-labs-avito/internal/infra/postgres"
	"github.com/exf1kzz/go-labs-avito/internal/infra/postgres/transaction"
	"github.com/exf1kzz/go-labs-avito/internal/infra/telemetry"
	triprepo "github.com/exf1kzz/go-labs-avito/internal/repository/trip"
	createuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/create"
	finishuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/finish"
	getuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/get"
	positioncreateuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/position/create"
	positionlistuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/position/list"
	"github.com/exf1kzz/go-labs-avito/internal/worker/positions"
)

type Container struct {
	Handler        http.Handler
	pool           *pgxpool.Pool
	Logger         *slog.Logger
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	PushGateway    *pushgateway.Gateway
	pushClient     *http.Client
	positionPool   *positions.Pool
	scannerCancel  context.CancelFunc
	scannerDone    chan struct{}
	scannerTicker  positions.Ticker
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	logger := telemetry.NewLogger(cfg.LogLevel)

	pushClient := &http.Client{Timeout: cfg.Push.RequestTimeout}
	pushGateway, err := pushgateway.New(cfg.Push.URL, pushClient)
	if err != nil {
		return nil, fmt.Errorf("create push gateway: %w", err)
	}

	tracerProvider, err := telemetry.NewTracerProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("create tracer provider: %w", err)
	}

	pushClient.Transport = otelhttp.NewTransport(
		http.DefaultTransport,
		otelhttp.WithTracerProvider(tracerProvider),
		otelhttp.WithPropagators(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)),
	)

	meterProvider, err := telemetry.NewMeterProvider(ctx)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create meter provider: %w", err),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	activeRequests, err := meterProvider.Meter("trip-service/http").
		Int64UpDownCounter("http.server.active_requests", metric.WithUnit("{request}"))
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create active requests counter: %w", err),
			meterProvider.Shutdown(cleanupCtx),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	tripMeter := meterProvider.Meter("trip-service/trip")

	tripsCreated, err := tripMeter.Int64Counter("tripgo.trips.created")
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create trips created counter: %w", err),
			meterProvider.Shutdown(cleanupCtx),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	tripsCompleted, err := tripMeter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create trips completed counter: %w", err),
			meterProvider.Shutdown(cleanupCtx),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	positionsSaved, err := tripMeter.Int64Counter("tripgo.positions.saved")
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create positions saved counter: %w", err),
			meterProvider.Shutdown(cleanupCtx),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	staleChecks, err := tripMeter.Int64Counter("tripgo.stale_checks")
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create stale checks counter: %w", err),
			meterProvider.Shutdown(cleanupCtx),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	pool, err := postgres.NewPool(ctx, cfg.Database, tracerProvider, meterProvider)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create pool: %w", err),
			meterProvider.Shutdown(cleanupCtx),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	txManager := transaction.NewTransactionManager(pool)
	tripRepository := triprepo.NewTripRepository(txManager)

	positionPool, err := positions.NewPool(
		tripRepository,
		pushGateway,
		positions.PoolConfig{
			Workers:        cfg.Positions.Workers,
			QueueSize:      cfg.Positions.QueueSize,
			StaleAfter:     cfg.Positions.StaleAfter,
			QueryTimeout:   cfg.Database.QueryTimeout,
			RequestTimeout: cfg.Push.RequestTimeout,
		},
		time.Now,
		logger,
		meterProvider.Meter("trip-service/positions"),
		tracerProvider.Tracer("trip-service/positions"),
	)
	if err != nil {
		pool.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		return nil, errors.Join(
			fmt.Errorf("create positions pool: %w", err),
			meterProvider.Shutdown(cleanupCtx),
			tracerProvider.Shutdown(cleanupCtx),
		)
	}

	timeout := cfg.Database.QueryTimeout
	tripHandler := transport.NewServer(
		pool,
		createhttp.New(createuc.New(tripRepository, txManager), timeout, logger, tripsCreated),
		gethttp.New(getuc.New(tripRepository), timeout, logger),
		finishhttp.New(finishuc.New(tripRepository, txManager), timeout, logger, tripsCompleted),
		positioncreatehttp.New(positioncreateuc.New(tripRepository, txManager, positionPool, time.Now), timeout, logger, positionsSaved),
		positionlisthttp.New(positionlistuc.New(tripRepository), timeout, logger),
		timeout,
	)

	scanner := positions.New(
		tripRepository,
		positionPool,
		cfg.Positions.StaleAfter,
		cfg.Database.QueryTimeout,
		logger,
		staleChecks,
		tracerProvider.Tracer("trip-service/positions"),
	)

	scannerCtx, scannerCancel := context.WithCancel(context.WithoutCancel(ctx))
	scannerTicker := positions.NewSystemTicker(cfg.Positions.CheckInterval)
	scannerDone := make(chan struct{})

	go func() {
		defer close(scannerDone)
		scanner.Run(scannerCtx, scannerTicker)
	}()

	logger.Info("positions scanner started",
		"interval", cfg.Positions.CheckInterval,
	)

	return &Container{
		Handler:        transport.New(tripHandler, logger, tracerProvider, meterProvider, activeRequests),
		pool:           pool,
		Logger:         logger,
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		PushGateway:    pushGateway,
		pushClient:     pushClient,
		positionPool:   positionPool,
		scannerCancel:  scannerCancel,
		scannerDone:    scannerDone,
		scannerTicker:  scannerTicker,
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	c.Logger.Info("stopping positions scanner")
	c.scannerCancel()
	c.scannerTicker.Stop()

	select {
	case <-c.scannerDone:
		c.Logger.Info("positions scanner stopped")
	case <-ctx.Done():
		c.positionPool.Close()
		c.positionPool.Abort()
		return fmt.Errorf("stop positions scanner: %w", ctx.Err())
	}

	c.Logger.Info("draining position workers")
	c.positionPool.Close()

	workersDone := make(chan struct{})
	go func() {
		c.positionPool.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		c.Logger.Info("position workers stopped")
	case <-ctx.Done():
		c.positionPool.Abort()
		select {
		case <-workersDone:
			c.Logger.Info("position workers stopped after cancellation")
		default:
			return fmt.Errorf("drain position workers: %w", ctx.Err())
		}
	}

	c.pushClient.CloseIdleConnections()
	c.pool.Close()

	return errors.Join(
		c.meterProvider.Shutdown(ctx),
		c.tracerProvider.Shutdown(ctx),
	)
}
