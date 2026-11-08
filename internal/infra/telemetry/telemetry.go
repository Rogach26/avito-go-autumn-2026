package telemetry

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/Rogach26/avito-go-autumn-2026/internal/config"
)

type Telemetry struct {
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
}

func New(ctx context.Context, cfg config.Config) (*Telemetry, error) {
	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.OTelServiceName),
			semconv.ServiceVersion(buildVersion()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create telemetry resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpointURL(strings.TrimRight(cfg.OTelExporterEndpoint, "/")+"/v1/traces"),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	metricExporter, err := otlpmetrichttp.New(
		ctx,
		otlpmetrichttp.WithEndpointURL(strings.TrimRight(cfg.OTelExporterEndpoint, "/")+"/v1/metrics"),
	)
	if err != nil {
		if shutdownErr := traceExporter.Shutdown(ctx); shutdownErr != nil {
			return nil, errors.Join(
				fmt.Errorf("create metric exporter: %w", err),
				fmt.Errorf("shutdown trace exporter: %w", shutdownErr),
			)
		}
		return nil, fmt.Errorf("create metric exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
	)
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		metric.WithResource(res),
	)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &Telemetry{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
	}, nil
}

func (t *Telemetry) TracerProvider() *trace.TracerProvider {
	return t.tracerProvider
}

func (t *Telemetry) MeterProvider() *metric.MeterProvider {
	return t.meterProvider
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(
		t.meterProvider.Shutdown(ctx),
		t.tracerProvider.Shutdown(ctx),
	)
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "dev"
	}
	return info.Main.Version
}
