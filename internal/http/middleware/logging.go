package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "<unmatched>"
			}

			attrs := []any{
				"method", r.Method,
				"route", route,
				"status", status,
				"duration_ms", time.Since(start).Milliseconds(),
			}

			span := trace.SpanContextFromContext(r.Context())
			if span.IsValid() {
				attrs = append(attrs, "trace_id", span.TraceID().String(), "span_id", span.SpanID().String())
			}

			if status >= http.StatusInternalServerError {
				logger.ErrorContext(r.Context(), "request handled", attrs...)
			} else {
				logger.InfoContext(r.Context(), "request handled", attrs...)
			}
		})
	}
}
