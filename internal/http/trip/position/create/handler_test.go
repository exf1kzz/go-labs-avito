package create_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	createhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/position/create"
)

const invalidRequest = "invalid_request"

func TestCreateTripPosition(t *testing.T) {
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	recordedAt := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	validBody := `{"coordinates":{"latitude":55.75,"longitude":37.61},"recorded_at":"2026-08-24T12:00:00Z"}`

	input := domaintrip.Position{
		TripID:      tripID,
		Coordinates: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		RecordedAt:  recordedAt,
	}
	saved := input
	saved.ID = 42
	dbErr := errors.New("secret database detail")

	counter, err := metricnoop.NewMeterProvider().Meter("test").Int64Counter("tripgo.positions.saved")
	require.NoError(t, err)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name       string
		body       string
		setup      func(*Mockusecase)
		wantStatus int
		wantCode   string
	}{
		{
			name: "created", body: validBody,
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), input).Return(saved, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "malformed time",
			body:       `{"coordinates":{"latitude":55.75,"longitude":37.61},"recorded_at":"bad"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   invalidRequest,
		},
		{
			name:       "missing latitude",
			body:       `{"coordinates":{"longitude":37.61},"recorded_at":"2026-08-24T12:00:00Z"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   invalidRequest,
		},
		{
			name: "coordinates out of range",
			body: `{"coordinates":{"latitude":91,"longitude":37.61},"recorded_at":"2026-08-24T12:00:00Z"}`,
			setup: func(uc *Mockusecase) {
				invalid := input
				invalid.Coordinates.Latitude = 91
				uc.EXPECT().Execute(gomock.Any(), invalid).Return(
					domaintrip.Position{},
					&domaintrip.ValidationError{Field: "coordinates.latitude", Reason: "is out of range"},
				)
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   invalidRequest,
		},
		{
			name: "trip not found", body: validBody,
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), input).Return(domaintrip.Position{}, domaintrip.ErrTripNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name: "trip completed", body: validBody,
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), input).Return(domaintrip.Position{}, domaintrip.ErrTripCompleted)
			},
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name: "repository error", body: validBody,
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), input).Return(domaintrip.Position{}, dbErr)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMockusecase(ctrl)
			if tt.setup != nil {
				tt.setup(uc)
			}

			handler := createhttp.New(uc, time.Second, logger, counter)
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/trips/"+tripID.String()+"/positions",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateTripPosition(rec, req, tripID)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			if tt.wantCode == "" {
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
				var got api.TripPosition
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				assert.Equal(t, api.TripPosition{
					Id:     saved.ID,
					TripId: tripID,
					Coordinates: api.Coordinates{
						Latitude: 55.75, Longitude: 37.61,
					},
					RecordedAt: recordedAt,
				}, got)
				return
			}

			assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
			var problem api.Problem
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
			assert.Equal(t, tt.wantCode, problem.Code)
			assert.NotContains(t, rec.Body.String(), dbErr.Error())
		})
	}
}
