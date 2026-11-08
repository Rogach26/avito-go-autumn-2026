package finish_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	finishhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/finish"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func TestHandlerCompleted(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	useCase := NewMocktripUseCase(controller)
	id := uuid.New()
	useCase.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{}, tripdomain.ErrTripCompleted)
	handler := finishhttp.NewHandler(useCase, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, id)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
}
