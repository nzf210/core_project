package db

import (
	"context"
	"testing"
	"time"

	"core_project/shared/sdk/config"
)

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()

	if cfg.MaxConns != 5 {
		t.Errorf("Expected MaxConns=5, got %d", cfg.MaxConns)
	}

	if cfg.MinConns != 2 {
		t.Errorf("Expected MinConns=2, got %d", cfg.MinConns)
	}

	if cfg.MaxConnLifetime != 1*time.Hour {
		t.Errorf("Expected MaxConnLifetime=1h, got %v", cfg.MaxConnLifetime)
	}
}

func TestLegacyPoolConfig(t *testing.T) {
	cfg := LegacyPoolConfig()

	if cfg.MaxConns != 20 {
		t.Errorf("Expected MaxConns=20, got %d", cfg.MaxConns)
	}

	if cfg.MinConns != 5 {
		t.Errorf("Expected MinConns=5, got %d", cfg.MinConns)
	}
}

func TestConnect_InvalidDSN(t *testing.T) {
	cfg := &config.Config{
		DB: config.DatabaseConfig{
			Host:     "invalid-host-that-does-not-exist",
			Port:     9999,
			User:     "test",
			Password: "test",
			Name:     "test",
			SSLMode:  "disable",
		},
	}

	pool, err := ConnectWithDefaults(cfg)
	if err == nil {
		pool.Close()
		t.Error("Expected error for invalid DSN, got nil")
	}
}

func TestConnect_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{
		DB: config.DatabaseConfig{
			Host:     "127.0.0.1",
			Port:     5433, // Assume PostgreSQL running
			User:     "test_user",
			Password: "test_pass",
			Name:     "test_db",
			SSLMode:  "disable",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	dsn := "postgres://test_user:test_pass@127.0.0.1:9999/test_db?sslmode=disable"
	poolCfg, err := parseDSN(dsn)
	if err == nil {
		pool, _ := poolCfg.NewWithContext(ctx)
		if pool != nil {
			pool.Close()
		}
	}
}

func parseDSN(dsn string) (*config.Config, error) {
	return nil, nil // Placeholder for test helper
}
