//go:build integration

package trip_test

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListStaleActiveIntegration(t *testing.T) {
	ctx, pool, repo := setupRepository(t)
	now := time.Now().UTC()

	create := func(startedAt time.Time) uuid.UUID {
		t.Helper()
		trip := newTestTrip(uuid.New())
		trip.StartedAt = startedAt
		cleanupTrip(t, pool, trip.ID)
		require.NoError(t, repo.Create(ctx, trip))
		return trip.ID
	}

	oldNoPosition := create(now.Add(-40 * time.Minute))

	oldWithPosition := create(now.Add(-50 * time.Minute))
	require.NoError(t, repo.UpdateLastPositionAt(
		ctx, oldWithPosition, now.Add(-20*time.Minute),
	))

	freshPosition := create(now.Add(-60 * time.Minute))
	require.NoError(t, repo.UpdateLastPositionAt(
		ctx, freshPosition, now.Add(-2*time.Minute),
	))

	freshStart := create(now.Add(-2 * time.Minute))

	completed := create(now.Add(-70 * time.Minute))
	_, err := repo.Finish(ctx, completed, now)
	require.NoError(t, err)

	ids, err := repo.ListStaleActive(ctx, 10*time.Minute)
	require.NoError(t, err)

	require.Contains(t, ids, oldNoPosition)
	require.Contains(t, ids, oldWithPosition)
	assert.Less(t, slices.Index(ids, oldNoPosition), slices.Index(ids, oldWithPosition))

	assert.NotContains(t, ids, freshPosition)
	assert.NotContains(t, ids, freshStart)
	assert.NotContains(t, ids, completed)
}
