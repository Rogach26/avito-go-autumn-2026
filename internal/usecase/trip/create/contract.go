package create

import (
	"context"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type (
	tripRepository interface {
		Save(ctx context.Context, trip model.Trip) error
		AppendStatusHistory(ctx context.Context, change model.TripStatusChange) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	tripMetrics interface {
		Created(ctx context.Context)
	}
)
