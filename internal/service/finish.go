package service

import (
	"context"
	"fmt"
	"time"

	"github.com/exf1kzz/go-labs-avito/internal/model"
	"github.com/google/uuid"
)

func (s *TripService) Finish(
	ctx context.Context,
	tripID uuid.UUID,
) (model.Trip, error) {
	finishedAt := time.Now().UTC()

	var trip model.Trip

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		var err error

		trip, err = s.repository.Finish(
			txCtx,
			tripID,
			finishedAt,
		)
		if err != nil {
			return err
		}

		fromStatus := model.ActiveStatus

		if err := s.repository.AddStatusHistory(
			txCtx,
			tripID,
			&fromStatus,
			model.CompletedStatus,
			nil,
			finishedAt,
		); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	return trip, nil
}
