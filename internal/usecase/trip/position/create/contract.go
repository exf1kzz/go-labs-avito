package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create_test
type tripRepository interface {
	GetStatusForUpdate(ctx context.Context, tripID uuid.UUID) (domaintrip.TripStatus, error)
	InsertPosition(ctx context.Context, position domaintrip.Position) (domaintrip.Position, error)
	UpdateLastPositionAt(ctx context.Context, tripID uuid.UUID, recordedAt time.Time) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
