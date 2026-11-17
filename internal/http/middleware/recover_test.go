package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	httpmiddleware "github.com/exf1kzz/go-labs-avito/internal/http/middleware"
)

func TestRecover(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	router := chi.NewRouter()
	router.Use(httpmiddleware.Recover(logger))
	router.Use(httpmiddleware.Logging(logger))
	router.Get("/panic", func(http.ResponseWriter, *http.Request) {
		panic("secret panic detail")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	var problem api.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
	assert.Equal(t, "internal_error", problem.Code)
	assert.NotContains(t, rec.Body.String(), "secret panic detail")

	var entry struct {
		Level  string `json:"level"`
		Msg    string `json:"msg"`
		Status int    `json:"status"`
		Panic  string `json:"panic"`
		Stack  string `json:"stack"`
	}
	require.NoError(t, json.Unmarshal(logs.Bytes(), &entry), logs.String())

	assert.Equal(t, "ERROR", entry.Level)
	assert.Equal(t, "request handled", entry.Msg)
	assert.Equal(t, http.StatusInternalServerError, entry.Status)
	assert.Equal(t, "secret panic detail", entry.Panic)
	assert.NotEmpty(t, entry.Stack)
}

type logSpanCapture struct {
	slog.Handler
	span  trace.SpanContext
	calls int
}

func (h *logSpanCapture) Handle(ctx context.Context, record slog.Record) error {
	h.span = trace.SpanContextFromContext(ctx)
	h.calls++
	return h.Handler.Handle(ctx, record)
}

func TestRecoverPreservesPanicTrace(t *testing.T) {
	capture := &logSpanCapture{
		Handler: slog.NewJSONHandler(io.Discard, nil),
	}
	logger := slog.New(capture)
	provider := sdktrace.NewTracerProvider()
	t.Cleanup(func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	})

	var requestSpan trace.SpanContext

	router := chi.NewRouter()
	router.Use(httpmiddleware.Recover(logger))
	router.Use(httpmiddleware.Tracing(provider, metricnoop.NewMeterProvider()))
	router.Use(httpmiddleware.Logging(logger))
	router.Get("/panic", func(_ http.ResponseWriter, r *http.Request) {
		requestSpan = trace.SpanContextFromContext(r.Context())
		panic("test panic")
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.True(t, requestSpan.IsValid(), "HTTP span was not created")
	require.Equal(t, 1, capture.calls)
	assert.Equal(t, requestSpan.TraceID(), capture.span.TraceID())
	assert.Equal(t, requestSpan.SpanID(), capture.span.SpanID())
}
