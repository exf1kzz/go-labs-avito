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
		err        error
		wantStatus int
		wantCode   string
	}{
		{"found", nil, http.StatusOK, ""},
		{"not found", fmt.Errorf("get: %w", domaintrip.ErrTripNotFound), http.StatusNotFound, "trip_not_found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMockusecase(ctrl)
			uc.EXPECT().Execute(gomock.Any(), id).Return(domaintrip.Trip{ID: id, Status: domaintrip.ActiveStatus}, tt.err)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
			rec := httptest.NewRecorder()
			gethttp.New(uc, time.Second, slog.New(slog.NewJSONHandler(io.Discard, nil))).GetTrip(rec, req, id)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.err == nil {
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
