package list_test

import (
	"encoding/json"
	"errors"
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
	listhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/position/list"
)

func TestListTripPositions(t *testing.T) {
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	recordedAt := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	position := domaintrip.Position{
		ID:          42,
		TripID:      tripID,
		Coordinates: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		RecordedAt:  recordedAt,
	}
	dbErr := errors.New("secret database detail")
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name       string
		setup      func(*Mockusecase)
		wantStatus int
		wantCount  int
		wantCode   string
	}{
		{
			name: "route with position",
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return([]domaintrip.Position{position}, nil)
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "empty route",
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return([]domaintrip.Position{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "trip not found",
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(nil, domaintrip.ErrTripNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name: "repository error",
			setup: func(uc *Mockusecase) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(nil, dbErr)
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMockusecase(ctrl)
			tt.setup(uc)

			handler := listhttp.New(uc, time.Second, logger)
			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/trips/"+tripID.String()+"/positions",
				nil,
			)
			rec := httptest.NewRecorder()
			handler.ListTripPositions(rec, req, tripID)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			if tt.wantCode != "" {
				assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
				var problem api.Problem
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
				assert.Equal(t, tt.wantCode, problem.Code)
				assert.NotContains(t, rec.Body.String(), dbErr.Error())
				return
			}

			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			var result api.TripPositionsResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
			require.NotNil(t, result.Positions)
			require.Len(t, result.Positions, tt.wantCount)
			if tt.wantCount == 1 {
				assert.Equal(t, int64(42), result.Positions[0].Id)
				assert.Equal(t, tripID, result.Positions[0].TripId)
				assert.Equal(t, api.Coordinates{Latitude: 55.75, Longitude: 37.61}, result.Positions[0].Coordinates)
				assert.True(t, result.Positions[0].RecordedAt.Equal(recordedAt))
			}
		})
	}
}
