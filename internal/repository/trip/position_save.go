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

func (r *TripRepository) GetStatusForUpdate(
	ctx context.Context,
	tripID uuid.UUID,
) (domaintrip.TripStatus, error) {
	query, args, err := sq.
		Select("status").
		From("trips").
		Where(sq.Eq{"id": tripID}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return "", fmt.Errorf("build select trip status query: %w", err)
	}

	var status domaintrip.TripStatus
	err = r.transactionManager.Executor(ctx).
		QueryRow(ctx, query, args...).
		Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domaintrip.ErrTripNotFound
	}
	if err != nil {
		return "", fmt.Errorf("select trip status for update: %w", err)
	}

	return status, nil
}
