package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/exf1kzz/go-labs-avito/internal/config"
	transport "github.com/exf1kzz/go-labs-avito/internal/http"
	createhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/create"
	finishhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/finish"
	gethttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/get"
	"github.com/exf1kzz/go-labs-avito/internal/infra/postgres"
	"github.com/exf1kzz/go-labs-avito/internal/infra/postgres/transaction"
	"github.com/exf1kzz/go-labs-avito/internal/infra/telemetry"
	triprepo "github.com/exf1kzz/go-labs-avito/internal/repository/trip"
	createuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/create"
	finishuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/finish"
	getuc "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/get"
)

type Container struct {
	Handler        http.Handler
	pool           *pgxpool.Pool
	Logger         *slog.Logger
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	logger := telemetry.NewLogger(cfg.LogLevel)

	tracerProvider, err := telemetry.NewTracerProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("create tracer provider: %w", err)
	}

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

	timeout := cfg.Database.QueryTimeout
	tripHandler := transport.NewServer(
		pool,
		createhttp.New(createuc.New(tripRepository, txManager), timeout, logger, tripsCreated),
		gethttp.New(getuc.New(tripRepository), timeout, logger),
		finishhttp.New(finishuc.New(tripRepository, txManager), timeout, logger, tripsCompleted),
		timeout,
	)

	return &Container{
		Handler:        transport.New(tripHandler, logger, tracerProvider, meterProvider, activeRequests),
		pool:           pool,
		Logger:         logger,
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
	}, nil
}

func (c *Container) Close(ctx context.Context) error {
	c.pool.Close()
	return errors.Join(
		c.meterProvider.Shutdown(ctx),
		c.tracerProvider.Shutdown(ctx),
	)
}
