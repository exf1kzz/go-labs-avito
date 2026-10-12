package service

import (
	"context"
	"fmt"
	"time"

	"github.com/exf1kzz/go-labs-avito/internal/model"
	"github.com/exf1kzz/go-labs-avito/internal/repository"
	"github.com/google/uuid"
)

type CreateTripInput struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint model.Point
	EndPoint   model.Point
	Price      int64
}

type TripService struct {
	repository *repository.TripRepository
	txManager  TxManager
}

func NewTripService(
	repository *repository.TripRepository,
	txManager TxManager,
) *TripService {
	return &TripService{
		repository: repository,
		txManager:  txManager,
	}
}

func (s *TripService) Create(
	ctx context.Context,
	input CreateTripInput,
) (model.Trip, error) {
	trip := model.Trip{
		ID:         uuid.New(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     model.ActiveStatus,
		StartedAt:  time.Now().UTC(),
	}

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.repository.Create(txCtx, trip); err != nil {
			return err
		}

		if err := s.repository.AddStatusHistory(
			txCtx,
			trip.ID,
			nil,
			model.ActiveStatus,
			nil,
			trip.StartedAt,
		); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	return trip, nil
}
