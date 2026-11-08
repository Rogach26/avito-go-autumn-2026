package trip

import (
	"github.com/jackc/pgx/v5"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func scan(row pgx.Row) (model.Trip, error) {
	var trip model.Trip
	if err := row.Scan(
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
	); err != nil {
		return model.Trip{}, err
	}
	trip.StartedAt = trip.StartedAt.UTC()
	if trip.FinishedAt != nil {
		*trip.FinishedAt = trip.FinishedAt.UTC()
	}
	return trip, nil
}
