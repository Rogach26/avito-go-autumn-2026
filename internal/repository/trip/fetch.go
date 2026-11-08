package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	return r.get(queryContext, executor(queryContext, r.pool), id)
}

func (r *Repository) get(ctx context.Context, db database, id uuid.UUID) (model.Trip, error) {
	query, args, err := squirrel.
		Select(columns()...).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	trip, err := scan(db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, tripdomain.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return trip, nil
}

func (r *Repository) CountActive(ctx context.Context) (int64, error) {
	query, args, err := squirrel.
		Select("count(*)").
		From("trips").
		Where(squirrel.Eq{statusColumn: string(model.TripStatusActive)}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count active trips query: %w", err)
	}

	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var count int64
	if err := executor(queryContext, r.pool).QueryRow(queryContext, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active trips: %w", err)
	}
	return count, nil
}
