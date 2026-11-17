package finish_test

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
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	finishhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/finish"
)

const contentType = "application/problem+json"

func TestFinishTrip(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	driverID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	startedAt := time.Now().UTC()
	finishedAt := startedAt.Add(time.Hour)
	dbErr := errors.New("secret database detail")

	completedTrip := domaintrip.Trip{
		ID:         id,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domaintrip.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
		Status:     domaintrip.CompletedStatus,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
	}
	expectedBody := api.Trip{
		Id:         id,
		UserId:     userID,
		DriverId:   driverID,
		StartPoint: api.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   api.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
		Status:     api.TripStatus(domaintrip.CompletedStatus),
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
	}

	counter, err := metricnoop.NewMeterProvider().Meter("test").Int64Counter("tripgo.trips.completed")
	require.NoError(t, err)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name            string
		setup           func(*gomock.Controller) *finishhttp.Handler
		in              uuid.UUID
		wantStatus      int
		wantContentType string
		wantTrip        *api.Trip
		wantProblemCode string
	}{
		{
			name: "completed",
			setup: func(ctrl *gomock.Controller) *finishhttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), id).Return(completedTrip, nil)
				return finishhttp.New(uc, time.Second, logger, counter)
			},
			in:              id,
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantTrip:        &expectedBody,
		},
		{
			name: "already completed",
			setup: func(ctrl *gomock.Controller) *finishhttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), id).Return(domaintrip.Trip{}, fmt.Errorf("finish: %w", domaintrip.ErrTripCompleted))
				return finishhttp.New(uc, time.Second, logger, counter)
			},
			in:              id,
			wantStatus:      http.StatusConflict,
			wantContentType: contentType,
			wantProblemCode: "trip_completed",
		},
		{
			name: "not found",
			setup: func(ctrl *gomock.Controller) *finishhttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), id).Return(domaintrip.Trip{}, domaintrip.ErrTripNotFound)
				return finishhttp.New(uc, time.Second, logger, counter)
			},
			in:              id,
			wantStatus:      http.StatusNotFound,
			wantContentType: contentType,
			wantProblemCode: "trip_not_found",
		},
		{
			name: "repository error",
			setup: func(ctrl *gomock.Controller) *finishhttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), id).Return(domaintrip.Trip{}, dbErr)
				return finishhttp.New(uc, time.Second, logger, counter)
			},
			in:              id,
			wantStatus:      http.StatusInternalServerError,
			wantContentType: contentType,
			wantProblemCode: "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			handler := tt.setup(ctrl)

			req := httptest.NewRequest(
				http.MethodPost, "/api/v1/trips/"+tt.in.String()+"/finish", nil,
			)
			rec := httptest.NewRecorder()
			handler.FinishTrip(rec, req, tt.in)

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
