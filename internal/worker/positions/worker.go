package positions

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Rogach26/avito-go-autumn-2026/internal/model"
)

const (
	staleCheckQueued    = "queued"
	staleCheckDuplicate = "duplicate"
	staleCheckDropped   = "dropped"
	requestSuccess      = "success"
	requestError        = "error"
)

type task struct {
	trip        model.StaleTrip
	token       uint64
	scanContext trace.SpanContext
}

type WorkerPool struct {
	repository     staleTripRepository
	pushGateway    pushGateway
	metrics        positionMetrics
	logger         *slog.Logger
	tracer         trace.Tracer
	tickerFactory  tickerFactory
	checkInterval  time.Duration
	staleAfter     time.Duration
	requestTimeout time.Duration
	workerCount    int
	queue          chan task

	pendingMu sync.Mutex
	pending   map[uuid.UUID]uint64
	nextToken uint64

	scannerCancel context.CancelFunc
	poolCancel    context.CancelFunc
	scannerDone   chan struct{}
	workersDone   chan struct{}
}

func New(
	repository staleTripRepository,
	pushGateway pushGateway,
	metrics positionMetrics,
	logger *slog.Logger,
	tracerProvider trace.TracerProvider,
	checkInterval time.Duration,
	staleAfter time.Duration,
	requestTimeout time.Duration,
	workerCount int,
	queueSize int,
) (*WorkerPool, error) {
	return newWorkerPool(
		repository,
		pushGateway,
		metrics,
		logger,
		tracerProvider,
		systemTickerFactory{},
		checkInterval,
		staleAfter,
		requestTimeout,
		workerCount,
		queueSize,
	)
}

func newWorkerPool(
	repository staleTripRepository,
	pushGateway pushGateway,
	metrics positionMetrics,
	logger *slog.Logger,
	tracerProvider trace.TracerProvider,
	tickerFactory tickerFactory,
	checkInterval time.Duration,
	staleAfter time.Duration,
	requestTimeout time.Duration,
	workerCount int,
	queueSize int,
) (*WorkerPool, error) {
	if checkInterval <= 0 || staleAfter <= 0 || requestTimeout <= 0 {
		return nil, fmt.Errorf("position worker durations must be greater than zero")
	}
	if workerCount <= 0 {
		return nil, fmt.Errorf("position worker count must be greater than zero")
	}
	if queueSize <= 0 {
		return nil, fmt.Errorf("position queue size must be greater than zero")
	}

	return &WorkerPool{
		repository:     repository,
		pushGateway:    pushGateway,
		metrics:        metrics,
		logger:         logger,
		tracer:         tracerProvider.Tracer("trip-service/worker/positions"),
		tickerFactory:  tickerFactory,
		checkInterval:  checkInterval,
		staleAfter:     staleAfter,
		requestTimeout: requestTimeout,
		workerCount:    workerCount,
		queue:          make(chan task, queueSize),
		pending:        make(map[uuid.UUID]uint64),
	}, nil
}

func (wp *WorkerPool) Start(ctx context.Context) {
	poolContext, poolCancel := context.WithCancel(context.WithoutCancel(ctx))
	scannerContext, scannerCancel := context.WithCancel(poolContext)
	wp.poolCancel = poolCancel
	wp.scannerCancel = scannerCancel
	wp.scannerDone = make(chan struct{})
	wp.workersDone = make(chan struct{})

	go wp.scan(scannerContext)

	var workers sync.WaitGroup
	workers.Add(wp.workerCount)
	for range wp.workerCount {
		go func() {
			defer workers.Done()
			wp.process(poolContext)
		}()
	}
	go func() {
		workers.Wait()
		close(wp.workersDone)
	}()

	wp.logger.Info("position worker started", "workers", wp.workerCount, "queue_size", cap(wp.queue))
}

func (wp *WorkerPool) Cancel(tripID uuid.UUID) {
	wp.pendingMu.Lock()
	delete(wp.pending, tripID)
	wp.pendingMu.Unlock()
}

func (wp *WorkerPool) Shutdown(ctx context.Context) error {
	if wp.scannerCancel == nil {
		return nil
	}

	wp.scannerCancel()
	select {
	case <-wp.scannerDone:
	case <-ctx.Done():
		wp.poolCancel()
		<-wp.scannerDone
		close(wp.queue)
		<-wp.workersDone
		return fmt.Errorf("stop position scanner: %w", ctx.Err())
	}

	close(wp.queue)
	select {
	case <-wp.workersDone:
	case <-ctx.Done():
		wp.poolCancel()
		<-wp.workersDone
		return fmt.Errorf("drain position workers: %w", ctx.Err())
	}

	wp.poolCancel()
	wp.logger.Info("position worker stopped")
	return nil
}

func (wp *WorkerPool) scan(ctx context.Context) {
	defer close(wp.scannerDone)
	ticker := wp.tickerFactory.NewTicker(wp.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			wp.scanOnce(ctx)
		}
	}
}

func (wp *WorkerPool) scanOnce(ctx context.Context) {
	scanContext, span := wp.tracer.Start(ctx, "positions.scan")
	defer span.End()

	trips, err := wp.repository.ListStale(scanContext, wp.staleAfter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		wp.logger.ErrorContext(scanContext, "stale trip scan failed", "error", err)
		return
	}

	for _, trip := range trips {
		wp.enqueue(scanContext, trip)
	}
}

func (wp *WorkerPool) enqueue(ctx context.Context, trip model.StaleTrip) bool {
	wp.pendingMu.Lock()
	if _, exists := wp.pending[trip.ID]; exists {
		wp.pendingMu.Unlock()
		wp.metrics.StaleCheck(ctx, staleCheckDuplicate)
		return false
	}

	wp.nextToken++
	queuedTask := task{
		trip:        trip,
		token:       wp.nextToken,
		scanContext: trace.SpanContextFromContext(ctx),
	}
	wp.pending[trip.ID] = queuedTask.token
	select {
	case wp.queue <- queuedTask:
		wp.metrics.QueueSizeChanged(1)
		wp.pendingMu.Unlock()
		wp.metrics.StaleCheck(ctx, staleCheckQueued)
		return true
	default:
		delete(wp.pending, trip.ID)
		wp.pendingMu.Unlock()
		wp.metrics.StaleCheck(ctx, staleCheckDropped)
		wp.logger.WarnContext(ctx, "position queue is full", "trip_id", trip.ID)
		return false
	}
}

func (wp *WorkerPool) process(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case queuedTask, ok := <-wp.queue:
			if !ok {
				return
			}
			pending := wp.isPending(queuedTask)
			wp.metrics.QueueSizeChanged(-1)
			if !pending {
				wp.metrics.RequestCancelled(ctx)
				continue
			}
			wp.processTask(ctx, queuedTask)
		}
	}
}

func (wp *WorkerPool) processTask(ctx context.Context, queuedTask task) {
	defer wp.finish(queuedTask)

	options := []trace.SpanStartOption{trace.WithNewRoot()}
	if queuedTask.scanContext.IsValid() {
		options = append(options, trace.WithLinks(trace.Link{SpanContext: queuedTask.scanContext}))
	}
	taskContext, span := wp.tracer.Start(ctx, "positions.request", options...)
	defer span.End()
	span.SetAttributes(
		attribute.String("trip.id", queuedTask.trip.ID.String()),
		attribute.String("driver.id", queuedTask.trip.DriverID.String()),
	)

	wp.metrics.WorkerActiveChanged(1)
	defer wp.metrics.WorkerActiveChanged(-1)

	startedAt := time.Now()
	requestContext, cancel := context.WithTimeout(taskContext, wp.requestTimeout)
	err := wp.pushGateway.RequestPosition(requestContext, queuedTask.trip.DriverID, queuedTask.trip.ID)
	cancel()

	result := requestSuccess
	if err != nil {
		result = requestError
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		wp.logger.WarnContext(taskContext, "position request failed", "trip_id", queuedTask.trip.ID, "error", err)
	}
	wp.metrics.RequestFinished(taskContext, result, time.Since(startedAt))
}

func (wp *WorkerPool) isPending(queuedTask task) bool {
	wp.pendingMu.Lock()
	defer wp.pendingMu.Unlock()
	return wp.pending[queuedTask.trip.ID] == queuedTask.token
}

func (wp *WorkerPool) finish(queuedTask task) {
	wp.pendingMu.Lock()
	if wp.pending[queuedTask.trip.ID] == queuedTask.token {
		delete(wp.pending, queuedTask.trip.ID)
	}
	wp.pendingMu.Unlock()
}
