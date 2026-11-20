package positions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

const testTimeout = 3 * time.Second

type fakeTicker struct {
	values  chan time.Time
	stopped chan struct{}
	stop    sync.Once
}

type fakeTickerFactory struct {
	ticker *fakeTicker
}

func newFakeTicker() *fakeTicker {
	return &fakeTicker{
		values:  make(chan time.Time, 10),
		stopped: make(chan struct{}),
	}
}

func (t *fakeTicker) C() <-chan time.Time {
	return t.values
}

func (t *fakeTicker) Stop() {
	t.stop.Do(func() {
		close(t.stopped)
	})
}

func (t *fakeTicker) Tick() {
	t.values <- time.Now()
}

func (f fakeTickerFactory) NewTicker(time.Duration) ticker {
	return f.ticker
}

func TestDuplicateTripIsNotQueuedByNextTick(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMockstaleTripRepository(controller)
	gateway := NewMockpushGateway(controller)
	metrics := NewMockpositionMetrics(controller)
	metrics.EXPECT().StaleCheck(gomock.Any(), staleCheckDuplicate)
	allowMetrics(metrics)

	trip := newStaleTrip()
	scanned := make(chan struct{}, 2)
	repository.EXPECT().
		ListStale(gomock.Any(), time.Minute).
		DoAndReturn(func(context.Context, time.Duration) ([]model.StaleTrip, error) {
			scanned <- struct{}{}
			return []model.StaleTrip{trip}, nil
		}).
		Times(2)

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	gateway.EXPECT().
		RequestPosition(gomock.Any(), trip.DriverID, trip.ID).
		DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
			close(requestStarted)
			<-releaseRequest
			return nil
		})

	ticker := newFakeTicker()
	worker := newTestWorker(t, repository, gateway, metrics, ticker, 1, 1)
	worker.Start(t.Context())

	ticker.Tick()
	waitFor(t, scanned)
	waitFor(t, requestStarted)
	ticker.Tick()
	waitFor(t, scanned)
	close(releaseRequest)
	shutdownWorker(t, worker)
}

func TestCancelSkipsQueuedTask(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMockstaleTripRepository(controller)
	gateway := NewMockpushGateway(controller)
	metrics := NewMockpositionMetrics(controller)
	metrics.EXPECT().RequestCancelled(gomock.Any())
	allowMetrics(metrics)

	ticker := newFakeTicker()
	worker := newTestWorker(t, repository, gateway, metrics, ticker, 1, 1)
	trip := newStaleTrip()
	if !worker.enqueue(t.Context(), trip) {
		t.Fatal("enqueue returned false")
	}
	worker.Cancel(trip.ID)

	worker.Start(t.Context())
	shutdownWorker(t, worker)
}

func TestFullQueueDropsWithoutBlocking(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMockstaleTripRepository(controller)
	gateway := NewMockpushGateway(controller)
	metrics := NewMockpositionMetrics(controller)
	metrics.EXPECT().QueueSizeChanged(int64(1))
	metrics.EXPECT().StaleCheck(gomock.Any(), staleCheckQueued)
	metrics.EXPECT().StaleCheck(gomock.Any(), staleCheckDropped)

	worker := newTestWorker(t, repository, gateway, metrics, newFakeTicker(), 1, 1)
	if !worker.enqueue(t.Context(), newStaleTrip()) {
		t.Fatal("first enqueue returned false")
	}
	if worker.enqueue(t.Context(), newStaleTrip()) {
		t.Fatal("second enqueue returned true for a full queue")
	}
	if len(worker.queue) != 1 {
		t.Fatalf("queue length: got %d, want 1", len(worker.queue))
	}
}

func TestWorkerPoolProcessesTasksConcurrently(t *testing.T) {
	const workerCount = 3

	controller := gomock.NewController(t)
	repository := NewMockstaleTripRepository(controller)
	gateway := NewMockpushGateway(controller)
	metrics := NewMockpositionMetrics(controller)
	allowMetrics(metrics)

	started := make(chan struct{}, workerCount)
	release := make(chan struct{})
	gateway.EXPECT().
		RequestPosition(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
			started <- struct{}{}
			<-release
			return nil
		}).
		Times(workerCount)

	worker := newTestWorker(t, repository, gateway, metrics, newFakeTicker(), workerCount, workerCount)
	for range workerCount {
		if !worker.enqueue(t.Context(), newStaleTrip()) {
			t.Fatal("enqueue returned false")
		}
	}
	worker.Start(t.Context())
	for range workerCount {
		waitFor(t, started)
	}
	close(release)
	shutdownWorker(t, worker)
}

func TestShutdownStopsScannerAndWorkers(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMockstaleTripRepository(controller)
	gateway := NewMockpushGateway(controller)
	metrics := NewMockpositionMetrics(controller)
	allowMetrics(metrics)

	ticker := newFakeTicker()
	worker := newTestWorker(t, repository, gateway, metrics, ticker, 2, 2)
	worker.Start(t.Context())
	shutdownWorker(t, worker)
	waitFor(t, ticker.stopped)
}

func TestPushErrorDoesNotStopWorker(t *testing.T) {
	controller := gomock.NewController(t)
	repository := NewMockstaleTripRepository(controller)
	gateway := NewMockpushGateway(controller)
	metrics := NewMockpositionMetrics(controller)
	allowMetrics(metrics)

	first := newStaleTrip()
	second := newStaleTrip()
	requests := make(chan struct{}, 2)
	gomock.InOrder(
		gateway.EXPECT().
			RequestPosition(gomock.Any(), first.DriverID, first.ID).
			DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
				requests <- struct{}{}
				return errors.New("push unavailable")
			}),
		gateway.EXPECT().
			RequestPosition(gomock.Any(), second.DriverID, second.ID).
			DoAndReturn(func(context.Context, uuid.UUID, uuid.UUID) error {
				requests <- struct{}{}
				return nil
			}),
	)

	worker := newTestWorker(t, repository, gateway, metrics, newFakeTicker(), 1, 2)
	if !worker.enqueue(t.Context(), first) || !worker.enqueue(t.Context(), second) {
		t.Fatal("enqueue returned false")
	}
	worker.Start(t.Context())
	waitFor(t, requests)
	waitFor(t, requests)
	shutdownWorker(t, worker)
}

func newTestWorker(
	t *testing.T,
	repository staleTripRepository,
	gateway pushGateway,
	metrics positionMetrics,
	ticker *fakeTicker,
	workerCount int,
	queueSize int,
) *WorkerPool {
	t.Helper()
	worker, err := newWorkerPool(
		repository,
		gateway,
		metrics,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		noop.NewTracerProvider(),
		fakeTickerFactory{ticker: ticker},
		time.Second,
		time.Minute,
		time.Second,
		workerCount,
		queueSize,
	)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}
	return worker
}

func allowMetrics(metrics *MockpositionMetrics) {
	metrics.EXPECT().StaleCheck(gomock.Any(), gomock.Any()).AnyTimes()
	metrics.EXPECT().QueueSizeChanged(gomock.Any()).AnyTimes()
	metrics.EXPECT().WorkerActiveChanged(gomock.Any()).AnyTimes()
	metrics.EXPECT().RequestCancelled(gomock.Any()).AnyTimes()
	metrics.EXPECT().RequestFinished(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
}

func newStaleTrip() model.StaleTrip {
	return model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
}

func shutdownWorker(t *testing.T, worker *WorkerPool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()
	if err := worker.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown worker: %v", err)
	}
}

func waitFor(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()
	select {
	case <-signal:
	case <-ctx.Done():
		t.Fatal("timed out waiting for signal")
	}
}
