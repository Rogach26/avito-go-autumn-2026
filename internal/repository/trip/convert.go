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
		&trip.LastPositionAt,
	); err != nil {
		return model.Trip{}, err
	}
	trip.StartedAt = trip.StartedAt.UTC()
	if trip.FinishedAt != nil {
		*trip.FinishedAt = trip.FinishedAt.UTC()
	}
	if trip.LastPositionAt != nil {
		*trip.LastPositionAt = trip.LastPositionAt.UTC()
	}
	return trip, nil
}

func scanPosition(row pgx.Row) (model.TripPosition, error) {
	var position model.TripPosition
	if err := row.Scan(
		&position.ID,
		&position.TripID,
		&position.Coordinates.Latitude,
		&position.Coordinates.Longitude,
		&position.RecordedAt,
	); err != nil {
		return model.TripPosition{}, err
	}
	position.RecordedAt = position.RecordedAt.UTC()
	return position, nil
}
