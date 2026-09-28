package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/exf1kzz/go-labs-avito/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *TripRepository) Finish(
	ctx context.Context,
	tripID uuid.UUID,
	finishedAt time.Time,
) (model.Trip, error) {
	query, args, err := sq.
		Update("trips").
		Set("status", model.CompletedStatus).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{
			"id":     tripID,
			"status": model.ActiveStatus,
		}).
		Suffix(`
			RETURNING
				id,
				user_id,
				driver_id,
				start_latitude,
				start_longitude,
				end_latitude,
				end_longitude,
				price,
				status,
				started_at,
				finished_at
		`).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build finish trip query: %w", err)
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
	if err == nil {
		return trip, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	existingTrip, err := r.GetByID(ctx, tripID)
	if errors.Is(err, model.ErrTripNotFound) {
		return model.Trip{}, model.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip after unsuccessful finish: %w", err)
	}

	if existingTrip.Status == model.CompletedStatus {
		return model.Trip{}, model.ErrTripCompleted
	}

	return model.Trip{}, fmt.Errorf("finish trip: unexpected current status %q", existingTrip.Status)
}
