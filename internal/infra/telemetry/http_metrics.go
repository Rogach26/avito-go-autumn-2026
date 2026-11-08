package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HTTPMetrics struct {
	duration metric.Float64Histogram
	active   metric.Int64UpDownCounter
}

func NewHTTPMetrics(meter metric.Meter) (*HTTPMetrics, error) {
	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request duration histogram: %w", err)
	}
	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create active HTTP requests counter: %w", err)
	}

	return &HTTPMetrics{duration: duration, active: active}, nil
}

func (m *HTTPMetrics) RequestStarted(ctx context.Context, method, route string) {
	m.active.Add(ctx, 1, metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
	))
}

func (m *HTTPMetrics) RequestFinished(
	ctx context.Context,
	method string,
	route string,
	status int,
	duration time.Duration,
) {
	m.active.Add(ctx, -1, metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
	))
	m.duration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.response.status_code", status),
	))
}
