package handler

import (
	api "github.com/exf1kzz/go-labs-avito/internal/generated"
	"github.com/exf1kzz/go-labs-avito/internal/model"
)

func tripToAPI(trip model.Trip) api.Trip {
	return api.Trip{
		Id:       trip.ID,
		UserId:   trip.UserID,
		DriverId: trip.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  trip.StartPoint.Latitude,
			Longitude: trip.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  trip.EndPoint.Latitude,
			Longitude: trip.EndPoint.Longitude,
		},
		Price:          trip.Price,
		Status:         api.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: trip.LastPositionAt,
	}
}
