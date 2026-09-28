package service

import (
	"context"
	"fmt"

	"github.com/exf1kzz/go-labs-avito/internal/model"
	"github.com/google/uuid"
)

func (s *TripService) Get(
	ctx context.Context,
	tripID uuid.UUID,
) (model.Trip, error) {
	trip, err := s.repository.GetByID(ctx, tripID)
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}
