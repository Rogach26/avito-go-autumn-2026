package list_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	listhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/position/list"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func TestHandlerSuccess(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	useCase := NewMockpositionUseCase(controller)
	tripID := uuid.New()
	positions := []model.TripPosition{
		{
			ID:          1,
			TripID:      tripID,
			Coordinates: model.Coordinates{Latitude: 59.9, Longitude: 30.3},
			RecordedAt:  time.Date(2026, time.November, 1, 10, 0, 0, 0, time.UTC),
		},
	}
	useCase.EXPECT().Execute(gomock.Any(), tripID).Return(positions, nil)

	handler := listhttp.NewHandler(useCase, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String()+"/positions", nil)
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, tripID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response api.TripPositionsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Positions) != 1 || response.Positions[0].Id != positions[0].ID {
		t.Fatalf("unexpected response: %+v", response)
	}
}
