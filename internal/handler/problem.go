package handler

import (
	"encoding/json"
	"log"
	"net/http"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	problemType string,
	title string,
	detail string,
	code string,
) {
	instance := r.URL.Path

	response := api.Problem{
		Type:     problemType,
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(response)
}

func WriteInvalidRequest(
	w http.ResponseWriter,
	r *http.Request,
) {
	writeProblem(
		w,
		r,
		http.StatusBadRequest,
		"https://tripgo.example/problems/invalid-request",
		"Invalid request",
		"Request validation failed",
		"invalid_request",
	)
}

func writeInternalError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	log.Printf(
		"internal request error: method=%s path=%s error=%v",
		r.Method,
		r.URL.Path,
		err,
	)

	writeProblem(
		w,
		r,
		http.StatusInternalServerError,
		"https://tripgo.example/problems/internal-error",
		"Internal Server Error",
		"Internal server error",
		"internal_error",
	)
}
