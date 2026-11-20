package trip

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

func (r *TripRepository) ListPositions(
	ctx context.Context,
	tripID uuid.UUID,
) ([]domaintrip.Position, error) {
	query, args, err := sq.
		Select(
			"id",
			"trip_id",
			"latitude",
			"longitude",
			"recorded_at",
		).
		From("trip_positions").
		Where(sq.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list positions query: %w", err)
	}

	rows, err := r.transactionManager.Executor(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}
	defer rows.Close()

	positions := make([]domaintrip.Position, 0)
	for rows.Next() {
		var p domaintrip.Position
		if err := rows.Scan(
			&p.ID,
			&p.TripID,
			&p.Coordinates.Latitude,
			&p.Coordinates.Longitude,
			&p.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate positions: %w", err)
	}

	return positions, nil
}
