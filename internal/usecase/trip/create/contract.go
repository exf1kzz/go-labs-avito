package create

import (
	"context"
	"time"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/google/uuid"
)

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=create_test
type tripRepository interface {
	Create(ctx context.Context, trip domaintrip.Trip) error
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
