package domainerror

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/response"
)

func Problem(r *http.Request, err error) (int, api.Problem) {
	var validationError *tripdomain.ValidationError
	switch {
	case errors.As(err, &validationError):
		return newProblem(r, http.StatusBadRequest, "invalid_request", "Invalid request", validationError.Error())
	case errors.Is(err, tripdomain.ErrDriverBusy):
		return newProblem(r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	case errors.Is(err, tripdomain.ErrTripNotFound):
		return newProblem(r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
	case errors.Is(err, tripdomain.ErrTripCompleted):
		return newProblem(r, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip")
	default:
		return newProblem(r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
	}
}

func Write(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status, problem := Problem(r, err)
	if status == http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		span := trace.SpanFromContext(r.Context())
		span.RecordError(err)
		span.SetStatus(codes.Error, http.StatusText(status))
		span.SetAttributes(attribute.String("error.type", reflect.TypeOf(err).String()))
	}
	response.JSON(w, status, "application/problem+json", problem)
}

func WriteInvalidRequest(w http.ResponseWriter, r *http.Request, err error) {
	status, problem := newProblem(r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
	response.JSON(w, status, "application/problem+json", problem)
}

func WriteInternal(w http.ResponseWriter, r *http.Request) {
	status, problem := newProblem(r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
	response.JSON(w, status, "application/problem+json", problem)
}

func newProblem(r *http.Request, status int32, code, title, detail string) (int, api.Problem) {
	return int(status), api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   status,
		Detail:   &detail,
		Instance: &r.URL.Path,
		Code:     code,
	}
}
