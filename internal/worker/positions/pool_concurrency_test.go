package positions_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/worker/positions"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestPoolRunsConfiguredWorkers(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripReader(ctrl)
	push := NewMockpushSender(ctrl)

	now := time.Now()
	driverID := uuid.New()
	started := make(chan struct{}, 3)
	release := make(chan struct{})

	repo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Times(3).
		DoAndReturn(func(_ context.Context, id uuid.UUID) (domaintrip.Trip, error) {
			return domaintrip.Trip{
				ID:        id,
				DriverID:  driverID,
				Status:    domaintrip.ActiveStatus,
				StartedAt: now.Add(-2 * time.Minute),
			}, nil
		})

	push.EXPECT().RequestPosition(gomock.Any(), driverID, gomock.Any()).Times(3).
		DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
			started <- struct{}{}
			<-release
			return nil
		})

	cfg := testPoolConfig()
	cfg.Workers = 3
	cfg.QueueSize = 3

	pool, err := positions.NewPool(
		repo, push, cfg, func() time.Time { return now },
		testLogger(),
		metricnoop.NewMeterProvider().Meter("test"),
		tracenoop.NewTracerProvider().Tracer("test"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		pool.Abort()
		pool.Close()
		pool.Wait()
	})

	for range 3 {
		require.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), uuid.New()))
	}

	for range 3 {
		waitForSignal(t, started)
	}

	close(release)
	pool.Close()
	pool.Wait()
}

func TestPoolContinuesAfterPushError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripReader(ctrl)
	push := NewMockpushSender(ctrl)

	now := time.Now()
	first, second, driverID := uuid.New(), uuid.New(), uuid.New()
	firstDone := make(chan struct{})

	firstTrip := domaintrip.Trip{
		ID: first, DriverID: driverID,
		Status: domaintrip.ActiveStatus, StartedAt: now.Add(-2 * time.Minute),
	}
	secondTrip := domaintrip.Trip{
		ID: second, DriverID: driverID,
		Status: domaintrip.ActiveStatus, StartedAt: now.Add(-2 * time.Minute),
	}

	gomock.InOrder(
		repo.EXPECT().GetByID(gomock.Any(), first).Return(firstTrip, nil),
		push.EXPECT().RequestPosition(gomock.Any(), driverID, first).
			DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
				close(firstDone)
				return assert.AnError
			}),
		repo.EXPECT().GetByID(gomock.Any(), second).Return(secondTrip, nil),
		push.EXPECT().RequestPosition(gomock.Any(), driverID, second).Return(nil),
	)

	pool, err := positions.NewPool(
		repo, push, testPoolConfig(), func() time.Time { return now },
		testLogger(),
		metricnoop.NewMeterProvider().Meter("test"),
		tracenoop.NewTracerProvider().Tracer("test"),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Abort()
		pool.Close()
		pool.Wait()
	})

	require.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), first))
	waitForSignal(t, firstDone)
	require.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), second))

	pool.Close()
	pool.Wait()
}
