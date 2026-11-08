package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			writer := &statusWriter{ResponseWriter: w}
			defer func() {
				recovered := recover()
				status := writer.statusCode()
				if recovered != nil {
					status = http.StatusInternalServerError
				}
				level := slog.LevelInfo
				if status >= http.StatusInternalServerError {
					level = slog.LevelError
				}
				logger.LogAttrs(
					r.Context(),
					level,
					"request handled",
					slog.String("method", r.Method),
					slog.String("route", routePattern(r)),
					slog.Int("status", status),
					slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
				)
				if recovered != nil {
					panic(recovered)
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
