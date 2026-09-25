package handler

import (
	"context"
	"time"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

type databasePinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	api.Unimplemented

	database databasePinger
	queryTimeout time.Duration
}

func New(
	pinger databasePinger,
	queryTimeout time.Duration,
	) *Handler {
	return &Handler{
		database: pinger,
		queryTimeout: queryTimeout,
	}
}

var _ api.ServerInterface = (*Handler)(nil)
