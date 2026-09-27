package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.tripService.Get(ctx, tripID)
	if err != nil {
		h.writeGetTripError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(tripToAPI(trip))
}

func (h *Handler) writeGetTripError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	if errors.Is(err, model.ErrTripNotFound) {
		writeProblem(
			w,
			r,
			http.StatusNotFound,
			"https://tripgo.example/problems/trip-not-found",
			"Trip not found",
			"Trip was not found",
			"trip_not_found",
		)
		return
	}

	writeProblem(
		w,
		r,
		http.StatusInternalServerError,
		"https://tripgo.example/problems/internal-error",
		"Internal Server Error",
		"Internal server error",
		"internal_error",
	)
}
