package positions_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	"github.com/exf1kzz/go-labs-avito/internal/worker/positions"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testStaleChecks(t *testing.T) metric.Int64Counter {
	t.Helper()
	counter, err := metricnoop.NewMeterProvider().
		Meter("test").
		Int64Counter("tripgo.stale_checks")
	require.NoError(t, err)
	return counter
}

type fakeTicker struct {
	ch chan time.Time
}

func (t *fakeTicker) C() <-chan time.Time { return t.ch }

func (t *fakeTicker) Stop() {}

func TestScanOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleRepository(ctrl)
	queue := NewMockjobQueue(ctrl)
	first, second := uuid.New(), uuid.New()

	repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute).Return([]uuid.UUID{first, second}, nil)

	gomock.InOrder(
		queue.EXPECT().TrySubmit(gomock.Any(), first).Return(positions.SubmitQueued),
		queue.EXPECT().TrySubmit(gomock.Any(), second).Return(positions.SubmitDropped),
	)

	scanner := positions.New(
		repo, queue, time.Minute, time.Second, testLogger(),
		testStaleChecks(t), tracenoop.NewTracerProvider().Tracer("test"),
	)
	require.NoError(t, scanner.ScanOnce(context.Background()))
}

func TestScanOnceRepositoryError(t *testing.T) {
	repo := NewMockstaleRepository(gomock.NewController(t))
	queue := NewMockjobQueue(gomock.NewController(t))

	repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute).Return(nil, assert.AnError)

	scanner := positions.New(
		repo, queue, time.Minute, time.Second, testLogger(),
		testStaleChecks(t), tracenoop.NewTracerProvider().Tracer("test"),
	)
	assert.ErrorIs(t, scanner.ScanOnce(context.Background()), assert.AnError)
}

func TestRunScansOnTickAndStops(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleRepository(ctrl)
	queue := NewMockjobQueue(ctrl)
	id := uuid.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute).Return([]uuid.UUID{id}, nil)
	queue.EXPECT().TrySubmit(gomock.Any(), id).DoAndReturn(
		func(context.Context, uuid.UUID) positions.SubmitResult {
			cancel()
			return positions.SubmitQueued
		},
	)

	scanner := positions.New(
		repo, queue, time.Minute, time.Second, testLogger(),
		testStaleChecks(t), tracenoop.NewTracerProvider().Tracer("test"),
	)
	ticker := &fakeTicker{ch: make(chan time.Time, 1)}
	ticker.ch <- time.Now()

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner.Run(ctx, ticker)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("scanner did not stop after cancellation")
	}
}

func TestRunChecksSameTripOnNextTick(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleRepository(ctrl)
	queue := NewMockjobQueue(ctrl)
	id := uuid.New()
	firstQueued := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gomock.InOrder(
		repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute).Return([]uuid.UUID{id}, nil),
		queue.EXPECT().TrySubmit(gomock.Any(), id).
			DoAndReturn(func(context.Context, uuid.UUID) positions.SubmitResult {
				close(firstQueued)
				return positions.SubmitQueued
			}),
		repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute).Return([]uuid.UUID{id}, nil),
		queue.EXPECT().TrySubmit(gomock.Any(), id).
			DoAndReturn(func(context.Context, uuid.UUID) positions.SubmitResult {
				cancel()
				return positions.SubmitDuplicate
			}),
	)

	scanner := positions.New(
		repo, queue, time.Minute, time.Second, testLogger(),
		testStaleChecks(t), tracenoop.NewTracerProvider().Tracer("test"),
	)
	ticker := &fakeTicker{ch: make(chan time.Time, 1)}
	done := make(chan struct{})

	go func() {
		defer close(done)
		scanner.Run(ctx, ticker)
	}()

	ticker.ch <- time.Now()
	waitForSignal(t, firstQueued)
	ticker.ch <- time.Now()
	waitForSignal(t, done)
}
