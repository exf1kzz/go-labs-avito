package handler

import (
	"context"
	"time"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/service"
)

type databasePinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	api.Unimplemented

	database     databasePinger
	tripService  *service.TripService
	queryTimeout time.Duration
}

func New(
	pinger databasePinger,
	tripService *service.TripService,
	queryTimeout time.Duration,
) *Handler {
	return &Handler{
		database:     pinger,
		tripService:  tripService,
		queryTimeout: queryTimeout,
	}
}

var _ api.ServerInterface = (*Handler)(nil)
