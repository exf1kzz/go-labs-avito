package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const unmatchedRoute = "<unmatched>"

func ActiveRequests(router *chi.Mux, counter metric.Int64UpDownCounter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := router.Find(chi.NewRouteContext(), r.Method, r.URL.Path)
			if route == "" {
				route = unmatchedRoute
			}

			attrs := metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
			)

			counter.Add(r.Context(), 1, attrs)
			defer counter.Add(r.Context(), -1, attrs)

			next.ServeHTTP(w, r)
		})
	}
}
