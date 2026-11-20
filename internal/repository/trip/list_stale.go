package trip

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

func (r *TripRepository) ListStaleActive(
	ctx context.Context,
	timeout time.Duration,
) ([]uuid.UUID, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("list stale active trips: timeout must be positive")
	}

	query, args, err := sq.
		Select("id").
		From("trips").
		Where(sq.Eq{"status": domaintrip.ActiveStatus}).
		Where(sq.Expr(
			"COALESCE(last_position_at, started_at) < now() - make_interval(secs => ?::double precision)",
			timeout.Seconds(),
		)).
		OrderBy("COALESCE(last_position_at, started_at) ASC", "id ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list stale active trips query: %w", err)
	}

	rows, err := r.transactionManager.Executor(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stale active trips: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan stale active trip: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stale active trips: %w", err)
	}

	return ids, nil
}
