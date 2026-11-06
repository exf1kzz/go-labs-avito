package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

type Usecase struct {
	repository tripRepository
	txManager  txManager
}

func New(repository tripRepository, txManager txManager) *Usecase {
	return &Usecase{
		repository: repository,
		txManager:  txManager,
	}
}

func (u *Usecase) Execute(ctx context.Context, input model.CreateTripInput) (domaintrip.Trip, error) {
	trip := domaintrip.Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domaintrip.ActiveStatus,
		StartedAt:  time.Now().UTC(),
	}

	if err := trip.ValidateCreate(); err != nil {
		return domaintrip.Trip{}, fmt.Errorf("validate trip: %w", err)
	}

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.repository.Create(txCtx, trip); err != nil {
			return err
		}
		return u.repository.AddStatusHistory(
			txCtx, trip.ID, nil, domaintrip.ActiveStatus, nil, trip.StartedAt,
		)
	})
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	return trip, nil
}
