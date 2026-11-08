package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/middleware"
)

func NewRouter(server *Server, logger *slog.Logger, tracerProvider trace.TracerProvider, metrics middleware.MetricsRecorder) http.Handler {
	router := chi.NewRouter()
	resolveRoute := func(r *http.Request) string {
		routeContext := chi.NewRouteContext()
		if router.Match(routeContext, r.Method, r.URL.Path) {
			if pattern := routeContext.RoutePattern(); pattern != "" {
				return pattern
			}
		}
		return "unmatched"
	}

	router.Use(middleware.Recover(logger))
	router.Use(middleware.Tracing(tracerProvider))
	router.Use(middleware.Logging(logger))
	router.Use(middleware.Metrics(metrics, resolveRoute))

	return api.HandlerWithOptions(server, api.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: server.parameterError,
	})
}
