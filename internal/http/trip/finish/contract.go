package finish

import (
	"context"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/google/uuid"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish_test
type usecase interface {
	Execute(context.Context, uuid.UUID) (domaintrip.Trip, error)
}
