package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

type UseCase struct {
	repository tripRepository
}

func New(repository tripRepository) *UseCase {
	return &UseCase{repository: repository}
}

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	trip, err := u.repository.Get(ctx, id)
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return trip, nil
}
