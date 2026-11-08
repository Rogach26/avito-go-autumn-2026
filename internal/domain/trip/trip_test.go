package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
)

func TestNewValidation(t *testing.T) {
	t.Parallel()

	validUserID := uuid.New()
	validStartPoint := tripdomain.Coordinates{Latitude: 59.9, Longitude: 30.3}

	tests := []struct {
		name       string
		userID     uuid.UUID
		startPoint tripdomain.Coordinates
		price      int64
		field      string
	}{
		{name: "empty user", userID: uuid.Nil, startPoint: validStartPoint, price: 100, field: "user_id"},
		{name: "invalid latitude", userID: validUserID, startPoint: tripdomain.Coordinates{Latitude: 91, Longitude: 30.3}, price: 100, field: "start_point.latitude"},
		{name: "negative price", userID: validUserID, startPoint: validStartPoint, price: -1, field: "price"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := tripdomain.New(
				uuid.New(),
				test.userID,
				uuid.New(),
				test.startPoint,
				tripdomain.Coordinates{Latitude: 59.8, Longitude: 30.4},
				test.price,
				time.Now(),
			)
			var validationError *tripdomain.ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("expected ValidationError, got %v", err)
			}
			if validationError.Field != test.field {
				t.Fatalf("expected field %q, got %q", test.field, validationError.Field)
			}
		})
	}
}

func TestTripFinish(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.October, 1, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(time.Hour)
	tests := []struct {
		name             string
		alreadyCompleted bool
		finishAt         time.Time
		wantErr          error
	}{
		{name: "active trip", finishAt: finishedAt},
		{name: "completed trip", alreadyCompleted: true, finishAt: finishedAt.Add(time.Minute), wantErr: tripdomain.ErrTripCompleted},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			entity, err := tripdomain.New(
				uuid.New(),
				uuid.New(),
				uuid.New(),
				tripdomain.Coordinates{Latitude: 59.9, Longitude: 30.3},
				tripdomain.Coordinates{Latitude: 59.8, Longitude: 30.4},
				100,
				startedAt,
			)
			if err != nil {
				t.Fatalf("create trip: %v", err)
			}
			if test.alreadyCompleted {
				if err := entity.Finish(finishedAt); err != nil {
					t.Fatalf("prepare completed trip: %v", err)
				}
			}

			err = entity.Finish(test.finishAt)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			if entity.Status() != tripdomain.StatusCompleted {
				t.Fatalf("expected completed status, got %s", entity.Status())
			}
			if entity.FinishedAt() == nil || !entity.FinishedAt().Equal(finishedAt) {
				t.Fatalf("unexpected finished_at: %v", entity.FinishedAt())
			}
		})
	}
}
