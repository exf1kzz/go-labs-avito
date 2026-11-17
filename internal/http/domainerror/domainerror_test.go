package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	"github.com/exf1kzz/go-labs-avito/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantDetail string
	}{
		{
			"driver busy",
			domaintrip.ErrDriverBusy,
			http.StatusConflict,
			"driver_busy",
			"Driver already has an active trip",
		},
		{
			"trip completed",
			domaintrip.ErrTripCompleted,
			http.StatusConflict,
			"trip_completed",
			"Operation is not allowed for a completed trip",
		},
		{
			"trip not found",
			domaintrip.ErrTripNotFound,
			http.StatusNotFound,
			"trip_not_found",
			"Trip was not found",
		},
		{
			"internal error",
			errors.New("secret database detail"),
			http.StatusInternalServerError,
			"internal_error",
			"Internal server error",
		},
		{
			"validation",
			&domaintrip.ValidationError{Field: "price", Reason: "must not be negative"},
			http.StatusBadRequest,
			"invalid_request",
			"price must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := domainerror.Problem(fmt.Errorf("wrapped: %w", tt.err))

			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantStatus, int(body.Status))
			assert.Equal(t, tt.wantCode, body.Code)
			require.NotNil(t, body.Detail)
			assert.Equal(t, tt.wantDetail, *body.Detail)
		})
	}
}
