package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *TripRepository) Finish(
	ctx context.Context,
	tripID uuid.UUID,
	finishedAt time.Time,
) (domaintrip.Trip, error) {
	query, args, err := sq.
		Update("trips").
		Set("status", domaintrip.CompletedStatus).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{
			"id":     tripID,
			"status": domaintrip.ActiveStatus,
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
		return domaintrip.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}

	executor := r.transactionManager.Executor(ctx)

	trip, err := scanTrip(executor.QueryRow(ctx, query, args...))

	if err == nil {
		return trip, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return domaintrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	existingTrip, err := r.GetByID(ctx, tripID)
	if errors.Is(err, domaintrip.ErrTripNotFound) {
		return domaintrip.Trip{}, domaintrip.ErrTripNotFound
	}
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("get trip after unsuccessful finish: %w", err)
	}

	if existingTrip.Status == domaintrip.CompletedStatus {
		return domaintrip.Trip{}, domaintrip.ErrTripCompleted
	}

	return domaintrip.Trip{}, fmt.Errorf("finish trip: unexpected current status %q", existingTrip.Status)
}
