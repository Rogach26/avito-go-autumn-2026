package create

import (
	"log/slog"
	"net/http"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/domainerror"
	positionhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/position"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/response"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

type Handler struct {
	useCase positionUseCase
	logger  *slog.Logger
}

func NewHandler(useCase positionUseCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id api.TripId) {
	request, err := decodePositionData(w, r)
	if err != nil {
		domainerror.WriteInvalidRequest(w, r, err)
		return
	}

	position, err := h.useCase.Execute(r.Context(), model.CreateTripPositionRequest{
		TripID: id,
		Coordinates: model.Coordinates{
			Latitude:  request.Coordinates.Latitude,
			Longitude: request.Coordinates.Longitude,
		},
		RecordedAt: request.RecordedAt,
	})
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	response.JSON(w, http.StatusCreated, "application/json", positionhttp.MapModelPositionToAPIPosition(position))
}
