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
	createhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/create"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

const contentType = "application/problem+json"

func TestCreateTrip(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	driverID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	tripID := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	startedAt := time.Now().UTC()

	validBody := `{
		"user_id":"00000000-0000-4000-8000-000000000001",
		"driver_id":"00000000-0000-4000-8000-000000000002",
		"start_point":{"latitude":55.75,"longitude":37.61},
		"end_point":{"latitude":55.76,"longitude":37.62},
		"price":100
	}`
	dbErr := errors.New("secret database detail")

	input := model.CreateTripInput{
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: domaintrip.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   domaintrip.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
	}
	createdTrip := domaintrip.Trip{
		ID:         tripID,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     domaintrip.ActiveStatus,
		StartedAt:  startedAt,
	}
	expectedBody := api.Trip{
		Id:         tripID,
		UserId:     userID,
		DriverId:   driverID,
		StartPoint: api.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   api.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
		Status:     api.TripStatus(domaintrip.ActiveStatus),
		StartedAt:  startedAt,
	}

	counter, err := metricnoop.NewMeterProvider().Meter("test").Int64Counter("tripgo.trips.created")
	require.NoError(t, err)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	tests := []struct {
		name            string
		in              string
		setup           func(*gomock.Controller) *createhttp.Handler
		wantStatus      int
		wantContentType string
		wantLocation    string
		wantTrip        *api.Trip
		wantProblemCode string
	}{
		{
			name: "success",
			in:   validBody,
			setup: func(ctrl *gomock.Controller) *createhttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), input).Return(createdTrip, nil)
				return createhttp.New(uc, time.Second, logger, counter)
			},
			wantStatus:      http.StatusCreated,
			wantContentType: "application/json",
			wantLocation:    "/api/v1/trips/" + tripID.String(),
			wantTrip:        &expectedBody,
		},
		{
			name: "invalid body",
			in:   `{`,
			setup: func(ctrl *gomock.Controller) *createhttp.Handler {
				uc := NewMockusecase(ctrl)
				return createhttp.New(uc, time.Second, logger, counter)
			},
			wantStatus:      http.StatusBadRequest,
			wantContentType: contentType,
			wantProblemCode: "invalid_request",
		},
		{
			name: "driver busy",
			in:   validBody,
			setup: func(ctrl *gomock.Controller) *createhttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), input).Return(domaintrip.Trip{}, domaintrip.ErrDriverBusy)
				return createhttp.New(uc, time.Second, logger, counter)
			},
			wantStatus:      http.StatusConflict,
			wantContentType: contentType,
			wantProblemCode: "driver_busy",
		},
		{
			name: "repository error",
			in:   validBody,
			setup: func(ctrl *gomock.Controller) *createhttp.Handler {
				uc := NewMockusecase(ctrl)
				uc.EXPECT().Execute(gomock.Any(), input).Return(domaintrip.Trip{}, dbErr)
				return createhttp.New(uc, time.Second, logger, counter)
			},
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
				http.MethodPost, "/api/v1/trips", strings.NewReader(tt.in),
			)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateTrip(rec, req, api.CreateTripParams{})

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			assert.Equal(t, tt.wantContentType, rec.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantLocation, rec.Header().Get("Location"))

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
