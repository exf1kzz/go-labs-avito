package create

import (
	"context"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=create_test
type usecase interface {
	Execute(context.Context, model.CreateTripInput) (domaintrip.Trip, error)
}
