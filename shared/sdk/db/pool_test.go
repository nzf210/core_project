package db

import (
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
	cfg := &config.Config{}
	cfg.DB.Host = "invalid-host-that-does-not-exist"
	cfg.DB.Port = 9999
	cfg.DB.User = "test"
	cfg.DB.Password = "test"
	cfg.DB.Name = "test"
	cfg.DB.SSLMode = "disable"

	pool, err := ConnectWithDefaults(cfg)
	if err == nil {
		pool.Close()
		t.Error("Expected error for invalid DSN, got nil")
	}
}

func TestConnect_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &config.Config{}
	cfg.DB.Host = "127.0.0.1"
	cfg.DB.Port = 9999
	cfg.DB.User = "test_user"
	cfg.DB.Password = "test_pass"
	cfg.DB.Name = "test_db"
	cfg.DB.SSLMode = "disable"

	pool, err := ConnectWithDefaults(cfg)
	if err == nil {
		pool.Close()
		t.Error("Expected error for unreachable host, got nil")
	}
}
