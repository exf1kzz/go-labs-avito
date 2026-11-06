package create

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

type Handler struct {
	usecase      usecase
	queryTimeout time.Duration
	logger       *slog.Logger
}

func New(usecase usecase, queryTimeout time.Duration, logger *slog.Logger) *Handler {
	return &Handler{
		usecase:      usecase,
		queryTimeout: queryTimeout,
		logger:       logger,
	}
}

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	request, err := decodeCreateTripRequest(w, r)
	if err != nil {
		domainerror.WriteInvalidRequest(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.usecase.Execute(ctx, model.CreateTripInput{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		StartPoint: domaintrip.Point{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},
		EndPoint: domaintrip.Point{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},
		Price: request.Price,
	})
	if err != nil {
		domainerror.Write(h.logger, w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(tripToAPI(trip))
}
