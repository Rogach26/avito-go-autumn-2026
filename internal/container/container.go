package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rogach26/avito-go-autumn-2026/internal/config"
	pushgateway "github.com/Rogach26/avito-go-autumn-2026/internal/gateway/push"
	httpapi "github.com/Rogach26/avito-go-autumn-2026/internal/http"
	createpositionhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/position/create"
	listpositionhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/position/list"
	createhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/create"
	finishhttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/finish"
	gethttp "github.com/Rogach26/avito-go-autumn-2026/internal/http/trip/get"
	"github.com/Rogach26/avito-go-autumn-2026/internal/infra/postgres"
	"github.com/Rogach26/avito-go-autumn-2026/internal/infra/postgres/transaction"
	"github.com/Rogach26/avito-go-autumn-2026/internal/infra/telemetry"
	triprepository "github.com/Rogach26/avito-go-autumn-2026/internal/repository/trip"
	createposition "github.com/Rogach26/avito-go-autumn-2026/internal/usecase/position/create"
	listposition "github.com/Rogach26/avito-go-autumn-2026/internal/usecase/position/list"
	createusecase "github.com/Rogach26/avito-go-autumn-2026/internal/usecase/trip/create"
	finishusecase "github.com/Rogach26/avito-go-autumn-2026/internal/usecase/trip/finish"
	getusecase "github.com/Rogach26/avito-go-autumn-2026/internal/usecase/trip/get"
	positionworker "github.com/Rogach26/avito-go-autumn-2026/internal/worker/positions"
)

type Container struct {
	server    *http.Server
	pool      *pgxpool.Pool
	push      *pushgateway.Client
	positions *positionworker.WorkerPool
	telemetry *telemetry.Telemetry
	logger    *slog.Logger
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	logger := telemetry.NewLogger(cfg.LogLevel, cfg.OTelServiceName)
	telemetryProviders, err := telemetry.New(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize telemetry: %w", err)
	}

	databaseTracer, err := postgres.NewTracer(
		telemetryProviders.TracerProvider(),
		telemetryProviders.MeterProvider(),
	)
	if err != nil {
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}
	pool, err := postgres.NewPool(ctx, cfg, databaseTracer)
	if err != nil {
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}

	repository := triprepository.NewRepository(pool, cfg.DatabaseQueryTimeout)
	activeTrips, err := repository.CountActive(ctx)
	if err != nil {
		pool.Close()
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}

	meter := telemetryProviders.MeterProvider().Meter(cfg.OTelServiceName)
	tripMetrics, err := telemetry.NewTripMetrics(meter, activeTrips)
	if err != nil {
		pool.Close()
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}
	httpMetrics, err := telemetry.NewHTTPMetrics(meter)
	if err != nil {
		pool.Close()
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}
	positionMetrics, err := telemetry.NewPositionMetrics(meter)
	if err != nil {
		pool.Close()
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}

	pushClient, err := pushgateway.New(cfg.PushHTTPURL, cfg.PushRequestTimeout, telemetryProviders.TracerProvider())
	if err != nil {
		pool.Close()
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}
	positionWorker, err := positionworker.New(
		repository,
		pushClient,
		positionMetrics,
		logger,
		telemetryProviders.TracerProvider(),
		cfg.PositionCheckInterval,
		cfg.PositionStaleAfter,
		cfg.PushRequestTimeout,
		cfg.PositionWorkers,
		cfg.PositionQueueSize,
	)
	if err != nil {
		pushClient.CloseIdleConnections()
		pool.Close()
		return nil, errors.Join(err, telemetryProviders.Shutdown(ctx))
	}

	txManager := transaction.NewManager(pool, cfg.DatabaseQueryTimeout, logger)
	createUseCase := createusecase.New(repository, txManager, tripMetrics, logger)
	getUseCase := getusecase.New(repository)
	finishUseCase := finishusecase.New(repository, txManager, tripMetrics, logger)
	createPositionUseCase := createposition.New(repository, txManager, positionMetrics, positionWorker)
	listPositionUseCase := listposition.New(repository)

	serverHandler := httpapi.NewServer(
		createhttp.NewHandler(createUseCase, logger),
		gethttp.NewHandler(getUseCase, logger),
		finishhttp.NewHandler(finishUseCase, logger),
		createpositionhttp.NewHandler(createPositionUseCase, logger),
		listpositionhttp.NewHandler(listPositionUseCase, logger),
		pool,
		cfg.DatabaseQueryTimeout,
	)
	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewRouter(
			serverHandler,
			logger,
			telemetryProviders.TracerProvider(),
			httpMetrics,
		),
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	return &Container{
		server:    server,
		pool:      pool,
		push:      pushClient,
		positions: positionWorker,
		telemetry: telemetryProviders,
		logger:    logger,
	}, nil
}

func (c *Container) Start(ctx context.Context) {
	c.positions.Start(ctx)
}

func (c *Container) Run() error {
	c.logger.Info("service started", "http_addr", c.server.Addr)
	err := c.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("serve HTTP: %w", err)
}

func (c *Container) Shutdown(ctx context.Context) error {
	var shutdownError error
	if err := c.server.Shutdown(ctx); err != nil {
		shutdownError = fmt.Errorf("shutdown HTTP server: %w", err)
		if closeErr := c.server.Close(); closeErr != nil {
			shutdownError = errors.Join(shutdownError, fmt.Errorf("close HTTP server: %w", closeErr))
		}
	}
	if err := c.positions.Shutdown(ctx); err != nil {
		shutdownError = errors.Join(shutdownError, fmt.Errorf("shutdown position worker: %w", err))
	}
	c.push.CloseIdleConnections()

	poolClosed := make(chan struct{})
	go func() {
		c.pool.Close()
		close(poolClosed)
	}()
	select {
	case <-poolClosed:
	case <-ctx.Done():
		shutdownError = errors.Join(shutdownError, fmt.Errorf("close database pool: %w", ctx.Err()))
	}

	if err := c.telemetry.Shutdown(ctx); err != nil {
		shutdownError = errors.Join(shutdownError, fmt.Errorf("shutdown telemetry: %w", err))
	}
	if shutdownError == nil {
		c.logger.Info("service stopped")
	}
	return shutdownError
}
