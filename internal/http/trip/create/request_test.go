package create

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeCreateTripRequest(t *testing.T) {
	valid := `{"user_id":"00000000-0000-0000-0000-000000000001", "driver_id":"00000000-0000-0000-0000-000000000002", "start_point":{"latitude":55.75,"longitude":37.61}, "end_point":{"latitude":55.76,"longitude":37.62},"price":100}`

	tests := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"valid", valid, ""},
		{"invalid JSON", `{`, "decode request body"},
		{"missing price", strings.Replace(valid, `,"price":100`, "", 1), `field "price" is required`},
		{"missing coordinate", strings.Replace(valid, `"latitude":55.75,`, "", 1), `field "start_point.latitude" is required`},
		{"two JSON objects", valid + `{}`, "exactly one JSON object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			got, err := decodeCreateTripRequest(rec, req)
			if tt.wantErr == "" {
				if err != nil || got.Price != 100 {
					t.Fatalf("valid request: price = %d, error = %v", got.Price, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}
