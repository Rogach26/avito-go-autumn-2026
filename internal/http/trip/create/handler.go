package create

import (
	"log/slog"
	"net/http"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/domainerror"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/response"
	triphttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

type Handler struct {
	useCase tripUseCase
	logger  *slog.Logger
}

func NewHandler(useCase tripUseCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	request, err := decodeTripData(w, r)
	if err != nil {
		domainerror.WriteInvalidRequest(w, r, err)
		return
	}

	trip, err := h.useCase.Execute(r.Context(), model.CreateTripRequest{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		StartPoint: model.Coordinates{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},
		EndPoint: model.Coordinates{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},
		Price: request.Price,
	})
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	response.JSON(w, http.StatusCreated, "application/json", triphttp.MapModelTripToAPITrip(trip))
}
