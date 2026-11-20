package trip_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
)

func TestNewPositionValidation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 1, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		coordinates tripdomain.Coordinates
		recordedAt  time.Time
		field       string
	}{
		{
			name:        "invalid latitude",
			coordinates: tripdomain.Coordinates{Latitude: 91, Longitude: 30.3},
			recordedAt:  now,
			field:       "coordinates.latitude",
		},
		{
			name:        "too far in future",
			coordinates: tripdomain.Coordinates{Latitude: 59.9, Longitude: 30.3},
			recordedAt:  now.Add(time.Minute + time.Microsecond),
			field:       "recorded_at",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := tripdomain.NewPosition(uuid.New(), test.coordinates, test.recordedAt, now)
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
