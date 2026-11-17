package create

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/exf1kzz/go-labs-avito/internal/generated"
)

func TestDecodeCreateTripRequest(t *testing.T) {
	valid := `{"user_id":"00000000-0000-0000-0000-000000000001", "driver_id":"00000000-0000-0000-0000-000000000002", "start_point":{"latitude":55.75,"longitude":37.61}, "end_point":{"latitude":55.76,"longitude":37.62},"price":100}`

	expected := api.TripData{
		UserId:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		DriverId:   uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		StartPoint: api.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   api.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      100,
	}

	tests := []struct {
		name    string
		in      string
		want    api.TripData
		wantErr string
	}{
		{name: "valid", in: valid, want: expected},
		{name: "invalid JSON", in: `{`, wantErr: "decode request body"},
		{
			name:    "missing price",
			in:      strings.Replace(valid, `,"price":100`, "", 1),
			wantErr: `field "price" is required`,
		},
		{
			name:    "missing coordinate",
			in:      strings.Replace(valid, `"latitude":55.75,`, "", 1),
			wantErr: `field "start_point.latitude" is required`,
		},
		{
			name:    "two JSON objects",
			in:      valid + `{}`,
			wantErr: "exactly one JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.in))
			rec := httptest.NewRecorder()

			got, err := decodeCreateTripRequest(rec, req)

			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
