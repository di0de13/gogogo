package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, name := range []string{envName, httpAddrName, shutdownTimeoutName, logLevelName} {
		t.Setenv(name, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Environment != "dev" || cfg.HTTPAddr != ":8080" || cfg.ShutdownTimeout != 10*time.Second || cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv(envName, "test")
	t.Setenv(httpAddrName, "127.0.0.1:9000")
	t.Setenv(shutdownTimeoutName, "3s")
	t.Setenv(logLevelName, "debug")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Environment != "test" || cfg.HTTPAddr != "127.0.0.1:9000" || cfg.ShutdownTimeout != 3*time.Second || cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	t.Setenv(shutdownTimeoutName, "later")
	if _, err := Load(); err == nil {
		t.Fatal("Load() expected an error")
	}
}

func TestValidateRejectsLongShutdown(t *testing.T) {
	cfg := Config{Environment: "test", HTTPAddr: ":1", ShutdownTimeout: 6 * time.Minute}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() expected an error")
	}
}
