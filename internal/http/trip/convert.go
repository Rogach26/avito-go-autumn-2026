package trip

import (
	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func MapModelTripToAPITrip(trip model.Trip) api.Trip {
	return api.Trip{
		Id:         trip.ID,
		UserId:     trip.UserID,
		DriverId:   trip.DriverID,
		StartPoint: api.Coordinates{Latitude: trip.StartPoint.Latitude, Longitude: trip.StartPoint.Longitude},
		EndPoint:   api.Coordinates{Latitude: trip.EndPoint.Latitude, Longitude: trip.EndPoint.Longitude},
		Price:      trip.Price,
		Status:     api.TripStatus(trip.Status),
		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,
	}
}
