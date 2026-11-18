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
	assert.True(t, lastPositionAt.Equal(freshAt))

	loaded, err := repo.GetByID(ctx, trip.ID)
	require.NoError(t, err)
	require.NotNil(t, loaded.LastPositionAt)
	assert.True(t, loaded.LastPositionAt.Equal(freshAt))

	finished, err := repo.Finish(ctx, trip.ID, time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, finished.LastPositionAt)
	assert.True(t, finished.LastPositionAt.Equal(freshAt))

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

func TestListPositionsIntegration(t *testing.T) {
	ctx, pool, repo := setupRepository(t)
	trip := newTestTrip(uuid.New())
	cleanupTrip(t, pool, trip.ID)
	require.NoError(t, repo.Create(ctx, trip))

	empty, err := repo.ListPositions(ctx, trip.ID)
	require.NoError(t, err)
	require.NotNil(t, empty)
	assert.Empty(t, empty)

	later := time.Now().UTC().Truncate(time.Microsecond)
	earlier := later.Add(-time.Minute)

	first, err := repo.InsertPosition(ctx, domaintrip.Position{
		TripID:      trip.ID,
		Coordinates: domaintrip.Point{Latitude: 1, Longitude: 1},
		RecordedAt:  later,
	})
	require.NoError(t, err)

	second, err := repo.InsertPosition(ctx, domaintrip.Position{
		TripID:      trip.ID,
		Coordinates: domaintrip.Point{Latitude: 2, Longitude: 2},
		RecordedAt:  earlier,
	})
	require.NoError(t, err)

	third, err := repo.InsertPosition(ctx, domaintrip.Position{
		TripID:      trip.ID,
		Coordinates: domaintrip.Point{Latitude: 3, Longitude: 3},
		RecordedAt:  later,
	})
	require.NoError(t, err)

	got, err := repo.ListPositions(ctx, trip.ID)
	require.NoError(t, err)
	require.Len(t, got, 3)

	want := []domaintrip.Position{second, first, third}
	for i := range want {
		assert.Equal(t, want[i].ID, got[i].ID, "position %d: order", i)
		assert.Equal(t, want[i].TripID, got[i].TripID)
		assert.Equal(t, want[i].Coordinates, got[i].Coordinates)
		assert.True(t, got[i].RecordedAt.Equal(want[i].RecordedAt), "position %d: recorded_at", i)
	}
}
