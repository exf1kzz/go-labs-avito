package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=positions_test
type staleRepository interface {
	ListStaleActive(ctx context.Context, duration time.Duration) ([]uuid.UUID, error)
}

type jobQueue interface {
	TrySubmit(context.Context, uuid.UUID) SubmitResult
}

type tripReader interface {
	GetByID(ctx context.Context, tripID uuid.UUID) (domaintrip.Trip, error)
}

type pushSender interface {
	RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
}
