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
	"go.uber.org/mock/gomock"

	"github.com/exf1kzz/go-labs-avito/internal/worker/positions"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestScanOnce(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleRepository(ctrl)
	queue := NewMockjobQueue(ctrl)
	first, second := uuid.New(), uuid.New()

	repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute, uint64(10)).Return([]uuid.UUID{first, second}, nil)

	gomock.InOrder(
		queue.EXPECT().TrySubmit(first).Return(true),
		queue.EXPECT().TrySubmit(second).Return(false),
	)

	scanner := positions.New(repo, queue, time.Minute, time.Second, 10, testLogger())
	require.NoError(t, scanner.ScanOnce(context.Background()))
}

func TestScanOnceRepositoryError(t *testing.T) {
	repo := NewMockstaleRepository(gomock.NewController(t))
	queue := NewMockjobQueue(gomock.NewController(t))

	repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute, uint64(10)).Return(nil, assert.AnError)

	scanner := positions.New(repo, queue, time.Minute, time.Second, 10, testLogger())
	assert.ErrorIs(t, scanner.ScanOnce(context.Background()), assert.AnError)
}

func TestRunScansOnTickAndStops(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleRepository(ctrl)
	queue := NewMockjobQueue(ctrl)
	id := uuid.New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo.EXPECT().ListStaleActive(gomock.Any(), time.Minute, uint64(10)).Return([]uuid.UUID{id}, nil)
	queue.EXPECT().TrySubmit(id).DoAndReturn(func(uuid.UUID) bool {
		cancel()
		return true
	})

	scanner := positions.New(repo, queue, time.Minute, time.Second, 10, testLogger())
	ticks := make(chan time.Time, 1)
	ticks <- time.Now()

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner.Run(ctx, ticks)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("scanner did not stop after cancellation")
	}
}
