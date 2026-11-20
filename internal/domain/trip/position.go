package trip

import (
	"time"

	"github.com/google/uuid"
)

type Position struct {
	tripID      uuid.UUID
	coordinates Coordinates
	recordedAt  time.Time
}

func NewPosition(
	tripID uuid.UUID,
	coordinates Coordinates,
	recordedAt time.Time,
	now time.Time,
) (Position, error) {
	position := Position{
		tripID:      tripID,
		coordinates: coordinates,
		recordedAt:  recordedAt,
	}
	if tripID == uuid.Nil {
		return Position{}, &ValidationError{Field: "trip_id", Reason: nonEmptyUUIDReason}
	}
	if err := validateCoordinates("coordinates", coordinates); err != nil {
		return Position{}, err
	}
	if recordedAt.IsZero() {
		return Position{}, &ValidationError{Field: "recorded_at", Reason: nonZeroTimeReason}
	}
	if recordedAt.After(now.Add(time.Minute)) {
		return Position{}, &ValidationError{Field: "recorded_at", Reason: "must not be more than one minute in the future"}
	}
	position.recordedAt = recordedAt.UTC().Truncate(time.Microsecond)
	return position, nil
}

func (p Position) TripID() uuid.UUID {
	return p.tripID
}

func (p Position) Coordinates() Coordinates {
	return p.coordinates
}

func (p Position) RecordedAt() time.Time {
	return p.recordedAt
}
