package finish_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	"github.com/Rogach26/avito-go-autumn-2026/internal/usecase/trip/finish"
)

func TestUseCaseExecute(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	txManager := NewMocktxManager(controller)
	metrics := NewMocktripMetrics(controller)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	current := activeTrip()
	completed := current
	completed.Status = model.TripStatusCompleted
	finishedAt := time.Now().UTC()
	completed.FinishedAt = &finishedAt

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
	repository.EXPECT().Get(gomock.Any(), current.ID).Return(current, nil)
	repository.EXPECT().Complete(gomock.Any(), current.ID, gomock.Any()).Return(completed, nil)
	repository.EXPECT().AppendStatusHistory(gomock.Any(), gomock.Any()).Return(nil)
	metrics.EXPECT().Completed(gomock.Any())

	result, err := finish.New(repository, txManager, metrics, logger).Execute(context.Background(), current.ID)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Status != model.TripStatusCompleted {
		t.Fatalf("expected completed status, got %s", result.Status)
	}
}

func TestUseCaseExecuteCompleted(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	txManager := NewMocktxManager(controller)
	metrics := NewMocktripMetrics(controller)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	current := activeTrip()
	current.Status = model.TripStatusCompleted
	finishedAt := time.Now().UTC()
	current.FinishedAt = &finishedAt

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
	repository.EXPECT().Get(gomock.Any(), current.ID).Return(current, nil)

	_, err := finish.New(repository, txManager, metrics, logger).Execute(context.Background(), current.ID)
	if !errors.Is(err, tripdomain.ErrTripCompleted) {
		t.Fatalf("expected ErrTripCompleted, got %v", err)
	}
}

func activeTrip() model.Trip {
	return model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Coordinates{Latitude: 59.9, Longitude: 30.3},
		EndPoint:   model.Coordinates{Latitude: 59.8, Longitude: 30.4},
		Price:      100,
		Status:     model.TripStatusActive,
		StartedAt:  time.Now().UTC(),
	}
}
