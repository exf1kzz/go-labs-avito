package handler

import (
	"context"
	"encoding/json"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	statusCode := http.StatusOK
	status := api.Ok

	if err := h.database.Ping(ctx); err != nil {
		statusCode = http.StatusServiceUnavailable
		status = api.Unavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(api.HealthResponse{
		Status: status,
	}); err != nil {
		return
	}
}
