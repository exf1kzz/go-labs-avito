package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, at time.Time) (domaintrip.Trip, error)
	AddStatusHistory(
		ctx context.Context,
		tripID uuid.UUID,
		fromStatus *domaintrip.TripStatus,
		toStatus domaintrip.TripStatus,
		reason *string,
		changedAt time.Time,
	) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
