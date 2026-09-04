package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	envName             = "LIVEGROW_ENV"
	httpAddrName        = "LIVEGROW_HTTP_ADDR"
	shutdownTimeoutName = "LIVEGROW_SHUTDOWN_TIMEOUT"
	logLevelName        = "LIVEGROW_LOG_LEVEL"
)

type Config struct {
	Environment     string
	HTTPAddr        string
	ShutdownTimeout time.Duration
	LogLevel        slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		Environment:     getEnv(envName, "dev"),
		HTTPAddr:        getEnv(httpAddrName, ":8080"),
		ShutdownTimeout: 10 * time.Second,
		LogLevel:        slog.LevelInfo,
	}

	if raw := strings.TrimSpace(os.Getenv(shutdownTimeoutName)); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("%s must be a duration: %w", shutdownTimeoutName, err)
		}
		cfg.ShutdownTimeout = d
	}
	if raw := strings.TrimSpace(os.Getenv(logLevelName)); raw != "" {
		level, err := ParseLogLevel(raw)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", logLevelName, err)
		}
		cfg.LogLevel = level
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Environment) == "" {
		return fmt.Errorf("environment must not be empty")
	}
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return fmt.Errorf("http address must not be empty")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}
	if c.ShutdownTimeout > 5*time.Minute {
		return fmt.Errorf("shutdown timeout must not exceed 5m")
	}
	return nil
}

func ParseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q", raw)
	}
}

func getEnv(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
