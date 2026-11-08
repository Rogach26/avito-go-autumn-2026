package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type tripRepository interface {
	Get(ctx context.Context, id uuid.UUID) (model.Trip, error)
}
