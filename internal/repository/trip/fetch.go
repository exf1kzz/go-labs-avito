package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

func (r *TripRepository) GetByID(
	ctx context.Context,
	tripID uuid.UUID,
) (domaintrip.Trip, error) {
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
		return domaintrip.Trip{}, fmt.Errorf("build select trip query: %w", err)
	}

	executor := r.transactionManager.Executor(ctx)

	trip, err := scanTrip(executor.QueryRow(ctx, query, args...))

	if errors.Is(err, pgx.ErrNoRows) {
		return domaintrip.Trip{}, domaintrip.ErrTripNotFound
	}
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("select trip: %w", err)
	}

	return trip, nil
}
