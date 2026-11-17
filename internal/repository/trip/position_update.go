package trip

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *TripRepository) UpdateLastPositionAt(
	ctx context.Context,
	tripID uuid.UUID,
	recordedAt time.Time,
) error {
	const query = `
		UPDATE trips
		SET last_position_at = GREATEST(last_position_at, $2),
		    updated_at = now()
		WHERE id = $1
	`

	_, err := r.transactionManager.Executor(ctx).
		Exec(ctx, query, tripID, recordedAt)
	if err != nil {
		return fmt.Errorf("update last position at: %w", err)
	}

	return nil
}
