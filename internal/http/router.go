package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
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
				domainerror.WriteInvalidRequest(w, r)
			},
		},
	)
}
