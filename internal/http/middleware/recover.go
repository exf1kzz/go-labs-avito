package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			defer func() {
				if value := recover(); value != nil {
					route := chi.RouteContext(r.Context()).RoutePattern()
					if route == "" {
						route = unmatchedRoute
					}

					logger.ErrorContext(r.Context(), "request handled",
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
