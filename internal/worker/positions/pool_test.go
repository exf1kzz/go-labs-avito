package positions_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/worker/positions"
)

func testPoolConfig() positions.PoolConfig {
	return positions.PoolConfig{
		Workers:        1,
		QueueSize:      1,
		StaleAfter:     time.Minute,
		QueryTimeout:   5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}
}

func waitForSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker")
	}
}

func TestPoolBoundsDeduplicatesAndDrains(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripReader(ctrl)
	push := NewMockpushSender(ctrl)

	now := time.Now()
	first, second, rejected := uuid.New(), uuid.New(), uuid.New()
	firstDriver, secondDriver := uuid.New(), uuid.New()
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }

	repo.EXPECT().GetByID(gomock.Any(), first).DoAndReturn(
		func(context.Context, uuid.UUID) (domaintrip.Trip, error) {
			close(started)
			<-release
			return domaintrip.Trip{
				ID: first, DriverID: firstDriver,
				Status:    domaintrip.ActiveStatus,
				StartedAt: now.Add(-2 * time.Minute),
			}, nil
		},
	)
	repo.EXPECT().GetByID(gomock.Any(), second).Return(domaintrip.Trip{
		ID: second, DriverID: secondDriver,
		Status:    domaintrip.ActiveStatus,
		StartedAt: now.Add(-2 * time.Minute),
	}, nil)
	push.EXPECT().RequestPosition(gomock.Any(), firstDriver, first).Return(nil)
	push.EXPECT().RequestPosition(gomock.Any(), secondDriver, second).Return(nil)

	pool, err := positions.NewPool(
		repo, push, testPoolConfig(), func() time.Time { return now },
		testLogger(),
		metricnoop.NewMeterProvider().Meter("test"),
		tracenoop.NewTracerProvider().Tracer("test"),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		unblock()
		pool.Abort()
		pool.Close()
		pool.Wait()
	})

	require.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), first))
	waitForSignal(t, started)

	assert.Equal(t, positions.SubmitDuplicate, pool.TrySubmit(context.Background(), first))
	assert.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), second))
	assert.Equal(t, positions.SubmitDropped, pool.TrySubmit(context.Background(), rejected))

	pool.Close()
	assert.Equal(t, positions.SubmitDropped, pool.TrySubmit(context.Background(), uuid.New()))
	unblock()
	pool.Wait()
}

func TestPoolCancelRunningRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripReader(ctrl)
	push := NewMockpushSender(ctrl)

	now := time.Now()
	id, driverID := uuid.New(), uuid.New()
	pushStarted := make(chan struct{})

	repo.EXPECT().GetByID(gomock.Any(), id).Return(domaintrip.Trip{
		ID: id, DriverID: driverID,
		Status:    domaintrip.ActiveStatus,
		StartedAt: now.Add(-2 * time.Minute),
	}, nil)

	push.EXPECT().RequestPosition(gomock.Any(), driverID, id).DoAndReturn(
		func(ctx context.Context, _, _ uuid.UUID) error {
			close(pushStarted)
			<-ctx.Done()
			return ctx.Err()
		},
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

	require.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), id))
	waitForSignal(t, pushStarted)
	pool.Cancel(id)
	pool.Close()
	pool.Wait()
}

func TestPoolCancelQueuedJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripReader(ctrl)
	push := NewMockpushSender(ctrl)

	now := time.Now()
	running, queued, driverID := uuid.New(), uuid.New(), uuid.New()
	pushStarted := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }

	repo.EXPECT().GetByID(gomock.Any(), running).Return(domaintrip.Trip{
		ID: running, DriverID: driverID,
		Status:    domaintrip.ActiveStatus,
		StartedAt: now.Add(-2 * time.Minute),
	}, nil)

	push.EXPECT().RequestPosition(gomock.Any(), driverID, running).DoAndReturn(
		func(context.Context, uuid.UUID, uuid.UUID) error {
			close(pushStarted)
			<-release
			return nil
		},
	)

	pool, err := positions.NewPool(
		repo, push, testPoolConfig(), func() time.Time { return now },
		testLogger(),
		metricnoop.NewMeterProvider().Meter("test"),
		tracenoop.NewTracerProvider().Tracer("test"),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		unblock()
		pool.Abort()
		pool.Close()
		pool.Wait()
	})

	require.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), running))
	waitForSignal(t, pushStarted)
	require.Equal(t, positions.SubmitQueued, pool.TrySubmit(context.Background(), queued))
	pool.Cancel(queued)

	pool.Close()
	unblock()
	pool.Wait()
}
