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

func TestCreateRollbackIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
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

	txManager := transaction.NewTransactionManager(pool)
	repo := triprepo.NewTripRepository(txManager)
	trip := domaintrip.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domaintrip.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
		Status:     domaintrip.ActiveStatus,
		StartedAt:  time.Now().UTC(),
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		if _, err := pool.Exec(cleanupCtx, "DELETE FROM trips WHERE id = $1", trip.ID); err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})

	rollbackErr := errors.New("force rollback")
	err = txManager.Do(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, trip); err != nil {
			return err
		}

		if err := repo.AddStatusHistory(
			txCtx, trip.ID, nil, domaintrip.ActiveStatus, nil, trip.StartedAt,
		); err != nil {
			return err
		}

		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("transaction error = %v, want rollback error", err)
	}

	for _, table := range []string{"trips", "trip_status_history"} {
		query := "SELECT count(*) FROM " + table + " WHERE "
		if table == "trips" {
			query += "id = $1"
		} else {
			query += "trip_id = $1"
		}

		var count int
		if err := pool.QueryRow(ctx, query, trip.ID).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s contains %d rows after rollback, want 0", table, count)
		}
	}
}
