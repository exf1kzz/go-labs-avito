package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

type queryMetricsTracer struct {
	*otelpgx.Tracer
	duration metric.Float64Histogram
}

type queryMetricsKey struct{}

type queryMetricsState struct {
	started    time.Time
	operation  string
	collection string
}

func newQueryMetricsTracer(
	tracerProvider trace.TracerProvider,
	meterProvider metric.MeterProvider,
) (*queryMetricsTracer, error) {
	duration, err := meterProvider.Meter("trip-service/db").Float64Histogram(
		"db.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("create database duration histogram: %w", err)
	}

	return &queryMetricsTracer{
		Tracer: otelpgx.NewTracer(
			otelpgx.WithTracerProvider(tracerProvider),
			otelpgx.WithMeterProvider(metricnoop.NewMeterProvider()),
		),
		duration: duration,
	}, nil
}

func (t *queryMetricsTracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)
	collection := queryCollection(data.SQL)
	trace.SpanFromContext(ctx).SetAttributes(attribute.String("db.collection.name", collection))
	return context.WithValue(ctx, queryMetricsKey{}, queryMetricsState{
		started:    time.Now(),
		operation:  otelpgx.SQLOperationName(ctx, data.SQL),
		collection: collection,
	})
}

func (t *queryMetricsTracer) TraceQueryEnd(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	if state, ok := ctx.Value(queryMetricsKey{}).(queryMetricsState); ok {
		errorType := "none"
		if data.Err != nil {
			var pgErr *pgconn.PgError
			if errors.As(data.Err, &pgErr) {
				errorType = pgErr.Code
			} else {
				errorType = fmt.Sprintf("%T", data.Err)
			}

			trace.SpanFromContext(ctx).SetAttributes(attribute.String("error.type", errorType))
		}

		t.duration.Record(
			ctx,
			time.Since(state.started).Seconds(),
			metric.WithAttributes(
				attribute.String("db.operation.name", state.operation),
				attribute.String("db.collection.name", state.collection),
				attribute.String("error.type", errorType),
			),
		)
	}

	t.Tracer.TraceQueryEnd(ctx, conn, data)
}

func queryCollection(sql string) string {
	fields := strings.Fields(sql)
	if len(fields) == 0 {
		return "unknown"
	}

	clean := func(value string) string {
		return strings.Trim(value, `";(),`)
	}

	switch strings.ToUpper(fields[0]) {
	case "INSERT":
		if len(fields) > 2 && strings.EqualFold(fields[1], "INTO") {
			return clean(fields[2])
		}
	case "UPDATE":
		if len(fields) > 1 {
			return clean(fields[1])
		}
	case "SELECT", "DELETE":
		for i := 1; i+1 < len(fields); i++ {
			if strings.EqualFold(fields[i], "FROM") {
				return clean(fields[i+1])
			}
		}
	}

	return "unknown"
}
