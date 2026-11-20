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
	"github.com/exf1kzz/go-labs-avito/internal/container"
)

func main() {
	if err := runWithSignal(); err != nil {
		log.Fatalf("run service: %v", err)
	}
}

func runWithSignal() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	return run(ctx)
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	app, err := container.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("build container: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.HTTP.Address,
		Handler:           app.Handler,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	app.Logger.Info("HTTP server listening", "address", server.Addr)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	var serveErr error
	serverStopped := false

	select {
	case err := <-serverErrors:
		serverStopped = true
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr = fmt.Errorf("serve HTTP: %w", err)
		}
	case <-ctx.Done():
		app.Logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	var shutdownErr error
	if !serverStopped {
		if err := server.Shutdown(shutdownCtx); err != nil {
			app.Logger.Error("graceful HTTP shutdown failed", "error", err)
			shutdownErr = fmt.Errorf("shutdown HTTP server: %w", err)

			if closeErr := server.Close(); closeErr != nil {
				shutdownErr = errors.Join(
					shutdownErr,
					fmt.Errorf("force close HTTP server: %w", closeErr),
				)
			}
		} else {
			app.Logger.Info("HTTP server stopped")
		}

		if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr = errors.Join(serveErr, fmt.Errorf("serve HTTP during shutdown: %w", err))
		}
	}

	closeErr := app.Close(shutdownCtx)
	if closeErr != nil {
		app.Logger.Error("close application", "error", closeErr)
	}

	return errors.Join(serveErr, shutdownErr, closeErr)
}
