package router

import (
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	triphandler "github.com/exf1kzz/go-labs-avito/internal/handler"
	"github.com/go-chi/chi/v5"
)

func New(h api.ServerInterface) http.Handler {
	r := chi.NewRouter()

	return api.HandlerWithOptions(
		h,
		api.ChiServerOptions{
			BaseRouter: r,
			ErrorHandlerFunc: func(
				w http.ResponseWriter,
				r *http.Request,
				_ error,
			) {
				triphandler.WriteInvalidRequest(w, r)
			},
		},
	)
}
