package domainerror_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "validation", err: &tripdomain.ValidationError{Field: "price", Reason: "must not be negative"}, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "driver busy", err: tripdomain.ErrDriverBusy, wantStatus: http.StatusConflict, wantCode: "driver_busy"},
		{name: "trip not found", err: tripdomain.ErrTripNotFound, wantStatus: http.StatusNotFound, wantCode: "trip_not_found"},
		{name: "trip completed", err: tripdomain.ErrTripCompleted, wantStatus: http.StatusConflict, wantCode: "trip_completed"},
		{name: "internal", err: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/trips/id", nil)
			status, problem := domainerror.Problem(request, test.err)
			if status != test.wantStatus {
				t.Fatalf("status = %d, want %d", status, test.wantStatus)
			}
			if problem.Code != test.wantCode {
				t.Fatalf("code = %q, want %q", problem.Code, test.wantCode)
			}
			if problem.Instance == nil || *problem.Instance != request.URL.Path {
				t.Fatalf("instance = %v, want %q", problem.Instance, request.URL.Path)
			}
		})
	}
}
