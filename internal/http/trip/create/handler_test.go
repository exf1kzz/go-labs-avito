package create_test

import (
	"context"
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

	tests := []struct {
		name        string
		body        string
		usecaseErr  error
		wantStatus  int
		wantCode    string
		callUsecase bool
	}{
		{"success", validBody, nil, http.StatusCreated, "", true},
		{"invalid body", `{`, nil, http.StatusBadRequest, "invalid_request", false},
		{"driver busy", validBody, domaintrip.ErrDriverBusy, http.StatusConflict, "driver_busy", true},
		{"repository error", validBody, dbErr, http.StatusInternalServerError, "internal_error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMockusecase(ctrl)
			if tt.callUsecase {
				uc.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, input model.CreateTripInput) (domaintrip.Trip, error) {
						if input.UserID != userID || input.DriverID != driverID ||
							input.Price != 100 || input.StartPoint.Latitude != 55.75 {
							t.Errorf("wrong usecase input: %+v", input)
						}

						return domaintrip.Trip{
							ID: tripID, UserID: userID, DriverID: driverID,
							Status:    domaintrip.ActiveStatus,
							StartedAt: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC),
						}, tt.usecaseErr
					},
				)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			createhttp.New(uc, time.Second, slog.New(slog.NewJSONHandler(io.Discard, nil))).CreateTrip(rec, req, api.CreateTripParams{})

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
