package finish

import (
	"log/slog"
	"net/http"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/domainerror"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/response"
	triphttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip"
)

type Handler struct {
	useCase tripUseCase
	logger  *slog.Logger
}

func NewHandler(useCase tripUseCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id api.TripId) {
	trip, err := h.useCase.Execute(r.Context(), id)
	if err != nil {
		domainerror.Write(w, r, h.logger, err)
		return
	}
	response.JSON(w, http.StatusOK, "application/json", triphttp.MapModelTripToAPITrip(trip))
}
