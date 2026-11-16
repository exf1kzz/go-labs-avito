package trip

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTripFinish(t *testing.T) {
	at := time.Now()

	tests := []struct {
		name           string
		initialStatus  TripStatus
		wantStatus     TripStatus
		wantFinishedAt bool
		wantErr        error
	}{
		{
			name:           "active trip is completed",
			initialStatus:  ActiveStatus,
			wantStatus:     CompletedStatus,
			wantFinishedAt: true,
		},
		{
			name:          "completed trip cannot be finished again",
			initialStatus: CompletedStatus,
			wantStatus:    CompletedStatus,
			wantErr:       ErrTripCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := Trip{Status: tt.initialStatus}

			err := trip.Finish(at)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantStatus, trip.Status)

			if assert.Equal(t, tt.wantFinishedAt, trip.FinishedAt != nil) && tt.wantFinishedAt {
				assert.True(t, trip.FinishedAt.Equal(at))
			}
		})
	}
}

func TestValidateCreate(t *testing.T) {
	valid := Trip{
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   Point{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
	}

	tests := []struct {
		name      string
		change    func(*Trip)
		wantField string
	}{
		{"valid", func(*Trip) {}, ""},
		{"empty user", func(t *Trip) { t.UserID = uuid.Nil }, "user_id"},
		{"empty driver", func(t *Trip) { t.DriverID = uuid.Nil }, "driver_id"},
		{"start latitude", func(t *Trip) { t.StartPoint.Latitude = 91 }, "start_point.latitude"},
		{"start longitude", func(t *Trip) { t.StartPoint.Longitude = 181 }, "start_point.longitude"},
		{"end latitude", func(t *Trip) { t.EndPoint.Latitude = -91 }, "end_point.latitude"},
		{"end longitude", func(t *Trip) { t.EndPoint.Longitude = -181 }, "end_point.longitude"},
		{"negative price", func(t *Trip) { t.Price = -1 }, "price"},
		{"zero coordinates and price", func(t *Trip) {
			t.StartPoint = Point{}
			t.EndPoint = Point{}
			t.Price = 0
		}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trip := valid
			tt.change(&trip)

			err := trip.ValidateCreate()
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("ValidateCreate() error = %v", err)
				}
				return
			}

			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("error = %v, want ValidationError", err)
			}
			if validationErr.Field != tt.wantField {
				t.Errorf("field = %q, want %q", validationErr.Field, tt.wantField)
			}
		})
	}
}
