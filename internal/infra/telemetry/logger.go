package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

type traceHandler struct {
	slog.Handler
}

func NewLogger(level slog.Level, serviceName string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(traceHandler{Handler: handler}).With("service", serviceName)
}

func (h traceHandler) Handle(ctx context.Context, record slog.Record) error {
	spanContext := trace.SpanContextFromContext(ctx)
	record.AddAttrs(
		slog.String("trace_id", spanContext.TraceID().String()),
		slog.String("span_id", spanContext.SpanID().String()),
	)
	return h.Handler.Handle(ctx, record)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{Handler: h.Handler.WithGroup(name)}
}
