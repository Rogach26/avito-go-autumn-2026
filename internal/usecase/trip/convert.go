package tripusecase

import (
	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func FromDomain(entity tripdomain.Trip) model.Trip {
	return model.Trip{
		ID:       entity.ID(),
		UserID:   entity.UserID(),
		DriverID: entity.DriverID(),
		StartPoint: model.Coordinates{
			Latitude:  entity.StartPoint().Latitude,
			Longitude: entity.StartPoint().Longitude,
		},
		EndPoint: model.Coordinates{
			Latitude:  entity.EndPoint().Latitude,
			Longitude: entity.EndPoint().Longitude,
		},
		Price:      entity.Price(),
		Status:     model.TripStatus(entity.Status()),
		StartedAt:  entity.StartedAt(),
		FinishedAt: entity.FinishedAt(),
	}
}

func ToDomain(trip model.Trip) (tripdomain.Trip, error) {
	return tripdomain.Restore(
		trip.ID,
		trip.UserID,
		trip.DriverID,
		tripdomain.Coordinates{
			Latitude:  trip.StartPoint.Latitude,
			Longitude: trip.StartPoint.Longitude,
		},
		tripdomain.Coordinates{
			Latitude:  trip.EndPoint.Latitude,
			Longitude: trip.EndPoint.Longitude,
		},
		trip.Price,
		tripdomain.Status(trip.Status),
		trip.StartedAt,
		trip.FinishedAt,
	)
}
