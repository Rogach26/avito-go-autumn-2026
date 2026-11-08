package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func routePattern(r *http.Request) string {
	if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
		if pattern := routeContext.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}
