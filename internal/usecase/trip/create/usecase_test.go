package create_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	"github.com/Rogach26/avito-go-autumn-2026/internal/usecase/trip/create"
)

func TestUseCaseExecute(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	txManager := NewMocktxManager(controller)
	metrics := NewMocktripMetrics(controller)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	request := model.CreateTripRequest{
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Coordinates{Latitude: 59.9, Longitude: 30.3},
		EndPoint:   model.Coordinates{Latitude: 59.8, Longitude: 30.4},
		Price:      100,
	}

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
	repository.EXPECT().Save(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, value model.Trip) error {
			if value.UserID != request.UserID || value.Status != model.TripStatusActive {
				t.Fatalf("unexpected trip: %+v", value)
			}
			return nil
		},
	)
	repository.EXPECT().AppendStatusHistory(gomock.Any(), gomock.Any()).Return(nil)
	metrics.EXPECT().Created(gomock.Any())

	result, err := create.New(repository, txManager, metrics, logger).Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ID == uuid.Nil || result.DriverID != request.DriverID {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUseCaseExecuteDriverBusy(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	txManager := NewMocktxManager(controller)
	metrics := NewMocktripMetrics(controller)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	request := model.CreateTripRequest{
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Coordinates{Latitude: 59.9, Longitude: 30.3},
		EndPoint:   model.Coordinates{Latitude: 59.8, Longitude: 30.4},
		Price:      100,
	}

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
	repository.EXPECT().Save(gomock.Any(), gomock.Any()).Return(tripdomain.ErrDriverBusy)

	_, err := create.New(repository, txManager, metrics, logger).Execute(context.Background(), request)
	if !errors.Is(err, tripdomain.ErrDriverBusy) {
		t.Fatalf("expected ErrDriverBusy, got %v", err)
	}
}
