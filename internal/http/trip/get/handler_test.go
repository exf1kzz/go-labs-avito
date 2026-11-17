package get_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	gethttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/get"
)

func TestGetTrip(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	driverID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	startedAt := time.Now().UTC()
	dbErr := errors.New("secret database detail")

	foundTrip := domaintrip.Trip{
		ID:         id,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domaintrip.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
		Status:     domaintrip.ActiveStatus,
		StartedAt:  startedAt,
	}
	expectedBody := api.Trip{
		Id:         id,
		UserId:     userID,
		DriverId:   driverID,
		StartPoint: api.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   api.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
		Status:     api.TripStatus(domaintrip.ActiveStatus),
		StartedAt:  startedAt,
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name            string
		setup           func(*gomock.Controller) *gethttp.Handler
		in              uuid.UUID
		wantStatus      int
		wantContentType string
		wantTrip        *api.Trip
		wantProblemCode string
	}{
		{
			name: "found",
			setup: func(ctrl *gomock.Controller) *gethttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), id).Return(foundTrip, nil)
				return gethttp.New(uc, time.Second, logger)
			},
			in:              id,
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantTrip:        &expectedBody,
		},
		{
			name: "not found",
			setup: func(ctrl *gomock.Controller) *gethttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), id).Return(domaintrip.Trip{}, fmt.Errorf("get: %w", domaintrip.ErrTripNotFound))
				return gethttp.New(uc, time.Second, logger)
			},
			in:              id,
			wantStatus:      http.StatusNotFound,
			wantContentType: "application/problem+json",
			wantProblemCode: "trip_not_found",
		},
		{
			name: "repository error",
			setup: func(ctrl *gomock.Controller) *gethttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), id).Return(domaintrip.Trip{}, dbErr)
				return gethttp.New(uc, time.Second, logger)
			},
			in:              id,
			wantStatus:      http.StatusInternalServerError,
			wantContentType: "application/problem+json",
			wantProblemCode: "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			handler := tt.setup(ctrl)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tt.in.String(), nil)
			rec := httptest.NewRecorder()
			handler.GetTrip(rec, req, tt.in)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Equal(t, tt.wantContentType, rec.Header().Get("Content-Type"))

			if tt.wantTrip != nil {
				var got api.Trip
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				assert.Equal(t, *tt.wantTrip, got)
				return
			}

			var got api.Problem
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Equal(t, tt.wantProblemCode, got.Code)
			assert.Equal(t, tt.wantStatus, int(got.Status))
			assert.NotContains(t, rec.Body.String(), dbErr.Error())
		})
	}
}
