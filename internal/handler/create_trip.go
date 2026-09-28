package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/model"
	"github.com/exf1kzz/go-labs-avito/internal/service"
)

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	request, err := decodeCreateTripRequest(w, r)
	if err != nil {
		WriteInvalidRequest(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.tripService.Create(ctx, service.CreateTripInput{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		StartPoint: model.Point{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},
		EndPoint: model.Point{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},
		Price: request.Price,
	})
	if err != nil {
		h.writeCreateTripError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(tripToAPI(trip))
}

func (h *Handler) writeCreateTripError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	if errors.Is(err, model.ErrDriverBusy) {
		writeProblem(
			w,
			r,
			http.StatusConflict,
			"https://tripgo.example/problems/driver-busy",
			"Driver busy",
			"Driver already has an active trip",
			"driver_busy",
		)
		return
	}

	writeInternalError(w, r, err)
}
