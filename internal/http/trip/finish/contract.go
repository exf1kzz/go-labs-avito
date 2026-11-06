package finish

import (
	"context"

	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
type usecase interface {
	Execute(context.Context, uuid.UUID) (domaintrip.Trip, error)
}
