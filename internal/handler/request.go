package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/google/uuid"
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

	if err := validateTripData(request); err != nil {
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
				return fmt.Errorf("field %q.%s is required", name, coordinate)
			}
		}
	}

	return nil
}

func validateTripData(request api.TripData) error {
	if request.UserId == uuid.Nil {
		return errors.New("user_id must not be empty")
	}

	if request.DriverId == uuid.Nil {
		return errors.New("driver_id must not be empty")
	}

	if request.StartPoint.Latitude < -90 || request.StartPoint.Latitude > 90 {
		return errors.New("start_point.latitude is out of range")
	}

	if request.StartPoint.Longitude < -180 || request.StartPoint.Longitude > 180 {
		return errors.New("start_point.longitude is out of range")
	}

	if request.EndPoint.Latitude < -90 || request.EndPoint.Latitude > 90 {
		return errors.New("end_point.latitude is out of range")
	}

	if request.EndPoint.Longitude < -180 || request.EndPoint.Longitude > 180 {
		return errors.New("end_point.longitude is out of range")
	}

	if request.Price < 0 {
		return errors.New("price must not be negative")
	}

	return nil
}
