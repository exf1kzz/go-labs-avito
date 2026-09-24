package main

import (
	"context"
	"fmt"
	"log"

	"github.com/exf1kzz/go-labs-avito/internal/config"
	"github.com/exf1kzz/go-labs-avito/internal/postgres"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatalf("run service: %v", err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	defer pool.Close()

	log.Printf(
		"database connection established: max_conns=%d",
		cfg.Database.MaxConns,
	)

	return nil
}
