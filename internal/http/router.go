package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
	"github.com/exf1kzz/go-labs-avito/internal/http/middleware"
)

func New(
	h api.ServerInterface,
	logger *slog.Logger,
	provider trace.TracerProvider,
	meterProvider metric.MeterProvider,
	activeRequests metric.Int64UpDownCounter,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Tracing(provider, meterProvider))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.ActiveRequests(r, activeRequests))

	return api.HandlerWithOptions(
		h,
		api.ChiServerOptions{
			BaseRouter: r,
			ErrorHandlerFunc: func(
				w http.ResponseWriter,
				r *http.Request,
				_ error,
			) {
				domainerror.WriteInvalidRequest(w, r)
			},
		},
	)
}
