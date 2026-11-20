package list

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

func (u *Usecase) Execute(
	ctx context.Context,
	tripID uuid.UUID,
) ([]domaintrip.Position, error) {
	if _, err := u.repository.GetByID(ctx, tripID); err != nil {
		return nil, fmt.Errorf("get trip for route: %w", err)
	}

	positions, err := u.repository.ListPositions(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("list trip positions: %w", err)
	}
	return positions, nil
}
