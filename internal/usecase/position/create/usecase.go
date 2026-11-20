package create

import (
	"context"
	"fmt"
	"time"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

type UseCase struct {
	repository tripRepository
	txManager  txManager
	metrics    positionMetrics
	queue      positionQueue
}

func New(repository tripRepository, txManager txManager, metrics positionMetrics, queue positionQueue) *UseCase {
	return &UseCase{repository: repository, txManager: txManager, metrics: metrics, queue: queue}
}

func (u *UseCase) Execute(ctx context.Context, request model.CreateTripPositionRequest) (model.TripPosition, error) {
	entity, err := tripdomain.NewPosition(
		request.TripID,
		tripdomain.Coordinates{
			Latitude:  request.Coordinates.Latitude,
			Longitude: request.Coordinates.Longitude,
		},
		request.RecordedAt,
		time.Now(),
	)
	if err != nil {
		return model.TripPosition{}, fmt.Errorf("create position domain model: %w", err)
	}

	position := model.TripPosition{
		TripID: entity.TripID(),
		Coordinates: model.Coordinates{
			Latitude:  entity.Coordinates().Latitude,
			Longitude: entity.Coordinates().Longitude,
		},
		RecordedAt: entity.RecordedAt(),
	}
	var saved model.TripPosition
	err = u.txManager.Do(ctx, func(txContext context.Context) error {
		var saveError error
		saved, saveError = u.repository.SavePosition(txContext, position)
		return saveError
	})
	if err != nil {
		return model.TripPosition{}, fmt.Errorf("create trip position: %w", err)
	}

	u.queue.Cancel(request.TripID)
	u.metrics.Saved(ctx)
	return saved, nil
}
