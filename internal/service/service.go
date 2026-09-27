package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	"github.com/google/uuid"
)

type TripRepository interface {
	CreateTrip(ctx context.Context, trip model.Trip) error
	GetTrip(ctx context.Context, id uuid.UUID) (model.Trip, error)
	CompleteTrip(ctx context.Context, id uuid.UUID, finishedAt time.Time) (model.Trip, error)
	AppendStatusHistory(ctx context.Context, change model.TripStatusChange) error
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type CreateTripRequest struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint model.Coordinates
	EndPoint   model.Coordinates
	Price      int64
}

type TripService struct {
	repository TripRepository
	txManager  TxManager
}

func NewTripService(repository TripRepository, txManager TxManager) *TripService {
	return &TripService{repository: repository, txManager: txManager}
}

func (s *TripService) CreateTrip(ctx context.Context, request CreateTripRequest) (model.Trip, error) {
	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	trip := model.Trip{
		ID:         uuid.New(),
		UserID:     request.UserID,
		DriverID:   request.DriverID,
		StartPoint: request.StartPoint,
		EndPoint:   request.EndPoint,
		Price:      request.Price,
		Status:     model.TripStatusActive,
		StartedAt:  startedAt,
	}

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		if err := s.repository.CreateTrip(ctx, trip); err != nil {
			return err
		}
		if err := s.repository.AppendStatusHistory(ctx, model.TripStatusChange{
			TripID:    trip.ID,
			ToStatus:  model.TripStatusActive,
			Reason:    "trip created",
			ChangedAt: startedAt,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	return trip, nil
}

func (s *TripService) GetTrip(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	return s.repository.GetTrip(ctx, id)
}

func (s *TripService) FinishTrip(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	var trip model.Trip
	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		completed, err := s.repository.CompleteTrip(ctx, id, finishedAt)
		if err != nil {
			return err
		}
		fromStatus := model.TripStatusActive
		if err := s.repository.AppendStatusHistory(ctx, model.TripStatusChange{
			TripID:     id,
			FromStatus: &fromStatus,
			ToStatus:   model.TripStatusCompleted,
			Reason:     "trip finished",
			ChangedAt:  finishedAt,
		}); err != nil {
			return err
		}
		trip = completed
		return nil
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return trip, nil
}
