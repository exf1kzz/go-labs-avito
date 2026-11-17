//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/infra/postgres/transaction"
	triprepo "github.com/exf1kzz/go-labs-avito/internal/repository/trip"
)

func setupRepository(t *testing.T) (context.Context, *pgxpool.Pool, *triprepo.TripRepository) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	manager := transaction.NewTransactionManager(pool)
	return ctx, pool, triprepo.NewTripRepository(manager)
}

func newTestTrip(driverID uuid.UUID) domaintrip.Trip {
	return domaintrip.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   driverID,
		StartPoint: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domaintrip.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
		Status:     domaintrip.ActiveStatus,
		StartedAt:  time.Now().UTC(),
	}
}

func cleanupTrip(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", id); err != nil {
			t.Errorf("cleanup trip %s: %v", id, err)
		}
	})
}

func TestTripLifecycleIntegration(t *testing.T) {
	ctx, pool, repo := setupRepository(t)
	trip := newTestTrip(uuid.New())
	cleanupTrip(t, pool, trip.ID)

	if err := repo.Create(ctx, trip); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.AddStatusHistory(
		ctx, trip.ID, nil, domaintrip.ActiveStatus, nil, trip.StartedAt,
	); err != nil {
		t.Fatalf("initial history: %v", err)
	}

	loaded, err := repo.GetByID(ctx, trip.ID)
	if err != nil || loaded.ID != trip.ID || loaded.Status != domaintrip.ActiveStatus {
		t.Fatalf("loaded trip = %+v, error = %v", loaded, err)
	}

	at := time.Now().UTC()
	finished, err := repo.Finish(ctx, trip.ID, at)
	if err != nil || finished.Status != domaintrip.CompletedStatus ||
		finished.FinishedAt == nil {
		t.Fatalf("finished trip = %+v, error = %v", finished, err)
	}

	from := domaintrip.ActiveStatus
	if err := repo.AddStatusHistory(
		ctx, trip.ID, &from, domaintrip.CompletedStatus, nil, at,
	); err != nil {
		t.Fatalf("completion history: %v", err)
	}

	if _, err := repo.Finish(ctx, trip.ID, at); !errors.Is(err, domaintrip.ErrTripCompleted) {
		t.Errorf("second finish error = %v, want ErrTripCompleted", err)
	}

	var historyCount int
	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM trip_status_history WHERE trip_id = $1",
		trip.ID,
	).Scan(&historyCount); err != nil || historyCount != 2 {
		t.Errorf("history count = %d, error = %v, want 2", historyCount, err)
	}
}

func TestDriverBusyIntegration(t *testing.T) {
	ctx, pool, repo := setupRepository(t)
	driverID := uuid.New()
	first := newTestTrip(driverID)
	second := newTestTrip(driverID)
	cleanupTrip(t, pool, first.ID)
	cleanupTrip(t, pool, second.ID)

	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("create first trip: %v", err)
	}
	if err := repo.Create(ctx, second); !errors.Is(err, domaintrip.ErrDriverBusy) {
		t.Errorf("create second trip error = %v, want ErrDriverBusy", err)
	}
}
