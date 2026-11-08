package get_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	gethttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/get"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func TestHandlerNotFound(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	useCase := NewMocktripUseCase(controller)
	id := uuid.New()
	useCase.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{}, tripdomain.ErrTripNotFound)
	handler := gethttp.NewHandler(useCase, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, id)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
