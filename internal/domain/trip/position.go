package trip

import (
	"time"

	"github.com/google/uuid"
)

type Position struct {
	ID          int64
	TripID      uuid.UUID
	Coordinates Point     `name:"coordinates"`
	RecordedAt  time.Time `validate:"required" name:"recorded_at"`
}

func (p Position) Validate(now time.Time) error {
	if err := validationError(createValidator.Struct(p)); err != nil {
		return err
	}
	if p.RecordedAt.After(now.Add(time.Minute)) {
		return &ValidationError{
			Field:  "recorded_at",
			Reason: "must not be more than a minute in the future",
		}
	}
	return nil
}
