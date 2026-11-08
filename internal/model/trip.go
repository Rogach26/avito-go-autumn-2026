package model

import (
	"time"

	"github.com/google/uuid"
)

type TripStatus string

const (
	TripStatusActive    TripStatus = "active"
	TripStatusCompleted TripStatus = "completed"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
	Status     TripStatus
	StartedAt  time.Time
	FinishedAt *time.Time
}

type TripStatusChange struct {
	TripID     uuid.UUID
	FromStatus *TripStatus
	ToStatus   TripStatus
	Reason     string
	ChangedAt  time.Time
}

type CreateTripRequest struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}
