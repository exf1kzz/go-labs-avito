package list

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
	positionhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/position"
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

func (h *Handler) ListTripPositions(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()

	positions, err := h.usecase.Execute(ctx, tripID)
	if err != nil {
		domainerror.Write(h.logger, w, r, err)
		return
	}

	result := make([]api.TripPosition, 0, len(positions))
	for _, p := range positions {
		result = append(result, positionhttp.ToAPI(p))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.TripPositionsResponse{
		Positions: result,
	})
}
