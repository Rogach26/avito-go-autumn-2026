package create_test

import (
	"bytes"
	"encoding/json"
	"errors"
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
	createhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/create"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func TestHandlerSuccess(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	useCase := NewMocktripUseCase(controller)
	id := uuid.New()
	startedAt := time.Now().UTC()
	useCase.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{
		ID: id, UserID: uuid.New(), DriverID: uuid.New(), Status: model.TripStatusActive, StartedAt: startedAt,
	}, nil)

	handler := createhttp.NewHandler(useCase, testLogger())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validRequestBody()))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, api.CreateTripParams{})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if location := recorder.Header().Get("Location"); location != "/api/v1/trips/"+id.String() {
		t.Fatalf("Location = %q", location)
	}
}

func TestHandlerInvalidRequest(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	handler := createhttp.NewHandler(NewMocktripUseCase(controller), testLogger())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(`{"user_id":null}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, api.CreateTripParams{})

	assertProblemCode(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestHandlerInternalErrorDoesNotLeakDetails(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	useCase := NewMocktripUseCase(controller)
	useCase.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{}, errors.New("password=secret"))
	handler := createhttp.NewHandler(useCase, testLogger())
	request := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validRequestBody()))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, request, api.CreateTripParams{})

	assertProblemCode(t, recorder, http.StatusInternalServerError, "internal_error")
	if strings.Contains(recorder.Body.String(), "secret") {
		t.Fatal("internal error leaked to response")
	}
}

func validRequestBody() string {
	return `{"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c","driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5","start_point":{"latitude":59.9398,"longitude":30.3146},"end_point":{"latitude":59.929,"longitude":30.3626},"price":1450}`
}

func assertProblemCode(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d", recorder.Code, status)
	}
	var problem api.Problem
	if err := json.NewDecoder(recorder.Body).Decode(&problem); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if problem.Code != code {
		t.Fatalf("code = %q, want %q", problem.Code, code)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
