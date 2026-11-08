package middleware

import (
	"fmt"
	"net/http"
	"reflect"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracerProvider trace.TracerProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		spanHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state, _ := r.Context().Value(traceStateKey{}).(*traceState)
			if state != nil {
				state.ctx = r.Context()
			}
			defer func() {
				recovered := recover()
				pattern := routePattern(r)
				span := trace.SpanFromContext(r.Context())
				span.SetName(r.Method + " " + pattern)
				span.SetAttributes(attribute.String("http.route", pattern))
				if recovered != nil {
					panicError := fmt.Errorf("panic: %v", recovered)
					span.RecordError(panicError)
					span.SetStatus(codes.Error, http.StatusText(http.StatusInternalServerError))
					span.SetAttributes(attribute.String("error.type", reflect.TypeOf(recovered).String()))
					panic(recovered)
				}
			}()
			next.ServeHTTP(w, r)
		})
		otelHandler := otelhttp.NewHandler(
			spanHandler,
			"HTTP request",
			otelhttp.WithTracerProvider(tracerProvider),
			otelhttp.WithMeterProvider(noop.NewMeterProvider()),
		)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attachTraceState(r)
			otelHandler.ServeHTTP(w, r)
		})
	}
}
