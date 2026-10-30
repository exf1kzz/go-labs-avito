package domainerror

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	switch {
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

func problem(status int, slug, title, detail, code string) (int, api.Problem) {
	return status, api.Problem{
		Type:   "https://tripgo.example/problems/" + slug,
		Title:  title,
		Status: int32(status),
		Detail: &detail,
		Code:   code,
	}
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, body := Problem(err)
	if status == http.StatusInternalServerError {
		log.Printf("internal request error: method=%s path=%s error=%v",
			r.Method, r.URL.Path, err)
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
