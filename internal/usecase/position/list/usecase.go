package list

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

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) ([]model.TripPosition, error) {
	positions, err := u.repository.ListPositions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list trip positions: %w", err)
	}
	return positions, nil
}
