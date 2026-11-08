package trip

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rogach26/avito-go-autumn-2026/internal/infra/postgres/transaction"
)

const (
	activeTripPerDriverIndex = "trips_driver_id_active_unique_idx"
	statusColumn             = "status"
)

type Repository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *Repository {
	return &Repository{pool: pool, queryTimeout: queryTimeout}
}

func executor(ctx context.Context, fallback database) database {
	if dbTx, ok := transaction.FromContext(ctx); ok {
		return dbTx
	}
	return fallback
}

func columns() []string {
	return []string{
		"id",
		"user_id",
		"driver_id",
		"start_latitude",
		"start_longitude",
		"end_latitude",
		"end_longitude",
		"price",
		statusColumn,
		"started_at",
		"finished_at",
	}
}
