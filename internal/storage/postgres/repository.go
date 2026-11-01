package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const activeTripPerDriverIndex = "trips_driver_id_active_unique_idx"

var tripColumns = []string{
	"id",
	"user_id",
	"driver_id",
	"start_latitude",
	"start_longitude",
	"end_latitude",
	"end_longitude",
	"price",
	"status",
	"started_at",
	"finished_at",
}

type TripRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTripRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *TripRepository {
	return &TripRepository{pool: pool, queryTimeout: queryTimeout}
}

func (r *TripRepository) CreateTrip(ctx context.Context, trip model.Trip) error {
	query, args, err := squirrel.
		Insert("trips").
		Columns(tripColumns...).
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

	_, err = executor(ctx, r.pool).Exec(queryContext, query, args...)
	if err == nil {
		return nil
	}

	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" && postgresError.ConstraintName == activeTripPerDriverIndex {
		return model.ErrDriverBusy
	}
	return fmt.Errorf("insert trip: %w", err)
}

func (r *TripRepository) GetTrip(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	return r.getTrip(queryContext, executor(ctx, r.pool), id)
}

func (r *TripRepository) CompleteTrip(ctx context.Context, id uuid.UUID, finishedAt time.Time) (model.Trip, error) {
	query, args, err := squirrel.
		Update("trips").
		Set("status", string(model.TripStatusCompleted)).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(squirrel.Eq{"id": id, "status": string(model.TripStatusActive)}).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build complete trip query: %w", err)
	}

	queryContext, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()
	db := executor(ctx, r.pool)

	trip, err := scanTrip(db.QueryRow(queryContext, query, args...))
	if err == nil {
		return trip, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, fmt.Errorf("complete trip: %w", err)
	}

	_, err = r.getTrip(queryContext, db, id)
	if errors.Is(err, model.ErrTripNotFound) {
		return model.Trip{}, model.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, err
	}
	return model.Trip{}, model.ErrTripCompleted
}

func (r *TripRepository) AppendStatusHistory(ctx context.Context, change model.TripStatusChange) error {
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

	if _, err = executor(ctx, r.pool).Exec(queryContext, query, args...); err != nil {
		return fmt.Errorf("append status history: %w", err)
	}
	return nil
}

func (r *TripRepository) getTrip(ctx context.Context, db DBTX, id uuid.UUID) (model.Trip, error) {
	query, args, err := squirrel.
		Select(tripColumns...).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	trip, err := scanTrip(db.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, model.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return trip, nil
}

func scanTrip(row pgx.Row) (model.Trip, error) {
	var trip model.Trip
	if err := row.Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
	); err != nil {
		return model.Trip{}, err
	}
	trip.StartedAt = trip.StartedAt.UTC()
	if trip.FinishedAt != nil {
		*trip.FinishedAt = trip.FinishedAt.UTC()
	}
	return trip, nil
}
