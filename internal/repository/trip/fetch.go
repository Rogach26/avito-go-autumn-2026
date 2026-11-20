package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) ListPositions(ctx context.Context, tripID uuid.UUID) ([]model.TripPosition, error) {
	query, args, err := squirrel.
		Select("id", "trip_id", "latitude", "longitude", "recorded_at").
		From("trip_positions").
		Where(squirrel.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip positions query: %w", err)
	}

	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	db := executor(queryContext, r.pool)

	if _, err := r.get(queryContext, db, tripID); err != nil {
		return nil, err
	}

	rows, err := db.Query(queryContext, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list trip positions: %w", err)
	}
	defer rows.Close()

	positions := make([]model.TripPosition, 0)
	for rows.Next() {
		position, scanErr := scanPosition(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan trip position: %w", scanErr)
		}
		positions = append(positions, position)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip positions: %w", err)
	}
	return positions, nil
}

func (r *Repository) ListStale(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error) {
	query, args, err := squirrel.
		Select("id", "driver_id").
		From("trips").
		Where(squirrel.Eq{statusColumn: string(model.TripStatusActive)}).
		Where(squirrel.Expr(
			"COALESCE(last_position_at, started_at) < now() - (? * INTERVAL '1 microsecond')",
			staleAfter.Microseconds(),
		)).
		OrderBy("COALESCE(last_position_at, started_at) ASC", "id ASC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list stale trips query: %w", err)
	}

	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	rows, err := executor(queryContext, r.pool).Query(queryContext, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stale trips: %w", err)
	}
	defer rows.Close()

	staleTrips := make([]model.StaleTrip, 0)
	for rows.Next() {
		var staleTrip model.StaleTrip
		if scanErr := rows.Scan(&staleTrip.ID, &staleTrip.DriverID); scanErr != nil {
			return nil, fmt.Errorf("scan stale trip: %w", scanErr)
		}
		staleTrips = append(staleTrips, staleTrip)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stale trips: %w", err)
	}
	return staleTrips, nil
}
