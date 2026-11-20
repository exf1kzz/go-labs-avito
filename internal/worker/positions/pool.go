package positions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

type SubmitResult string

const (
	SubmitQueued    SubmitResult = "queued"
	SubmitDuplicate SubmitResult = "duplicate"
	SubmitDropped   SubmitResult = "dropped"
)

type PoolConfig struct {
	Workers        int
	QueueSize      int
	StaleAfter     time.Duration
	QueryTimeout   time.Duration
	RequestTimeout time.Duration
}

type job struct {
	id                uuid.UUID
	ctx               context.Context
	cancel            context.CancelFunc
	checkSpan         trace.SpanContext
	positionCancelled bool
}

type Pool struct {
	mu      sync.Mutex
	jobs    chan *job
	pending map[uuid.UUID]*job
	closed  bool
	wg      sync.WaitGroup

	queueSize     metric.Int64Gauge
	workersActive metric.Int64Gauge
	active        int64

	requests metric.Int64Counter
	duration metric.Float64Histogram

	repository tripReader
	push       pushSender
	cfg        PoolConfig
	now        func() time.Time
	logger     *slog.Logger
	tracer     trace.Tracer
}

func NewPool(
	repo tripReader,
	push pushSender,
	cfg PoolConfig,
	now func() time.Time,
	logger *slog.Logger,
	meter metric.Meter,
	tracer trace.Tracer,
) (*Pool, error) {
	if cfg.Workers <= 0 {
		return nil, fmt.Errorf("position workers must be positive")
	}

	if cfg.QueueSize <= 0 {
		return nil, fmt.Errorf("position queue size must be positive")
	}

	if cfg.StaleAfter <= 0 {
		return nil, fmt.Errorf("position stale threshold must be positive")
	}

	if cfg.QueryTimeout <= 0 {
		return nil, fmt.Errorf("position query timeout must be positive")
	}

	if cfg.RequestTimeout <= 0 {
		return nil, fmt.Errorf("position request timeout must be positive")
	}

	if now == nil {
		return nil, fmt.Errorf("position clock must not be nil")
	}

	if logger == nil {
		return nil, fmt.Errorf("position logger must not be nil")
	}

	queueSize, err := meter.Int64Gauge(
		"tripgo.position_requests.queue.size",
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create position queue gauge: %w", err)
	}

	workersActive, err := meter.Int64Gauge(
		"tripgo.position_requests.workers.active",
		metric.WithUnit("{worker}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create active position workers gauge: %w", err)
	}

	requests, err := meter.Int64Counter("tripgo.position_requests")
	if err != nil {
		return nil, fmt.Errorf("create position requests counter: %w", err)
	}

	duration, err := meter.Float64Histogram(
		"tripgo.position_requests.duration",
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create position requests duration histogram: %w", err)
	}

	p := &Pool{
		jobs:          make(chan *job, cfg.QueueSize),
		pending:       make(map[uuid.UUID]*job),
		repository:    repo,
		push:          push,
		cfg:           cfg,
		now:           now,
		logger:        logger,
		tracer:        tracer,
		queueSize:     queueSize,
		workersActive: workersActive,
		requests:      requests,
		duration:      duration,
	}

	p.wg.Add(cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		go p.runWorker()
	}

	logger.Info("position workers started",
		"workers", cfg.Workers,
		"queue_size", cfg.QueueSize,
	)

	return p, nil
}

func (p *Pool) TrySubmit(ctx context.Context, id uuid.UUID) SubmitResult {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return SubmitDropped
	}
	if _, exists := p.pending[id]; exists {
		return SubmitDuplicate
	}

	jobBase := trace.ContextWithSpanContext(
		context.WithoutCancel(ctx), trace.SpanContext{},
	)
	jobCtx, cancel := context.WithCancel(jobBase)
	j := &job{
		id:        id,
		ctx:       jobCtx,
		cancel:    cancel,
		checkSpan: trace.SpanContextFromContext(ctx),
	}

	select {
	case p.jobs <- j:
		p.pending[id] = j
		p.queueSize.Record(jobCtx, int64(len(p.jobs)))
		return SubmitQueued
	default:
		cancel()
		return SubmitDropped
	}
}

func (p *Pool) Cancel(id uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if j, exists := p.pending[id]; exists {
		j.positionCancelled = true
		delete(p.pending, id)
		j.cancel()
	}
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.jobs)
}

func (p *Pool) Abort() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, j := range p.pending {
		j.cancel()
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}

func (p *Pool) runWorker() {
	defer p.wg.Done()

	for j := range p.jobs {
		started := time.Now()

		var spanOptions []trace.SpanStartOption
		if j.checkSpan.IsValid() {
			spanOptions = append(spanOptions,
				trace.WithLinks(trace.Link{SpanContext: j.checkSpan}),
			)
		}
		jobCtx, span := p.tracer.Start(j.ctx, "positions.request", spanOptions...)
		span.SetAttributes(attribute.String("trip.id", j.id.String()))

		p.mu.Lock()
		p.queueSize.Record(j.ctx, int64(len(p.jobs)))
		current := p.pending[j.id]
		positionCancelled := j.positionCancelled
		shouldProcess := current == j && j.ctx.Err() == nil
		if shouldProcess {
			p.active++
			p.workersActive.Record(j.ctx, p.active)
		}
		p.mu.Unlock()

		if shouldProcess {
			result := p.process(jobCtx, j)
			if result != "" {
				p.recordResult(jobCtx, result, time.Since(started))
			}

			p.mu.Lock()
			p.active--
			p.workersActive.Record(j.ctx, p.active)
			p.mu.Unlock()
		} else if positionCancelled {
			p.recordResult(jobCtx, cancelled, time.Since(started))
		}

		span.End()
		j.cancel()

		p.mu.Lock()
		if p.pending[j.id] == j {
			delete(p.pending, j.id)
		}
		p.mu.Unlock()
	}
}

func (p *Pool) recordResult(ctx context.Context, result string, elapsed time.Duration) {
	attrs := metric.WithAttributes(attribute.String("result", result))
	metricCtx := context.WithoutCancel(ctx)
	p.requests.Add(metricCtx, 1, attrs)
	p.duration.Record(metricCtx, elapsed.Seconds(), attrs)
}

func (p *Pool) cancelResult(j *job) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if j.positionCancelled {
		return cancelled
	}
	return ""
}

const cancelled = "cancelled"

func (p *Pool) process(ctx context.Context, j *job) string {
	queryCtx, cancel := context.WithTimeout(ctx, p.cfg.QueryTimeout)
	trip, err := p.repository.GetByID(queryCtx, j.id)
	cancel()

	if err != nil {
		if ctx.Err() != nil {
			return p.cancelResult(j)
		}
		if errors.Is(err, domaintrip.ErrTripNotFound) {
			return ""
		}

		p.logger.Error("load trip for position request",
			"trip_id", j.id,
			"error", err,
		)
		return "error"
	}

	if trip.Status != domaintrip.ActiveStatus {
		return ""
	}

	lastActivity := trip.StartedAt
	if trip.LastPositionAt != nil {
		lastActivity = *trip.LastPositionAt
	}
	if p.now().Sub(lastActivity) <= p.cfg.StaleAfter {
		return ""
	}

	if ctx.Err() != nil {
		return p.cancelResult(j)
	}

	pushCtx, cancel := context.WithTimeout(ctx, p.cfg.RequestTimeout)
	err = p.push.RequestPosition(pushCtx, trip.DriverID, trip.ID)
	cancel()
	if err != nil {
		if ctx.Err() != nil {
			return p.cancelResult(j)
		}
		p.logger.Warn("request trip position",
			"trip_id", j.id,
			"error", err,
		)
		return "error"
	}
	return "success"
}
