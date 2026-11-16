package get_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	gethttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/get"
)

func TestGetTrip(t *testing.T) {
	id := uuid.New()
	tests := []struct {
		name       string
		setup      func(*testing.T) *Mockusecase
		wantStatus int
		wantCode   string
	}{
		{
			name: "found",
			setup: func(t *testing.T) *Mockusecase {
				uc := NewMockusecase(gomock.NewController(t))
				uc.EXPECT().Execute(gomock.Any(), id).
					Return(domaintrip.Trip{ID: id, Status: domaintrip.ActiveStatus}, nil)
				return uc
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not found",
			setup: func(t *testing.T) *Mockusecase {
				uc := NewMockusecase(gomock.NewController(t))
				uc.EXPECT().Execute(gomock.Any(), id).
					Return(domaintrip.Trip{}, fmt.Errorf("get: %w", domaintrip.ErrTripNotFound))
				return uc
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := tt.setup(t)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
			rec := httptest.NewRecorder()
			gethttp.New(uc, time.Second, slog.New(slog.NewJSONHandler(io.Discard, nil))).GetTrip(rec, req, id)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var body struct {
					ID uuid.UUID `json:"id"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.ID != id {
					t.Errorf("response ID = %s, error = %v", body.ID, err)
				}
				return
			}

			var problem api.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
				t.Fatalf("decode Problem: %v", err)
			}
			if problem.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", problem.Code, tt.wantCode)
			}
		})
	}
}
