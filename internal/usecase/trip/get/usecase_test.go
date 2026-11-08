package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	"github.com/Rogach26/avito-go-autumn-2026/internal/usecase/trip/get"
)

func TestUseCaseExecute(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	id := uuid.New()
	expected := model.Trip{ID: id}
	repository.EXPECT().Get(gomock.Any(), id).Return(expected, nil)

	result, err := get.New(repository).Execute(context.Background(), id)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.ID != id {
		t.Fatalf("expected id %s, got %s", id, result.ID)
	}
}

func TestUseCaseExecuteNotFound(t *testing.T) {
	t.Parallel()

	controller := gomock.NewController(t)
	repository := NewMocktripRepository(controller)
	id := uuid.New()
	repository.EXPECT().Get(gomock.Any(), id).Return(model.Trip{}, tripdomain.ErrTripNotFound)

	_, err := get.New(repository).Execute(context.Background(), id)
	if !errors.Is(err, tripdomain.ErrTripNotFound) {
		t.Fatalf("expected ErrTripNotFound, got %v", err)
	}
}
