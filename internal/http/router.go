package transport

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
	"github.com/exf1kzz/go-labs-avito/internal/http/middleware"
)

func New(h api.ServerInterface, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Logging(logger))

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
