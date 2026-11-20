package list

import (
	"log/slog"
	"net/http"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/domainerror"
	positionhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/position"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/response"
)

type Handler struct {
	useCase positionUseCase
	logger  *slog.Logger
}

func NewHandler(useCase positionUseCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id api.TripId) {
	positions, err := h.useCase.Execute(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}

	result := make([]api.TripPosition, len(positions))
	for index, position := range positions {
		result[index] = positionhttp.MapModelPositionToAPIPosition(position)
	}
	response.JSON(w, http.StatusOK, "application/json", api.TripPositionsResponse{Positions: result})
}
