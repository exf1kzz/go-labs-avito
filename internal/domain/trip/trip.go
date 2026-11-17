package trip

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type TripStatus string

const (
	ActiveStatus    TripStatus = "active"
	CompletedStatus TripStatus = "completed"
)

const reasonOutOfRange = "is out of range"

type Point struct {
	Latitude  float64 `validate:"gte=-90,lte=90" name:"latitude"`
	Longitude float64 `validate:"gte=-180,lte=180" name:"longitude"`
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID `validate:"required" name:"user_id"`
	DriverID       uuid.UUID `validate:"required" name:"driver_id"`
	StartPoint     Point     `name:"start_point"`
	EndPoint       Point     `name:"end_point"`
	Price          int64     `validate:"gte=0" name:"price"`
	Status         TripStatus
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != ActiveStatus {
		return ErrTripCompleted
	}

	t.Status = CompletedStatus
	t.FinishedAt = &at
	return nil
}

func (t Trip) ValidateCreate() error {
	return validationError(createValidator.Struct(t))
}

func validationError(err error) error {
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return err
	}

	failure := validationErrors[0]
	_, field, _ := strings.Cut(failure.Namespace(), ".")

	reason := "is invalid"
	switch {
	case failure.Tag() == "required":
		reason = "must not be empty"
	case field == "price" && failure.Tag() == "gte":
		reason = "must not be negative"
	case failure.Tag() == "gte" || failure.Tag() == "lte":
		reason = reasonOutOfRange
	}

	return &ValidationError{Field: field, Reason: reason}
}

var createValidator = newCreateValidator()

func newCreateValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		if name := field.Tag.Get("name"); name != "" {
			return name
		}
		return field.Name
	})
	return v
}
