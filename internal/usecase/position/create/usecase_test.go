package create_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	"github.com/Rogach26/avito-go-autumn-2026/internal/usecase/position/create"
)

func TestUseCaseExecute(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	txManager := NewMocktxManager(controller)
	metrics := NewMockpositionMetrics(controller)
	queue := NewMockpositionQueue(controller)
	request := model.CreateTripPositionRequest{
		TripID:      uuid.New(),
		Coordinates: model.Coordinates{Latitude: 59.9, Longitude: 30.3},
		RecordedAt:  time.Now().Add(-time.Minute),
	}
	saved := model.TripPosition{
		ID:          42,
		TripID:      request.TripID,
		Coordinates: request.Coordinates,
		RecordedAt:  request.RecordedAt.UTC().Truncate(time.Microsecond),
	}

	txManager.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
	repository.EXPECT().SavePosition(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, position model.TripPosition) (model.TripPosition, error) {
			if position.TripID != request.TripID || !position.RecordedAt.Equal(saved.RecordedAt) {
				t.Fatalf("unexpected position: %+v", position)
			}
			return saved, nil
		},
	)
	metrics.EXPECT().Saved(gomock.Any())
	queue.EXPECT().Cancel(request.TripID)

	result, err := create.New(repository, txManager, metrics, queue).Execute(t.Context(), request)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ID != saved.ID || result.TripID != saved.TripID {
		t.Fatalf("unexpected result: %+v", result)
	}
}
