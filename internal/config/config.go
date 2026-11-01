package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr                string
	LogLevel                slog.Level
	ShutdownTimeout         time.Duration
	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	HTTPReadTimeout         time.Duration
	HTTPReadHeaderTimeout   time.Duration
	HTTPWriteTimeout        time.Duration
	HTTPIdleTimeout         time.Duration
}

func Load() (Config, error) {
	httpAddr, err := readRequiredStringEnv("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}
	logLevel, err := readLogLevelEnv("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := readDurationEnv("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	databaseURL, err := readRequiredStringEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	databaseMaxConns, err := readInt32Env("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}
	databaseMinConns, err := readInt32Env("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}
	if databaseMaxConns <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
	}
	if databaseMinConns < 0 {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must not be negative")
	}
	if databaseMinConns > databaseMaxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}
	databaseMaxConnLifetime, err := readDurationEnv("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}
	databaseConnectTimeout, err := readDurationEnv("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	databaseQueryTimeout, err := readDurationEnv("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	httpReadTimeout, err := readDurationEnv("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	httpReadHeaderTimeout, err := readDurationEnv("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	httpWriteTimeout, err := readDurationEnv("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	httpIdleTimeout, err := readDurationEnv("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:                httpAddr,
		LogLevel:                logLevel,
		ShutdownTimeout:         shutdownTimeout,
		DatabaseURL:             databaseURL,
		DatabaseMaxConns:        databaseMaxConns,
		DatabaseMinConns:        databaseMinConns,
		DatabaseMaxConnLifetime: databaseMaxConnLifetime,
		DatabaseConnectTimeout:  databaseConnectTimeout,
		DatabaseQueryTimeout:    databaseQueryTimeout,
		HTTPReadTimeout:         httpReadTimeout,
		HTTPReadHeaderTimeout:   httpReadHeaderTimeout,
		HTTPWriteTimeout:        httpWriteTimeout,
		HTTPIdleTimeout:         httpIdleTimeout,
	}, nil
}

func readRequiredStringEnv(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func readDurationEnv(name string) (time.Duration, error) {
	value, err := readRequiredStringEnv(name)
	if err != nil {
		return 0, err
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return parsed, nil
}

func readInt32Env(name string) (int32, error) {
	value, err := readRequiredStringEnv(name)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return int32(parsed), nil
}

func readLogLevelEnv(name string) (slog.Level, error) {
	value, err := readRequiredStringEnv(name)
	if err != nil {
		return 0, err
	}
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("parse %s: unsupported level %q", name, value)
	}
}
