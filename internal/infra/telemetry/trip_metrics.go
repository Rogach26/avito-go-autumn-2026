package telemetry

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/otel/metric"
)

type TripMetrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
	active    atomic.Int64
}

func NewTripMetrics(meter metric.Meter, activeTrips int64) (*TripMetrics, error) {
	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, fmt.Errorf("create trips created counter: %w", err)
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, fmt.Errorf("create trips completed counter: %w", err)
	}
	active, err := meter.Int64ObservableGauge("tripgo.trips.active")
	if err != nil {
		return nil, fmt.Errorf("create active trips gauge: %w", err)
	}

	metrics := &TripMetrics{created: created, completed: completed}
	metrics.active.Store(activeTrips)
	if _, err := meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		observer.ObserveInt64(active, metrics.active.Load())
		return nil
	}, active); err != nil {
		return nil, fmt.Errorf("register active trips gauge: %w", err)
	}

	return metrics, nil
}

func (m *TripMetrics) Created(ctx context.Context) {
	m.created.Add(ctx, 1)
	m.active.Add(1)
}

func (m *TripMetrics) Completed(ctx context.Context) {
	m.completed.Add(ctx, 1)
	m.active.Add(-1)
}
