package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive       Status = "active"
	StatusCompleted    Status = "completed"
	fieldFinishedAt           = "finished_at"
	nonEmptyUUIDReason        = "must be a non-empty UUID"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	id         uuid.UUID
	userID     uuid.UUID
	driverID   uuid.UUID
	startPoint Coordinates
	endPoint   Coordinates
	price      int64
	status     Status
	startedAt  time.Time
	finishedAt *time.Time
}

func New(
	id uuid.UUID,
	userID uuid.UUID,
	driverID uuid.UUID,
	startPoint Coordinates,
	endPoint Coordinates,
	price int64,
	startedAt time.Time,
) (Trip, error) {
	return Restore(id, userID, driverID, startPoint, endPoint, price, StatusActive, startedAt, nil)
}

func Restore(
	id uuid.UUID,
	userID uuid.UUID,
	driverID uuid.UUID,
	startPoint Coordinates,
	endPoint Coordinates,
	price int64,
	status Status,
	startedAt time.Time,
	finishedAt *time.Time,
) (Trip, error) {
	trip := Trip{
		id:         id,
		userID:     userID,
		driverID:   driverID,
		startPoint: startPoint,
		endPoint:   endPoint,
		price:      price,
		status:     status,
		startedAt:  startedAt,
		finishedAt: finishedAt,
	}
	if err := validate(trip); err != nil {
		return Trip{}, err
	}
	trip.startedAt = trip.startedAt.UTC().Truncate(time.Microsecond)
	if trip.finishedAt != nil {
		finishedAt := trip.finishedAt.UTC().Truncate(time.Microsecond)
		trip.finishedAt = &finishedAt
	}
	return trip, nil
}

func (t *Trip) Finish(finishedAt time.Time) error {
	if t.status != StatusActive {
		return ErrTripCompleted
	}
	if finishedAt.IsZero() {
		return &ValidationError{Field: fieldFinishedAt, Reason: "must not be zero"}
	}

	finishedAt = finishedAt.UTC().Truncate(time.Microsecond)
	t.status = StatusCompleted
	t.finishedAt = &finishedAt
	return nil
}

func (t Trip) ID() uuid.UUID {
	return t.id
}

func (t Trip) UserID() uuid.UUID {
	return t.userID
}

func (t Trip) DriverID() uuid.UUID {
	return t.driverID
}

func (t Trip) StartPoint() Coordinates {
	return t.startPoint
}

func (t Trip) EndPoint() Coordinates {
	return t.endPoint
}

func (t Trip) Price() int64 {
	return t.price
}

func (t Trip) Status() Status {
	return t.status
}

func (t Trip) StartedAt() time.Time {
	return t.startedAt
}

func (t Trip) FinishedAt() *time.Time {
	if t.finishedAt == nil {
		return nil
	}
	finishedAt := *t.finishedAt
	return &finishedAt
}

func validate(trip Trip) error {
	if trip.id == uuid.Nil {
		return &ValidationError{Field: "id", Reason: nonEmptyUUIDReason}
	}
	if trip.userID == uuid.Nil {
		return &ValidationError{Field: "user_id", Reason: nonEmptyUUIDReason}
	}
	if trip.driverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Reason: nonEmptyUUIDReason}
	}
	if err := validateCoordinates("start_point", trip.startPoint); err != nil {
		return err
	}
	if err := validateCoordinates("end_point", trip.endPoint); err != nil {
		return err
	}
	if trip.price < 0 {
		return &ValidationError{Field: "price", Reason: "must not be negative"}
	}
	if trip.startedAt.IsZero() {
		return &ValidationError{Field: "started_at", Reason: "must not be zero"}
	}
	if trip.status != StatusActive && trip.status != StatusCompleted {
		return &ValidationError{Field: "status", Reason: "must be active or completed"}
	}
	if trip.status == StatusActive && trip.finishedAt != nil {
		return &ValidationError{Field: fieldFinishedAt, Reason: "must be empty for an active trip"}
	}
	if trip.status == StatusCompleted && trip.finishedAt == nil {
		return &ValidationError{Field: fieldFinishedAt, Reason: "is required for a completed trip"}
	}
	return nil
}

func validateCoordinates(field string, coordinates Coordinates) error {
	if coordinates.Latitude < -90 || coordinates.Latitude > 90 {
		return &ValidationError{Field: field + ".latitude", Reason: "must be between -90 and 90"}
	}
	if coordinates.Longitude < -180 || coordinates.Longitude > 180 {
		return &ValidationError{Field: field + ".longitude", Reason: "must be between -180 and 180"}
	}
	return nil
}
