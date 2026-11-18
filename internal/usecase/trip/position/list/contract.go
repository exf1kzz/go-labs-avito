package list

import (
	"context"

	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=list_test
type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
	ListPositions(ctx context.Context, tripID uuid.UUID) ([]domaintrip.Position, error)
}
