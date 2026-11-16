package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q", got)
	}

	var problem api.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode Problem: %v", err)
	}
	if problem.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", problem.Code)
	}
	if strings.Contains(rec.Body.String(), "secret panic detail") {
		t.Error("panic detail leaked into HTTP response")
	}

	var entry struct {
		Level  string `json:"level"`
		Msg    string `json:"msg"`
		Status int    `json:"status"`
		Panic  string `json:"panic"`
		Stack  string `json:"stack"`
	}

	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("decode single JSON log entry: %v; logs: %s", err, logs.String())
	}

	if entry.Level != "ERROR" || entry.Msg != "request handled" ||
		entry.Status != http.StatusInternalServerError {
		t.Errorf("unexpected log entry: %+v", entry)
	}

	if entry.Panic != "secret panic detail" || entry.Stack == "" {
		t.Error("panic detail or stack missing from log")
	}
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
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown tracer provider: %v", err)
		}
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

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !requestSpan.IsValid() {
		t.Fatal("HTTP span was not created")
	}
	if capture.calls != 1 {
		t.Fatalf("log entries = %d, want 1", capture.calls)
	}
	if capture.span.TraceID() != requestSpan.TraceID() ||
		capture.span.SpanID() != requestSpan.SpanID() {
		t.Errorf("panic log span = %s/%s, request span = %s/%s",
			capture.span.TraceID(), capture.span.SpanID(),
			requestSpan.TraceID(), requestSpan.SpanID())
	}
}
