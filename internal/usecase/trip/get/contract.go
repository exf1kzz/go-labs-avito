package get

import (
	"context"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/google/uuid"
)

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=get_test
type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
}
