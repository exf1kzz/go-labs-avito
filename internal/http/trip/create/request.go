package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

const maxRequestBodySize = 1 << 20

func decodeCreateTripRequest(
	w http.ResponseWriter,
	r *http.Request,
) (api.TripData, error) {
	body, err := io.ReadAll(
		http.MaxBytesReader(w, r.Body, maxRequestBodySize),
	)
	if err != nil {
		return api.TripData{}, fmt.Errorf("read request body: %w", err)
	}

	var request api.TripData

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		return api.TripData{}, fmt.Errorf("decode request body: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return api.TripData{}, errors.New("request body must contain exactly one JSON object")
	}

	if err := validateRequiredTripFields(body); err != nil {
		return api.TripData{}, err
	}

	return request, nil
}

func validateRequiredTripFields(body []byte) error {
	var fields map[string]json.RawMessage

	if err := json.Unmarshal(body, &fields); err != nil {
		return fmt.Errorf("decode request fields: %w", err)
	}

	required := []string{
		"user_id",
		"driver_id",
		"start_point",
		"end_point",
		"price",
	}

	for _, name := range required {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("field %q is required", name)
		}
	}

	for _, name := range []string{"start_point", "end_point"} {
		var pointFields map[string]json.RawMessage

		if err := json.Unmarshal(fields[name], &pointFields); err != nil {
			return fmt.Errorf("field %q must be an object", name)
		}

		for _, coordinate := range []string{"latitude", "longitude"} {
			value, ok := pointFields[coordinate]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return fmt.Errorf("field %q is required", name+"."+coordinate)
			}
		}
	}

	return nil
}
