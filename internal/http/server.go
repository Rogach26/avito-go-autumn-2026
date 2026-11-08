package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/domainerror"
	"github.com/Rogach26/avito-go-autumn-2026/internal/http/response"
	createhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/create"
	finishhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/finish"
	gethttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/get"
)

type Server struct {
	createHandler *createhttp.Handler
	getHandler    *gethttp.Handler
	finishHandler *finishhttp.Handler
	database      databasePinger
	queryTimeout  time.Duration
}

func NewServer(
	createHandler *createhttp.Handler,
	getHandler *gethttp.Handler,
	finishHandler *finishhttp.Handler,
	database databasePinger,
	queryTimeout time.Duration,
) *Server {
	return &Server{
		createHandler: createHandler,
		getHandler:    getHandler,
		finishHandler: finishHandler,
		database:      database,
		queryTimeout:  queryTimeout,
	}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.createHandler.Handle(w, r, params)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.getHandler.Handle(w, r, id)
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.finishHandler.Handle(w, r, id)
}

func (s *Server) Health(w http.ResponseWriter, _ *http.Request) {
	response.JSON(w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	if err := s.database.Ping(ctx); err != nil {
		response.JSON(w, http.StatusServiceUnavailable, "application/json", api.HealthResponse{Status: api.Unavailable})
		return
	}
	response.JSON(w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func (s *Server) parameterError(w http.ResponseWriter, r *http.Request, err error) {
	domainerror.WriteInvalidRequest(w, r, fmt.Errorf("request parameters: %w", err))
}
