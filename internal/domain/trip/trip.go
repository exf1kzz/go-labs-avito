package trip

import (
	"time"

	"github.com/google/uuid"
)

type TripStatus string

const (
	ActiveStatus    TripStatus = "active"
	CompletedStatus TripStatus = "completed"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Point
	EndPoint       Point
	Price          int64
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
	if t.UserID == uuid.Nil {
		return &ValidationError{Field: "user_id", Reason: "must not be empty"}
	}
	if t.DriverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Reason: "must not be empty"}
	}
	if t.StartPoint.Latitude < -90 || t.StartPoint.Latitude > 90 {
		return &ValidationError{Field: "start_point.latitude", Reason: "is out of range"}
	}
	if t.StartPoint.Longitude < -180 || t.StartPoint.Longitude > 180 {
		return &ValidationError{Field: "start_point.longitude", Reason: "is out of range"}
	}
	if t.EndPoint.Latitude < -90 || t.EndPoint.Latitude > 90 {
		return &ValidationError{Field: "end_point.latitude", Reason: "is out of range"}
	}
	if t.EndPoint.Longitude < -180 || t.EndPoint.Longitude > 180 {
		return &ValidationError{Field: "end_point.longitude", Reason: "is out of range"}
	}
	if t.Price < 0 {
		return &ValidationError{Field: "price", Reason: "must not be negative"}
	}
	return nil
}
