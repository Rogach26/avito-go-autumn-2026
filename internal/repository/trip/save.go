package trip

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

func (r *Repository) Save(ctx context.Context, trip model.Trip) error {
	query, args, err := squirrel.
		Insert("trips").
		Columns(columns()...).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartPoint.Latitude,
			trip.StartPoint.Longitude,
			trip.EndPoint.Latitude,
			trip.EndPoint.Longitude,
			trip.Price,
			string(trip.Status),
			trip.StartedAt,
			trip.FinishedAt,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip query: %w", err)
	}

	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	_, err = executor(queryContext, r.pool).Exec(queryContext, query, args...)
	if err == nil {
		return nil
	}

	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" && postgresError.ConstraintName == activeTripPerDriverIndex {
		return tripdomain.ErrDriverBusy
	}
	return fmt.Errorf("insert trip: %w", err)
}

func (r *Repository) Complete(ctx context.Context, id uuid.UUID, finishedAt time.Time) (model.Trip, error) {
	query, args, err := squirrel.
		Update("trips").
		Set(statusColumn, string(model.TripStatusCompleted)).
		Set("finished_at", finishedAt).
		Set("updated_at", squirrel.Expr("now()")).
		Where(squirrel.Eq{"id": id, statusColumn: string(model.TripStatusActive)}).
		Suffix("RETURNING " + strings.Join(columns(), ", ")).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build complete trip query: %w", err)
	}

	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	db := executor(queryContext, r.pool)

	trip, err := scan(db.QueryRow(queryContext, query, args...))
	if err == nil {
		return trip, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, fmt.Errorf("complete trip: %w", err)
	}

	_, err = r.get(queryContext, db, id)
	if errors.Is(err, tripdomain.ErrTripNotFound) {
		return model.Trip{}, tripdomain.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, err
	}
	return model.Trip{}, tripdomain.ErrTripCompleted
}

func (r *Repository) AppendStatusHistory(ctx context.Context, change model.TripStatusChange) error {
	var fromStatus any
	if change.FromStatus != nil {
		fromStatus = string(*change.FromStatus)
	}

	query, args, err := squirrel.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason", "changed_at").
		Values(change.TripID, fromStatus, string(change.ToStatus), change.Reason, change.ChangedAt).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build append status history query: %w", err)
	}

	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	if _, err = executor(queryContext, r.pool).Exec(queryContext, query, args...); err != nil {
		return fmt.Errorf("append status history: %w", err)
	}
	return nil
}
