package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

type Usecase struct {
	repository tripRepository
}

func New(repository tripRepository) *Usecase {
	return &Usecase{repository: repository}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	trip, err := u.repository.GetByID(ctx, id)
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}
