package create

import (
	"context"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type tripUseCase interface {
	Execute(ctx context.Context, request model.CreateTripRequest) (model.Trip, error)
}
