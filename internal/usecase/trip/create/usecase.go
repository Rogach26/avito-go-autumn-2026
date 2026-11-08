package create

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
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

func (u *UseCase) Execute(ctx context.Context, request model.CreateTripRequest) (model.Trip, error) {
	startedAt := time.Now()
	entity, err := tripdomain.New(
		uuid.New(),
		request.UserID,
		request.DriverID,
		tripdomain.Coordinates{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},
		tripdomain.Coordinates{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},
		request.Price,
		startedAt,
	)
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip domain model: %w", err)
	}

	trip := tripusecase.FromDomain(entity)
	err = u.txManager.Do(ctx, func(txContext context.Context) error {
		if err := u.repository.Save(txContext, trip); err != nil {
			return err
		}
		return u.repository.AppendStatusHistory(txContext, model.TripStatusChange{
			TripID:    trip.ID,
			ToStatus:  model.TripStatusActive,
			Reason:    "trip created",
			ChangedAt: trip.StartedAt,
		})
	})
	if err != nil {
		return model.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	u.metrics.Created(ctx)
	u.logger.InfoContext(ctx, "trip created", "trip_id", trip.ID)
	return trip, nil
}
