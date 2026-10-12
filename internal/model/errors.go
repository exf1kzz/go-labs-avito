package model

import "errors"

var (
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip completed")
	ErrDriverBusy    = errors.New("driver busy")
)
