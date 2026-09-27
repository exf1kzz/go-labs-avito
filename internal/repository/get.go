package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/exf1kzz/go-labs-avito/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *TripRepository) GetByID(
	ctx context.Context,
	tripID uuid.UUID,
) (model.Trip, error) {
	query, args, err := sq.
		Select(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
		).
		From("trips").
		Where(sq.Eq{"id": tripID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build select trip query: %w", err)
	}

	executor := r.transactionManager.Executor(ctx)

	var trip model.Trip

	err = executor.QueryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, model.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("select trip: %w", err)
	}

	return trip, nil
}
