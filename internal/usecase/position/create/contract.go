package create

import (
	"context"

	"github.com/google/uuid"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type tripRepository interface {
	SavePosition(ctx context.Context, position model.TripPosition) (model.TripPosition, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type positionMetrics interface {
	Saved(ctx context.Context)
}

type positionQueue interface {
	Cancel(id uuid.UUID)
}
