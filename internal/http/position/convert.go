package position

import (
	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func MapModelPositionToAPIPosition(position model.TripPosition) api.TripPosition {
	return api.TripPosition{
		Id:     position.ID,
		TripId: position.TripID,
		Coordinates: api.Coordinates{
			Latitude:  position.Coordinates.Latitude,
			Longitude: position.Coordinates.Longitude,
		},
		RecordedAt: position.RecordedAt,
	}
}
