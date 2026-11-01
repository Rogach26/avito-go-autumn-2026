package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	"github.com/Rogach26/avito-go-autumn-2026/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxRequestBodyBytes = 1 << 20

type Handler struct {
	tripService  *service.TripService
	pool         *pgxpool.Pool
	queryTimeout time.Duration
	logger       *slog.Logger
}

func NewHandler(tripService *service.TripService, pool *pgxpool.Pool, queryTimeout time.Duration, logger *slog.Logger) *Handler {
	return &Handler{tripService: tripService, pool: pool, queryTimeout: queryTimeout, logger: logger}
}

func NewRouter(handler *Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(handler.recoverer)
	return api.HandlerWithOptions(handler, api.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.parameterError,
	})
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	request, err := decodeTripData(w, r)
	if err != nil {
		h.problem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}
	if err := validateTripData(request); err != nil {
		h.problem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}

	trip, err := h.tripService.CreateTrip(r.Context(), service.CreateTripRequest{
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
		h.serviceError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, http.StatusCreated, "application/json", mapModelTripToAPITrip(trip))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	trip, err := h.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		h.serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, "application/json", mapModelTripToAPITrip(trip))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	trip, err := h.tripService.FinishTrip(r.Context(), tripID)
	if err != nil {
		h.serviceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, "application/json", mapModelTripToAPITrip(trip))
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, "application/json", api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func (h *Handler) parameterError(w http.ResponseWriter, r *http.Request, _ error) {
	h.problem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
}

func (h *Handler) serviceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrDriverBusy):
		h.problem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	case errors.Is(err, model.ErrTripNotFound):
		h.problem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
	case errors.Is(err, model.ErrTripCompleted):
		h.problem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip")
	default:
		h.logger.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		h.problem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
	}
}

func (h *Handler) problem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	writeJSON(w, status, "application/problem+json", api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &r.URL.Path,
		Code:     code,
	})
}

func writeJSON(w http.ResponseWriter, status int, contentType string, value any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.ErrorContext(r.Context(), "request panicked", "method", r.Method, "path", r.URL.Path, "panic", recovered, "stack", string(debug.Stack()))
				h.problem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func decodeTripData(w http.ResponseWriter, r *http.Request) (api.TripData, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return api.TripData{}, fmt.Errorf("content type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return api.TripData{}, fmt.Errorf("read request body: %w", err)
	}
	if err := validateRequiredJSONFields(body); err != nil {
		return api.TripData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var request api.TripData
	if err := decoder.Decode(&request); err != nil {
		return api.TripData{}, fmt.Errorf("decode request body: %w", err)
	}
	return request, nil
}

func validateRequiredJSONFields(body []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return fmt.Errorf("decode required fields: %w", err)
	}
	for _, name := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("field %s is required", name)
		}
	}
	for _, name := range []string{"start_point", "end_point"} {
		var point map[string]json.RawMessage
		if err := json.Unmarshal(fields[name], &point); err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}
		for _, coordinate := range []string{"latitude", "longitude"} {
			value, ok := point[coordinate]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return fmt.Errorf("field %s.%s is required", name, coordinate)
			}
		}
	}
	return nil
}

func validateTripData(request api.TripData) error {
	if request.UserId == uuid.Nil || request.DriverId == uuid.Nil {
		return fmt.Errorf("user_id and driver_id must be non-empty UUIDs")
	}
	if err := validateCoordinates(request.StartPoint); err != nil {
		return err
	}
	if err := validateCoordinates(request.EndPoint); err != nil {
		return err
	}
	if request.Price < 0 {
		return fmt.Errorf("price must not be negative")
	}
	return nil
}

func validateCoordinates(coordinates api.Coordinates) error {
	if coordinates.Latitude < -90 || coordinates.Latitude > 90 {
		return fmt.Errorf("latitude is outside the allowed range")
	}
	if coordinates.Longitude < -180 || coordinates.Longitude > 180 {
		return fmt.Errorf("longitude is outside the allowed range")
	}
	return nil
}

func mapModelTripToAPITrip(trip model.Trip) api.Trip {
	return api.Trip{
		Id:         trip.ID,
		UserId:     trip.UserID,
		DriverId:   trip.DriverID,
		StartPoint: api.Coordinates{Latitude: trip.StartPoint.Latitude, Longitude: trip.StartPoint.Longitude},
		EndPoint:   api.Coordinates{Latitude: trip.EndPoint.Latitude, Longitude: trip.EndPoint.Longitude},
		Price:      trip.Price,
		Status:     api.TripStatus(trip.Status),
		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,
	}
}
