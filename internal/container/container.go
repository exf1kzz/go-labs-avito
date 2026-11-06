package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

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
	Handler http.Handler
	pool    *pgxpool.Pool
	Logger  *slog.Logger
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	txManager := transaction.NewTransactionManager(pool)
	tripRepository := triprepo.NewTripRepository(txManager)

	logger := telemetry.NewLogger(cfg.LogLevel)

	timeout := cfg.Database.QueryTimeout
	tripHandler := transport.NewServer(
		pool,
		createhttp.New(createuc.New(tripRepository, txManager), timeout, logger),
		gethttp.New(getuc.New(tripRepository), timeout, logger),
		finishhttp.New(finishuc.New(tripRepository, txManager), timeout, logger),
		timeout,
	)

	return &Container{
		Handler: transport.New(tripHandler, logger),
		pool:    pool,
		Logger:  logger,
	}, nil
}

func (c *Container) Close() {
	c.pool.Close()
}
