package positions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Scanner struct {
	repository   staleRepository
	queue        jobQueue
	staleAfter   time.Duration
	queryTimeout time.Duration
	logger       *slog.Logger
	staleChecks  metric.Int64Counter
	tracer       trace.Tracer
}

func New(
	repository staleRepository,
	queue jobQueue,
	staleAfter time.Duration,
	queryTimeout time.Duration,
	logger *slog.Logger,
	staleChecks metric.Int64Counter,
	tracer trace.Tracer,
) *Scanner {
	return &Scanner{
		repository:   repository,
		queue:        queue,
		staleAfter:   staleAfter,
		queryTimeout: queryTimeout,
		logger:       logger,
		staleChecks:  staleChecks,
		tracer:       tracer,
	}
}

func (s *Scanner) Run(ctx context.Context, ticker Ticker) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticker.C():
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
	ctx, span := s.tracer.Start(ctx, "positions.scan")
	defer span.End()

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	ids, err := s.repository.ListStaleActive(queryCtx, s.staleAfter)
	if err != nil {
		return fmt.Errorf("list stale active trips: %w", err)
	}

	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}

		result := s.queue.TrySubmit(ctx, id)
		s.staleChecks.Add(ctx, 1,
			metric.WithAttributes(attribute.String("result", string(result))),
		)
		if result == SubmitDropped {
			s.logger.Warn("position request queue is full", "trip_id", id)
		}
	}

	return nil
}
