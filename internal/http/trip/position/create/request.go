package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

const maxRequestBodySize = 1 << 20

func decodeCreatePositionRequest(
	w http.ResponseWriter,
	r *http.Request,
) (api.PositionData, error) {
	var request struct {
		Coordinates *struct {
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"coordinates"`
		RecordedAt *time.Time `json:"recorded_at"`
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		return api.PositionData{}, fmt.Errorf("decode position request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return api.PositionData{}, errors.New("request body must contain exactly one JSON object")
	}

	if request.Coordinates == nil ||
		request.Coordinates.Latitude == nil ||
		request.Coordinates.Longitude == nil ||
		request.RecordedAt == nil {
		return api.PositionData{}, errors.New("coordinates and recorded_at are required")
	}

	return api.PositionData{
		Coordinates: api.Coordinates{
			Latitude:  *request.Coordinates.Latitude,
			Longitude: *request.Coordinates.Longitude,
		},
		RecordedAt: *request.RecordedAt,
	}, nil
}
