package finish_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	finishhttp "github.com/exf1kzz/go-labs-avito/internal/http/trip/finish"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestFinishTrip(t *testing.T) {
	id := uuid.New()
	at := time.Now()
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"completed", nil, http.StatusOK, ""},
		{"already completed", fmt.Errorf("finish: %w", domaintrip.ErrTripCompleted), http.StatusConflict, "trip_completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMockusecase(ctrl)
			uc.EXPECT().Execute(gomock.Any(), id).Return(
				domaintrip.Trip{
					ID: id, Status: domaintrip.CompletedStatus, FinishedAt: &at,
				}, tt.err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
			rec := httptest.NewRecorder()
			finishhttp.New(uc, time.Second).FinishTrip(rec, req, id)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.err == nil {
				var body struct {
					ID     uuid.UUID `json:"id"`
					Status string    `json:"status"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil ||
					body.ID != id || body.Status != "completed" {
					t.Errorf("response = %+v, error = %v", body, err)
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
