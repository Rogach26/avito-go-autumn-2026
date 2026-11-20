package create_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	createhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/position/create"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func TestHandlerSuccess(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	useCase := NewMockpositionUseCase(controller)
	tripID := uuid.New()
	recordedAt := time.Date(2026, time.November, 1, 10, 0, 0, 0, time.UTC)
	saved := model.TripPosition{
		ID:          42,
		TripID:      tripID,
		Coordinates: model.Coordinates{Latitude: 59.9, Longitude: 30.3},
		RecordedAt:  recordedAt,
	}
	useCase.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(saved, nil)

	handler := createhttp.NewHandler(useCase, testLogger())
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips/"+tripID.String()+"/positions",
		bytes.NewBufferString(`{"coordinates":{"latitude":59.9,"longitude":30.3},"recorded_at":"2026-11-01T10:00:00Z"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, tripID)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	var response api.TripPosition
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Id != saved.ID || response.TripId != saved.TripID {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestHandlerInvalidRequest(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	handler := createhttp.NewHandler(NewMockpositionUseCase(controller), testLogger())
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/trips/"+uuid.NewString()+"/positions",
		strings.NewReader(`{"coordinates":{"latitude":59.9,"longitude":30.3}}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, uuid.New())

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var problem api.Problem
	if err := json.NewDecoder(recorder.Body).Decode(&problem); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if problem.Code != "invalid_request" {
		t.Fatalf("code = %q, want invalid_request", problem.Code)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
