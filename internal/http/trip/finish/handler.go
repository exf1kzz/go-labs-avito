package finish

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(tripToAPI(trip))
}
