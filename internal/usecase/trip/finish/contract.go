package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type (
	tripRepository interface {
		Get(ctx context.Context, id uuid.UUID) (model.Trip, error)
		Complete(ctx context.Context, id uuid.UUID, finishedAt time.Time) (model.Trip, error)
		AppendStatusHistory(ctx context.Context, change model.TripStatusChange) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	tripMetrics interface {
		Completed(ctx context.Context)
	}
)
