package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.tripService.Finish(ctx, tripID)
	if err != nil {
		h.writeFinishTripError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(tripToAPI(trip))
}

func (h *Handler) writeFinishTripError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, model.ErrTripNotFound):
		writeProblem(
			w,
			r,
			http.StatusNotFound,
			"https://tripgo.example/problems/trip-not-found",
			"Trip not found",
			"Trip was not found",
			"trip_not_found",
		)

	case errors.Is(err, model.ErrTripCompleted):
		writeProblem(
			w,
			r,
			http.StatusConflict,
			"https://tripgo.example/problems/trip-completed",
			"Trip completed",
			"Operation is not allowed for a completed trip",
			"trip_completed",
		)

	default:
		writeInternalError(w, r, err)
	}
}
