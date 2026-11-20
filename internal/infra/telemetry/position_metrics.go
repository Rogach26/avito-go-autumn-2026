package telemetry

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type PositionMetrics struct {
	saved        metric.Int64Counter
	staleChecks  metric.Int64Counter
	requests     metric.Int64Counter
	duration     metric.Float64Histogram
	queueSize    atomic.Int64
	activeWorker atomic.Int64
}

func NewPositionMetrics(meter metric.Meter) (*PositionMetrics, error) {
	saved, err := meter.Int64Counter("tripgo.positions.saved", metric.WithUnit("{position}"))
	if err != nil {
		return nil, fmt.Errorf("create positions saved counter: %w", err)
	}
	staleChecks, err := meter.Int64Counter("tripgo.stale_checks", metric.WithUnit("{check}"))
	if err != nil {
		return nil, fmt.Errorf("create stale checks counter: %w", err)
	}
	queueSize, err := meter.Int64ObservableGauge("tripgo.position_requests.queue.size", metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("create position queue size gauge: %w", err)
	}
	activeWorkers, err := meter.Int64ObservableGauge("tripgo.position_requests.workers.active", metric.WithUnit("{worker}"))
	if err != nil {
		return nil, fmt.Errorf("create active position workers gauge: %w", err)
	}
	requests, err := meter.Int64Counter("tripgo.position_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("create position requests counter: %w", err)
	}
	duration, err := meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("create position request duration histogram: %w", err)
	}

	metrics := &PositionMetrics{
		saved:       saved,
		staleChecks: staleChecks,
		requests:    requests,
		duration:    duration,
	}
	if _, err := meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		observer.ObserveInt64(queueSize, metrics.queueSize.Load())
		observer.ObserveInt64(activeWorkers, metrics.activeWorker.Load())
		return nil
	}, queueSize, activeWorkers); err != nil {
		return nil, fmt.Errorf("register position worker gauges: %w", err)
	}

	return metrics, nil
}

func (m *PositionMetrics) Saved(ctx context.Context) {
	m.saved.Add(ctx, 1)
}

func (m *PositionMetrics) StaleCheck(ctx context.Context, result string) {
	m.staleChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (m *PositionMetrics) QueueSizeChanged(delta int64) {
	m.queueSize.Add(delta)
}

func (m *PositionMetrics) WorkerActiveChanged(delta int64) {
	m.activeWorker.Add(delta)
}

func (m *PositionMetrics) RequestCancelled(ctx context.Context) {
	m.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "cancelled")))
}

func (m *PositionMetrics) RequestFinished(ctx context.Context, result string, elapsed time.Duration) {
	attributes := metric.WithAttributes(attribute.String("result", result))
	m.requests.Add(ctx, 1, attributes)
	m.duration.Record(ctx, elapsed.Seconds(), attributes)
}
