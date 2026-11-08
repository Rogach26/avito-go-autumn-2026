package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/semconv/v1.40.0/dbconv"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/Rogach26/avito-go-autumn-2026/internal/infra/postgres"

type Tracer struct {
	*otelpgx.Tracer
	operationDuration dbconv.ClientOperationDuration
}

type queryTrace struct {
	startedAt  time.Time
	operation  string
	collection string
}

type queryTraceKey struct{}

func NewTracer(tracerProvider trace.TracerProvider, meterProvider metric.MeterProvider) (*Tracer, error) {
	operationDuration, err := dbconv.NewClientOperationDuration(meterProvider.Meter(instrumentationName))
	if err != nil {
		return nil, err
	}

	return &Tracer{
		Tracer: otelpgx.NewTracer(
			otelpgx.WithTracerProvider(tracerProvider),
			otelpgx.WithMeterProvider(metricnoop.NewMeterProvider()),
		),
		operationDuration: operationDuration,
	}, nil
}

func (t *Tracer) TraceQueryStart(
	ctx context.Context,
	conn *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	query := queryTrace{
		startedAt:  time.Now(),
		operation:  otelpgx.SQLOperationName(ctx, data.SQL),
		collection: collectionName(data.SQL),
	}
	ctx = t.Tracer.TraceQueryStart(ctx, conn, data)

	attributes := queryAttributes(query)
	trace.SpanFromContext(ctx).SetAttributes(attributes...)

	return context.WithValue(ctx, queryTraceKey{}, query)
}

func (t *Tracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	query, ok := ctx.Value(queryTraceKey{}).(queryTrace)
	if ok {
		attributes := queryAttributes(query)
		if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
			errorAttribute := databaseErrorType(data.Err)
			attributes = append(attributes, errorAttribute)
			trace.SpanFromContext(ctx).SetAttributes(errorAttribute)
		}
		t.operationDuration.Record(
			ctx,
			time.Since(query.startedAt).Seconds(),
			dbconv.SystemNamePostgreSQL,
			attributes...,
		)
	}

	t.Tracer.TraceQueryEnd(ctx, conn, data)
}

func queryAttributes(query queryTrace) []attribute.KeyValue {
	attributes := make([]attribute.KeyValue, 0, 2)
	if query.operation != "" {
		attributes = append(attributes, semconv.DBOperationName(query.operation))
	}
	if query.collection != "" {
		attributes = append(attributes, semconv.DBCollectionName(query.collection))
	}
	return attributes
}

func databaseErrorType(err error) attribute.KeyValue {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		return semconv.ErrorTypeKey.String(postgresError.Code)
	}
	return semconv.ErrorType(err)
}

func collectionName(query string) string {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return ""
	}

	switch strings.ToUpper(fields[0]) {
	case "INSERT":
		return identifierAfter(fields, "INTO")
	case "SELECT", "DELETE":
		return identifierAfter(fields, "FROM")
	case "UPDATE":
		if len(fields) > 1 {
			return normalizeIdentifier(fields[1])
		}
	}

	return ""
}

func identifierAfter(fields []string, keyword string) string {
	for index := range len(fields) - 1 {
		if strings.EqualFold(fields[index], keyword) {
			return normalizeIdentifier(fields[index+1])
		}
	}
	return ""
}

func normalizeIdentifier(value string) string {
	value = strings.Trim(value, "\"'`;,()")
	if separator := strings.LastIndexByte(value, '.'); separator >= 0 {
		value = value[separator+1:]
	}
	return strings.Trim(value, "\"")
}
