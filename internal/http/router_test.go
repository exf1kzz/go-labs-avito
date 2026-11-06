package transport_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	transport "github.com/exf1kzz/go-labs-avito/internal/http"
)

func TestInvalidTripID(t *testing.T) {
	cnt, err := metricnoop.NewMeterProvider().Meter("test").Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		t.Fatal(err)
	}

	router := transport.New(
		api.Unimplemented{},
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		noop.NewTracerProvider(),
		metricnoop.NewMeterProvider(),
		cnt,
	)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}

	var problem api.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode Problem: %v", err)
	}
	if problem.Code != "invalid_request" {
		t.Errorf("code = %q, want invalid_request", problem.Code)
	}
}
