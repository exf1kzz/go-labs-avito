package trip

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

func (r *TripRepository) InsertPosition(
	ctx context.Context,
	position domaintrip.Position,
) (domaintrip.Position, error) {
	query, args, err := sq.
		Insert("trip_positions").
		Columns("trip_id", "latitude", "longitude", "recorded_at").
		Values(
			position.TripID,
			position.Coordinates.Latitude,
			position.Coordinates.Longitude,
			position.RecordedAt,
		).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domaintrip.Position{}, fmt.Errorf("build insert position query: %w", err)
	}

	err = r.transactionManager.Executor(ctx).
		QueryRow(ctx, query, args...).
		Scan(&position.ID)
	if err != nil {
		return domaintrip.Position{}, fmt.Errorf("insert position: %w", err)
	}

	return position, nil
}
