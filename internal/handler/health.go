package handler

import (
	"encoding/json"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.Ok,
	}); err != nil {
		return
	}
}
