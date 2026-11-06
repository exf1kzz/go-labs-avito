package finish

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/metric"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
)

type Handler struct {
	usecase      usecase
	queryTimeout time.Duration
	logger       *slog.Logger
	completed    metric.Int64Counter
}

func New(
	usecase usecase,
	queryTimeout time.Duration,
	logger *slog.Logger,
	completed metric.Int64Counter,
) *Handler {
	return &Handler{
		usecase:      usecase,
		queryTimeout: queryTimeout,
		logger:       logger,
		completed:    completed,
	}
}

func (h *Handler) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	trip, err := h.usecase.Execute(ctx, tripID)
	if err != nil {
		domainerror.Write(h.logger, w, r, err)
		return
	}

	h.completed.Add(ctx, 1)
	h.logger.InfoContext(ctx, "trip finished", "trip_id", trip.ID.String())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(tripToAPI(trip))
}
