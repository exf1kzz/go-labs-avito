package positions

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Scanner struct {
	repository   staleRepository
	queue        jobQueue
	staleAfter   time.Duration
	queryTimeout time.Duration
	batchSize    uint64
	logger       *slog.Logger
}

func New(
	repository staleRepository,
	queue jobQueue,
	staleAfter time.Duration,
	queryTimeout time.Duration,
	batchSize uint64,
	logger *slog.Logger,
) *Scanner {
	return &Scanner{
		repository:   repository,
		queue:        queue,
		staleAfter:   staleAfter,
		queryTimeout: queryTimeout,
		batchSize:    batchSize,
		logger:       logger,
	}
}

func (s *Scanner) Run(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok {
				return
			}
			if err := s.ScanOnce(ctx); err != nil && ctx.Err() == nil {
				s.logger.Error("scan stale trips", "error", err)
			}
		}
	}
}

func (s *Scanner) ScanOnce(ctx context.Context) error {
	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	ids, err := s.repository.ListStaleActive(queryCtx, s.staleAfter, s.batchSize)
	if err != nil {
		return fmt.Errorf("list stale active trips: %w", err)
	}

	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.queue.TrySubmit(id)
	}

	return nil
}
