package trip

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPositionValidate(t *testing.T) {
	now := time.Now().UTC()
	valid := Position{
		Coordinates: Point{Latitude: 55.75, Longitude: 37.61},
		RecordedAt:  now,
	}

	tests := []struct {
		name      string
		change    func(*Position)
		wantField string
	}{
		{"valid", func(*Position) {}, ""},
		{"zero coordinates", func(p *Position) { p.Coordinates = Point{} }, ""},
		{"boundary coordinates", func(p *Position) { p.Coordinates = Point{Latitude: -90, Longitude: 180} }, ""},
		{"latitude too high", func(p *Position) { p.Coordinates.Latitude = 91 }, "coordinates.latitude"},
		{"latitude too low", func(p *Position) { p.Coordinates.Latitude = -91 }, "coordinates.latitude"},
		{"longitude too high", func(p *Position) { p.Coordinates.Longitude = 181 }, "coordinates.longitude"},
		{"longitude too low", func(p *Position) { p.Coordinates.Longitude = -181 }, "coordinates.longitude"},
		{"missing recorded at", func(p *Position) { p.RecordedAt = time.Time{} }, "recorded_at"},
		{"one minute ahead is valid", func(p *Position) { p.RecordedAt = now.Add(time.Minute) }, ""},
		{"more than one minute ahead", func(p *Position) { p.RecordedAt = now.Add(time.Minute + time.Nanosecond) }, "recorded_at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			position := valid
			tt.change(&position)

			err := position.Validate(now)
			if tt.wantField == "" {
				assert.NoError(t, err)
				return
			}

			var validationErr *ValidationError
			if assert.ErrorAs(t, err, &validationErr) {
				assert.Equal(t, tt.wantField, validationErr.Field)
			}
		})
	}
}
