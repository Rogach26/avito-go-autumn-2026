//go:build integration

package trip_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	tripdomain "github.com/Rogach26/avito-go-autumn-2026/internal/domain/trip"
	"github.com/Rogach26/avito-go-autumn-2026/internal/infra/postgres/transaction"
	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
	triprepository "github.com/Rogach26/avito-go-autumn-2026/internal/repository/trip"
)

func TestRepositoryListStale(t *testing.T) {
	pool := newPool(t)
	repository := triprepository.NewRepository(pool, queryTimeout)
	txManager := transaction.NewManager(pool, queryTimeout, slog.New(slog.NewTextHandler(io.Discard, nil)))
	now := time.Now().UTC()

	stale := newActiveTrip()
	stale.StartedAt = now.Add(-5 * time.Minute)
	fresh := newActiveTrip()
	fresh.StartedAt = now
	completed := newActiveTrip()
	completed.StartedAt = now.Add(-5 * time.Minute)
	completed.Status = model.TripStatusCompleted
	finishedAt := now.Add(-4 * time.Minute)
	completed.FinishedAt = &finishedAt
	withFreshPosition := newActiveTrip()
	withFreshPosition.StartedAt = now.Add(-5 * time.Minute)

	for _, value := range []model.Trip{stale, fresh, completed, withFreshPosition} {
		cleanupTrip(t, pool, value.ID)
		if err := repository.Save(t.Context(), value); err != nil {
			t.Fatalf("save trip %s: %v", value.ID, err)
		}
	}

	err := txManager.Do(t.Context(), func(ctx context.Context) error {
		_, saveErr := repository.SavePosition(ctx, model.TripPosition{
			TripID: withFreshPosition.ID,
			Coordinates: model.Coordinates{
				Latitude:  59.935,
				Longitude: 30.335,
			},
			RecordedAt: now,
		})
		return saveErr
	})
	if err != nil {
		t.Fatalf("save fresh position: %v", err)
	}

	staleTrips, err := repository.ListStale(t.Context(), time.Minute)
	if err != nil {
		t.Fatalf("list stale trips: %v", err)
	}
	found := make(map[uuid.UUID]bool, len(staleTrips))
	for _, value := range staleTrips {
		found[value.ID] = true
	}
	if !found[stale.ID] {
		t.Fatalf("stale trip %s was not selected", stale.ID)
	}
	for _, value := range []model.Trip{fresh, completed, withFreshPosition} {
		if found[value.ID] {
			t.Fatalf("trip %s must not be selected", value.ID)
		}
	}
}

func TestRepositorySavePositionIsAtomic(t *testing.T) {
	pool := newPool(t)
	repository := triprepository.NewRepository(pool, queryTimeout)
	txManager := transaction.NewManager(pool, queryTimeout, slog.New(slog.NewTextHandler(io.Discard, nil)))

	active := newActiveTrip()
	active.StartedAt = time.Now().UTC().Add(-5 * time.Minute)
	cleanupTrip(t, pool, active.ID)
	if err := repository.Save(t.Context(), active); err != nil {
		t.Fatalf("save active trip: %v", err)
	}

	newerRecordedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	olderRecordedAt := newerRecordedAt.Add(-time.Minute)
	for _, recordedAt := range []time.Time{newerRecordedAt, olderRecordedAt} {
		err := txManager.Do(t.Context(), func(ctx context.Context) error {
			_, saveErr := repository.SavePosition(ctx, model.TripPosition{
				TripID: active.ID,
				Coordinates: model.Coordinates{
					Latitude:  59.935,
					Longitude: 30.335,
				},
				RecordedAt: recordedAt,
			})
			return saveErr
		})
		if err != nil {
			t.Fatalf("save position: %v", err)
		}
	}

	savedTrip, err := repository.Get(t.Context(), active.ID)
	if err != nil {
		t.Fatalf("get active trip: %v", err)
	}
	if savedTrip.LastPositionAt == nil || !savedTrip.LastPositionAt.Equal(newerRecordedAt) {
		t.Fatalf("last_position_at: got %v, want %v", savedTrip.LastPositionAt, newerRecordedAt)
	}

	positions, err := repository.ListPositions(t.Context(), active.ID)
	if err != nil {
		t.Fatalf("list positions: %v", err)
	}
	if len(positions) != 2 || !positions[0].RecordedAt.Equal(olderRecordedAt) || !positions[1].RecordedAt.Equal(newerRecordedAt) {
		t.Fatalf("positions are not sorted: %+v", positions)
	}

	err = txManager.Do(t.Context(), func(ctx context.Context) error {
		_, saveErr := repository.SavePosition(ctx, model.TripPosition{
			TripID: active.ID,
			Coordinates: model.Coordinates{
				Latitude:  91,
				Longitude: 30.335,
			},
			RecordedAt: newerRecordedAt.Add(time.Minute),
		})
		return saveErr
	})
	if err == nil {
		t.Fatal("save invalid position succeeded")
	}
	savedTrip, err = repository.Get(t.Context(), active.ID)
	if err != nil {
		t.Fatalf("get trip after rollback: %v", err)
	}
	if savedTrip.LastPositionAt == nil || !savedTrip.LastPositionAt.Equal(newerRecordedAt) {
		t.Fatalf("last_position_at after rollback: got %v, want %v", savedTrip.LastPositionAt, newerRecordedAt)
	}
	positions, err = repository.ListPositions(t.Context(), active.ID)
	if err != nil {
		t.Fatalf("list positions after rollback: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("positions after rollback: got %d, want 2", len(positions))
	}

	completed := newActiveTrip()
	completed.Status = model.TripStatusCompleted
	finishedAt := time.Now().UTC()
	completed.FinishedAt = &finishedAt
	cleanupTrip(t, pool, completed.ID)
	if err = repository.Save(t.Context(), completed); err != nil {
		t.Fatalf("save completed trip: %v", err)
	}

	err = txManager.Do(t.Context(), func(ctx context.Context) error {
		_, saveErr := repository.SavePosition(ctx, model.TripPosition{
			TripID: completed.ID,
			Coordinates: model.Coordinates{
				Latitude:  59.935,
				Longitude: 30.335,
			},
			RecordedAt: time.Now().UTC(),
		})
		return saveErr
	})
	if !errors.Is(err, tripdomain.ErrTripCompleted) {
		t.Fatalf("save completed trip position: got %v, want ErrTripCompleted", err)
	}

	var count int
	if err = pool.QueryRow(t.Context(), "SELECT count(*) FROM trip_positions WHERE trip_id = $1", completed.ID).Scan(&count); err != nil {
		t.Fatalf("count rolled back positions: %v", err)
	}
	if count != 0 {
		t.Fatalf("rolled back positions: got %d, want 0", count)
	}
}
