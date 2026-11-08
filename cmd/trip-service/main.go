package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rogach26/avito-go-autumn-2026/internal/config"
	"github.com/Rogach26/avito-go-autumn-2026/internal/container"
)

func main() {
	if err := run(); err != nil {
		slog.New(slog.NewJSONHandler(os.Stdout, nil)).Error(
			"service stopped",
			"service", "trip-service",
			"trace_id", "",
			"span_id", "",
			"error", err,
		)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	appContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := container.New(appContext, cfg)
	if err != nil {
		return fmt.Errorf("create application container: %w", err)
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Run()
	}()

	var runError error
	select {
	case runError = <-serverErrors:
	case <-appContext.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.WithoutCancel(appContext), cfg.ShutdownTimeout)
	defer cancel()
	shutdownError := app.Shutdown(shutdownContext)

	return errors.Join(runError, shutdownError)
}
