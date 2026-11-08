package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Rogach26/avito-go-autumn-2026/internal/http/domainerror"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(tracedContext(r), "request panicked", "method", r.Method, "path", r.URL.Path, "panic", recovered, "stack", string(debug.Stack()))
					domainerror.WriteInternal(w, r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
