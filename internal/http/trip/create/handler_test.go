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
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	createhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/create"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

func TestCreateTrip(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	driverID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	tripID := uuid.New()
	validBody := `{
		"user_id":"00000000-0000-0000-0000-000000000001",
		"driver_id":"00000000-0000-0000-0000-000000000002",
		"start_point":{"latitude":55.75,"longitude":37.61},
		"end_point":{"latitude":55.76,"longitude":37.62},
		"price":100
	}`
	dbErr := errors.New("secret database detail")

	wantInput := model.CreateTripInput{
		UserID:   userID,
		DriverID: driverID,
		StartPoint: domaintrip.Point{
			Latitude:  55.75,
			Longitude: 37.61,
		},
		EndPoint: domaintrip.Point{
			Latitude:  55.76,
			Longitude: 37.62,
		},
		Price: 100,
	}

	createdTrip := domaintrip.Trip{
		ID:        tripID,
		UserID:    userID,
		DriverID:  driverID,
		Status:    domaintrip.ActiveStatus,
		StartedAt: time.Now().UTC(),
	}

	tests := []struct {
		name       string
		body       string
		setup      func(*testing.T) *Mockusecase
		wantStatus int
		wantCode   string
	}{
		{
			name: "success",
			body: validBody,
			setup: func(t *testing.T) *Mockusecase {
				uc := NewMockusecase(gomock.NewController(t))
				uc.EXPECT().
					Execute(gomock.Any(), gomock.Eq(wantInput)).
					Return(createdTrip, nil)
				return uc
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "invalid body",
			body: `{`,
			setup: func(t *testing.T) *Mockusecase {
				return NewMockusecase(gomock.NewController(t))
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name: "driver busy",
			body: validBody,
			setup: func(t *testing.T) *Mockusecase {
				uc := NewMockusecase(gomock.NewController(t))
				uc.EXPECT().
					Execute(gomock.Any(), gomock.Eq(wantInput)).
					Return(domaintrip.Trip{}, domaintrip.ErrDriverBusy)
				return uc
			},
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name: "repository error",
			body: validBody,
			setup: func(t *testing.T) *Mockusecase {
				uc := NewMockusecase(gomock.NewController(t))
				uc.EXPECT().
					Execute(gomock.Any(), gomock.Eq(wantInput)).
					Return(domaintrip.Trip{}, dbErr)
				return uc
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := tt.setup(t)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			counter, err := metricnoop.NewMeterProvider().Meter("test").Int64Counter("tripgo.trips.created")
			if err != nil {
				t.Fatal(err)
			}

			createhttp.New(uc, time.Second, slog.New(slog.NewJSONHandler(io.Discard, nil)), counter).CreateTrip(rec, req, api.CreateTripParams{})

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				if rec.Header().Get("Location") != "/api/v1/trips/"+tripID.String() {
					t.Errorf("Location = %q", rec.Header().Get("Location"))
				}
				var body struct {
					ID uuid.UUID `json:"id"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.ID != tripID {
					t.Errorf("response ID = %s, error = %v", body.ID, err)
				}
				return
			}

			var problem api.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
				t.Fatalf("decode Problem: %v", err)
			}
			if problem.Code != tt.wantCode {
				t.Errorf("problem code = %q, want %q", problem.Code, tt.wantCode)
			}
			if strings.Contains(rec.Body.String(), dbErr.Error()) {
				t.Error("database error leaked into HTTP response")
			}
		})
	}
}
