package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/exf1kzz/go-labs-avito/internal/config"
	"github.com/exf1kzz/go-labs-avito/internal/handler"
	"github.com/exf1kzz/go-labs-avito/internal/postgres"
	"github.com/exf1kzz/go-labs-avito/internal/router"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(ctx); err != nil {
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

	tripHandler := handler.New(pool, cfg.Database.QueryTimeout)
	httpHandler := router.New(tripHandler)

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           httpHandler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	log.Printf("HTTP server listening on: %s", server.Addr)

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}

		return nil
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful HTTP shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil {
			return fmt.Errorf("force close HTTP server: %w", closeErr)
		}

		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}

	log.Printf("HTTP server stopped")

	return nil
}
