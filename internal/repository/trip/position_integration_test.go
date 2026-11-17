//go:build integration

package trip_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/infra/postgres/transaction"
	positioncreate "github.com/exf1kzz/go-labs-avito/internal/usecase/trip/position/create"
)

func TestCreatePositionIntegration(t *testing.T) {
	ctx, pool, repo := setupRepository(t)
	usecase := positioncreate.New(repo, transaction.NewTransactionManager(pool), time.Now)

	trip := newTestTrip(uuid.New())
	cleanupTrip(t, pool, trip.ID)
	require.NoError(t, repo.Create(ctx, trip))

	freshAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	oldAt := freshAt.Add(-time.Minute)

	for _, recordedAt := range []time.Time{freshAt, oldAt} {
		saved, err := usecase.Execute(ctx, domaintrip.Position{
			TripID:      trip.ID,
			Coordinates: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
			RecordedAt:  recordedAt,
		})
		require.NoError(t, err)
		assert.Positive(t, saved.ID)
		assert.Equal(t, recordedAt, saved.RecordedAt)
	}

	var lastPositionAt time.Time
	err := pool.QueryRow(ctx,
		"SELECT last_position_at FROM trips WHERE id = $1", trip.ID,
	).Scan(&lastPositionAt)
	require.NoError(t, err)
	assert.True(t, lastPositionAt.Equal(freshAt), "last_position_at = %s, want %s", lastPositionAt, freshAt)

	_, err = repo.Finish(ctx, trip.ID, time.Now().UTC())
	require.NoError(t, err)

	_, err = usecase.Execute(ctx, domaintrip.Position{
		TripID:      trip.ID,
		Coordinates: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		RecordedAt:  time.Now().UTC(),
	})
	assert.ErrorIs(t, err, domaintrip.ErrTripCompleted)

	var count int
	err = pool.QueryRow(ctx,
		"SELECT count(*) FROM trip_positions WHERE trip_id = $1", trip.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	_, err = usecase.Execute(ctx, domaintrip.Position{
		TripID:      uuid.New(),
		Coordinates: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		RecordedAt:  time.Now().UTC(),
	})
	assert.ErrorIs(t, err, domaintrip.ErrTripNotFound)
}

func TestPositionTransactionRollbackIntegration(t *testing.T) {
	ctx, pool, repo := setupRepository(t)
	txManager := transaction.NewTransactionManager(pool)

	trip := newTestTrip(uuid.New())
	cleanupTrip(t, pool, trip.ID)
	require.NoError(t, repo.Create(ctx, trip))

	err := txManager.Do(ctx, func(txCtx context.Context) error {
		_, err := repo.InsertPosition(txCtx, domaintrip.Position{
			TripID:      trip.ID,
			Coordinates: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
			RecordedAt:  time.Now().UTC(),
		})
		require.NoError(t, err)
		return assert.AnError
	})
	assert.ErrorIs(t, err, assert.AnError)

	var count int
	err = pool.QueryRow(ctx,
		"SELECT count(*) FROM trip_positions WHERE trip_id = $1", trip.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Zero(t, count)
}
