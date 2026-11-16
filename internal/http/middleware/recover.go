package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
)

type panicSpanContextKey struct{}

type panicSpanContext struct {
	value trace.SpanContext
}

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spanContext := &panicSpanContext{}
			r = r.WithContext(context.WithValue(r.Context(), panicSpanContextKey{}, spanContext))
			start := time.Now()

			defer func() {
				if value := recover(); value != nil {
					route := chi.RouteContext(r.Context()).RoutePattern()
					if route == "" {
						route = unmatchedRoute
					}

					logCtx := r.Context()
					if spanContext.value.IsValid() {
						logCtx = trace.ContextWithSpanContext(logCtx, spanContext.value)
					}

					logger.ErrorContext(logCtx, "request handled",
						"method", r.Method,
						"route", route,
						"status", http.StatusInternalServerError,
						"duration_ms", time.Since(start).Milliseconds(),
						"panic", fmt.Sprint(value),
						"stack", string(debug.Stack()),
					)

					domainerror.WriteInternal(w, r)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
