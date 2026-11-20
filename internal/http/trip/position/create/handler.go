package create

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
	positionhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/position"
)

type Handler struct {
	usecase      usecase
	queryTimeout time.Duration
	logger       *slog.Logger
	saved        metric.Int64Counter
}

func New(
	usecase usecase,
	queryTimeout time.Duration,
	logger *slog.Logger,
	saved metric.Int64Counter,
) *Handler {
	return &Handler{
		usecase:      usecase,
		queryTimeout: queryTimeout,
		logger:       logger,
		saved:        saved,
	}
}

func (h *Handler) CreateTripPosition(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	request, err := decodeCreatePositionRequest(w, r)
	if err != nil {
		domainerror.WriteInvalidRequest(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	saved, err := h.usecase.Execute(ctx, domaintrip.Position{
		TripID: tripID,
		Coordinates: domaintrip.Point{
			Latitude:  request.Coordinates.Latitude,
			Longitude: request.Coordinates.Longitude,
		},
		RecordedAt: request.RecordedAt,
	})
	if err != nil {
		domainerror.Write(h.logger, w, r, err)
		return
	}

	h.saved.Add(ctx, 1)
	h.logger.DebugContext(ctx, "trip position saved", "trip_id", tripID.String())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(positionhttp.ToAPI(saved))
}
