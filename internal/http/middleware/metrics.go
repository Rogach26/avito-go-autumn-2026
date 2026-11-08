package middleware

import (
	"net/http"
	"time"
)

type RouteResolver func(r *http.Request) string

func Metrics(recorder MetricsRecorder, resolveRoute RouteResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := resolveRoute(r)
			startedAt := time.Now()
			recorder.RequestStarted(r.Context(), r.Method, route)
			writer := &statusWriter{ResponseWriter: w}
			defer func() {
				recovered := recover()
				status := writer.statusCode()
				if recovered != nil {
					status = http.StatusInternalServerError
				}
				recorder.RequestFinished(r.Context(), r.Method, route, status, time.Since(startedAt))
				if recovered != nil {
					panic(recovered)
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
