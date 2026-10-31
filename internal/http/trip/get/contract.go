package get

import (
	"context"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/google/uuid"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get_test
type usecase interface {
	Execute(context.Context, uuid.UUID) (domaintrip.Trip, error)
}
