package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(
	provider trace.TracerProvider,
	meterProvider metric.MeterProvider,
) func(handler http.Handler) http.Handler {
	instrument := otelhttp.NewMiddleware(
		"HTTP",
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithPropagators(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)),
		otelhttp.WithMeterProvider(meterProvider),
	)

	return func(next http.Handler) http.Handler {
		return instrument(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				return
			}

			span := trace.SpanFromContext(r.Context())
			span.SetName(r.Method + " " + route)
			span.SetAttributes(attribute.String("http.route", route))
			if labeler, ok := otelhttp.LabelerFromContext(r.Context()); ok {
				labeler.Add(attribute.String("http.route", route))
			}
		}))
	}
}
