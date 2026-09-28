package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/exf1kzz/go-labs-avito/internal/model"
	"github.com/exf1kzz/go-labs-avito/internal/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type TripRepository struct {
	transactionManager *postgres.TransactionManager
}

func NewTripRepository(transactionManager *postgres.TransactionManager) *TripRepository {
	return &TripRepository{
		transactionManager: transactionManager,
	}
}

func (r *TripRepository) Create(
	ctx context.Context,
	trip model.Trip,
) error {
	query, args, err := sq.
		Insert("trips").
		Columns(
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
			"finished_at").
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartPoint.Latitude,
			trip.StartPoint.Longitude,
			trip.EndPoint.Latitude,
			trip.EndPoint.Longitude,
			trip.Price,
			trip.Status,
			trip.StartedAt,
			trip.FinishedAt).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("build insert trip query: %w", err)
	}

	executor := r.transactionManager.Executor(ctx)

	_, err = executor.Exec(ctx, query, args...)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
		return model.ErrDriverBusy
	}

	return fmt.Errorf("insert trip: %w", err)
}

func (r *TripRepository) AddStatusHistory(
	ctx context.Context,
	tripID uuid.UUID,
	fromStatus *model.TripStatus,
	toStatus model.TripStatus,
	reason *string,
	changedAt time.Time,
) error {
	query, args, err := sq.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
			"changed_at",
		).
		Values(
			tripID,
			fromStatus,
			toStatus,
			reason,
			changedAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip status history query: %w", err)
	}

	executor := r.transactionManager.Executor(ctx)

	if _, err := executor.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert trip status history: %w", err)
	}

	return nil
}
