package router

import (
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/go-chi/chi/v5"
)

func New(h api.ServerInterface) http.Handler {
	r := chi.NewRouter()
	return api.HandlerFromMux(h, r)
}
