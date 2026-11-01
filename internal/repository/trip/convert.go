package trip

import (
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/exf1kzz/go-labs-avito/internal/domain/trip"
)

func scanTrip(row pgx.Row) (domaintrip.Trip, error) {
	var trip domaintrip.Trip

	err := row.Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
	)
	if err != nil {
		return domaintrip.Trip{}, err
	}

	return trip, nil
}
