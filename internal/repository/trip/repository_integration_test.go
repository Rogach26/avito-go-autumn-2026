//go:build integration

package trip_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/infra/postgres/transaction"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	triprepository "github.com/Rogach26/avito-go-autumn-2026/internal/repository/trip"
)

const queryTimeout = 3 * time.Second

func TestRepositorySaveGetComplete(t *testing.T) {
	pool := newPool(t)
	repository := triprepository.NewRepository(pool, queryTimeout)
	value := newActiveTrip()
	cleanupTrip(t, pool, value.ID)

	if err := repository.Save(t.Context(), value); err != nil {
		t.Fatalf("save trip: %v", err)
	}

	saved, err := repository.Get(t.Context(), value.ID)
	if err != nil {
		t.Fatalf("get trip: %v", err)
	}
	if saved.ID != value.ID || saved.Status != model.TripStatusActive {
		t.Fatalf("unexpected saved trip: %+v", saved)
	}

	finishedAt := value.StartedAt.Add(time.Minute)
	completed, err := repository.Complete(t.Context(), value.ID, finishedAt)
	if err != nil {
		t.Fatalf("complete trip: %v", err)
	}
	if completed.Status != model.TripStatusCompleted {
		t.Fatalf("unexpected status: %s", completed.Status)
	}
	if completed.FinishedAt == nil || completed.FinishedAt.UnixMicro() != finishedAt.UnixMicro() {
		t.Fatalf("unexpected finished_at: %v", completed.FinishedAt)
	}
}

func TestRepositoryDriverBusy(t *testing.T) {
	pool := newPool(t)
	repository := triprepository.NewRepository(pool, queryTimeout)
	first := newActiveTrip()
	second := newActiveTrip()
	second.DriverID = first.DriverID
	cleanupTrip(t, pool, first.ID)
	cleanupTrip(t, pool, second.ID)

	if err := repository.Save(t.Context(), first); err != nil {
		t.Fatalf("save first trip: %v", err)
	}
	if err := repository.Save(t.Context(), second); !errors.Is(err, tripdomain.ErrDriverBusy) {
		t.Fatalf("save second trip: got %v, want ErrDriverBusy", err)
	}
}

func TestTransactionRollback(t *testing.T) {
	pool := newPool(t)
	repository := triprepository.NewRepository(pool, queryTimeout)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	txManager := transaction.NewManager(pool, queryTimeout, logger)
	value := newActiveTrip()
	cleanupTrip(t, pool, value.ID)

	err := txManager.Do(t.Context(), func(ctx context.Context) error {
		if err := repository.Save(ctx, value); err != nil {
			return err
		}
		return repository.AppendStatusHistory(ctx, model.TripStatusChange{
			TripID:    value.ID,
			ToStatus:  model.TripStatus("invalid"),
			Reason:    "force rollback",
			ChangedAt: value.StartedAt,
		})
	})
	if err == nil {
		t.Fatal("transaction succeeded, want error")
	}

	if _, err := repository.Get(t.Context(), value.ID); !errors.Is(err, tripdomain.ErrTripNotFound) {
		t.Fatalf("get rolled back trip: got %v, want ErrTripNotFound", err)
	}

	var historyCount int
	queryContext, cancel := context.WithTimeout(t.Context(), queryTimeout)
	defer cancel()
	if err := pool.QueryRow(queryContext, "SELECT count(*) FROM trip_status_history WHERE trip_id = $1", value.ID).Scan(&historyCount); err != nil {
		t.Fatalf("count status history: %v", err)
	}
	if historyCount != 0 {
		t.Fatalf("status history count: got %d, want 0", historyCount)
	}
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required")
	}
	connectContext, cancel := context.WithTimeout(t.Context(), queryTimeout)
	defer cancel()
	pool, err := pgxpool.New(connectContext, databaseURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	if err := pool.Ping(connectContext); err != nil {
		pool.Close()
		t.Fatalf("ping database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newActiveTrip() model.Trip {
	return model.Trip{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		StartPoint: model.Coordinates{
			Latitude:  59.9398,
			Longitude: 30.3146,
		},
		EndPoint: model.Coordinates{
			Latitude:  59.929,
			Longitude: 30.3626,
		},
		Price:     1450,
		Status:    model.TripStatusActive,
		StartedAt: time.Now().UTC(),
	}
}

func cleanupTrip(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()
		if _, err := pool.Exec(cleanupContext, "DELETE FROM trips WHERE id = $1", id); err != nil {
			t.Errorf("cleanup trip: %v", err)
		}
	})
}
