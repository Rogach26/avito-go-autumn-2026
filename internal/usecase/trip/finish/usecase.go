package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	tripusecase "github.com/Rogach26/avito-go-autumn-2026/internal/usecase/trip"
)

type UseCase struct {
	repository tripRepository
	txManager  txManager
	metrics    tripMetrics
	logger     *slog.Logger
}

func New(repository tripRepository, txManager txManager, metrics tripMetrics, logger *slog.Logger) *UseCase {
	return &UseCase{repository: repository, txManager: txManager, metrics: metrics, logger: logger}
}

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	finishedAt := time.Now()
	var completed model.Trip
	err := u.txManager.Do(ctx, func(txContext context.Context) error {
		current, err := u.repository.Get(txContext, id)
		if err != nil {
			return err
		}

		entity, err := tripusecase.ToDomain(current)
		if err != nil {
			return fmt.Errorf("restore trip domain model: %w", err)
		}
		if err := entity.Finish(finishedAt); err != nil {
			return err
		}

		completed, err = u.repository.Complete(txContext, id, *entity.FinishedAt())
		if err != nil {
			return err
		}
		fromStatus := model.TripStatusActive
		return u.repository.AppendStatusHistory(txContext, model.TripStatusChange{
			TripID:     id,
			FromStatus: &fromStatus,
			ToStatus:   model.TripStatusCompleted,
			Reason:     "trip finished",
			ChangedAt:  *entity.FinishedAt(),
		})
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	u.metrics.Completed(ctx)
	u.logger.InfoContext(ctx, "trip completed", "trip_id", completed.ID)
	return completed, nil
}
