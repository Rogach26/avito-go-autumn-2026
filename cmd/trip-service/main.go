package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rogach26/avito-go-autumn-2026/internal/config"
	"github.com/Rogach26/avito-go-autumn-2026/internal/httpapi"
	"github.com/Rogach26/avito-go-autumn-2026/internal/service"
	"github.com/Rogach26/avito-go-autumn-2026/internal/storage/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})).With("service", "trip-service")
	slog.SetDefault(logger)

	appContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(appContext, cfg)
	if err != nil {
		return err
	}
	poolIsClosed := false
	defer func() {
		if !poolIsClosed {
			pool.Close()
		}
	}()

	repository := postgres.NewTripRepository(pool, cfg.DatabaseQueryTimeout)
	txManager := postgres.NewTransactionManager(pool, cfg.DatabaseQueryTimeout)
	tripService := service.NewTripService(repository, txManager)
	handler := httpapi.NewHandler(tripService, pool, cfg.DatabaseQueryTimeout, logger)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(handler),
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	logger.Info("service started", "http_addr", cfg.HTTPAddr)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-appContext.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(appContext), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("shutdown timeout exceeded", "timeout", cfg.ShutdownTimeout, "error", err)
		os.Exit(1)
	}

	poolCloseDone := make(chan struct{})
	go func() {
		pool.Close()
		close(poolCloseDone)
	}()

	select {
	case <-poolCloseDone:
		poolIsClosed = true
	case <-shutdownContext.Done():
		logger.Error("shutdown timeout exceeded", "timeout", cfg.ShutdownTimeout, "error", shutdownContext.Err())
		os.Exit(1)
	}

	logger.Info("service stopped")
	return nil
}
