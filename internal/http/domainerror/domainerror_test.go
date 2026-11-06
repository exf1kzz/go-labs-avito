package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"driver busy", domaintrip.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{"trip completed", domaintrip.ErrTripCompleted, http.StatusConflict, "trip_completed"},
		{"trip not found", domaintrip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"internal error", errors.New("secret database detail"), http.StatusInternalServerError, "internal_error"},
		{"validation", &domaintrip.ValidationError{Field: "price", Reason: "must not be negative"}, http.StatusBadRequest, "invalid_request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := domainerror.Problem(fmt.Errorf("wrapped: %w", tt.err))

			if status != tt.wantStatus || int(body.Status) != tt.wantStatus {
				t.Errorf("status = %d, body status = %d, want %d", status, body.Status, tt.wantStatus)
			}
			if body.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if body.Detail != nil && strings.Contains(*body.Detail, "secret database detail") {
				t.Error("internal error detail leaked into response")
			}
		})
	}
}
