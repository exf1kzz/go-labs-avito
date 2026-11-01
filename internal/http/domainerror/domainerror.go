package domainerror

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	var validationErr *domaintrip.ValidationError

	switch {
	case errors.As(err, &validationErr):
		return problem(http.StatusBadRequest, "invalid-request", "Invalid request",
			validationErr.Error(), "invalid_request")
	case errors.Is(err, domaintrip.ErrDriverBusy):
		return problem(http.StatusConflict, "driver-busy", "Driver busy",
			"Driver already has an active trip", "driver_busy")
	case errors.Is(err, domaintrip.ErrTripCompleted):
		return problem(http.StatusConflict, "trip-completed", "Trip completed",
			"Operation is not allowed for a completed trip", "trip_completed")
	case errors.Is(err, domaintrip.ErrTripNotFound):
		return problem(http.StatusNotFound, "trip-not-found", "Trip not found",
			"Trip was not found", "trip_not_found")
	default:
		return problem(http.StatusInternalServerError, "internal-error",
			"Internal Server Error", "Internal server error", "internal_error")
	}
}

func problem(status int32, slug, title, detail, code string) (int, api.Problem) {
	return int(status), api.Problem{
		Type:   "https://tripgo.example/problems/" + slug,
		Title:  title,
		Status: status,
		Detail: &detail,
		Code:   code,
	}
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	if status == http.StatusInternalServerError {
		slog.Error("internal request error",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}

	write(w, r, status, body)
}

func WriteInvalidRequest(w http.ResponseWriter, r *http.Request) {
	detail := "Request validation failed"
	write(w, r, http.StatusBadRequest, api.Problem{
		Type:   "https://tripgo.example/problems/invalid-request",
		Title:  "Invalid request",
		Status: http.StatusBadRequest,
		Detail: &detail,
		Code:   "invalid_request",
	})
}

func write(w http.ResponseWriter, r *http.Request, status int, body api.Problem) {
	instance := r.URL.Path
	body.Instance = &instance

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
