package db

import (
	"testing"

	"core_project/shared/sdk/config"
)

func TestInitDB_InvalidDSN(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.Host = "invalid-host-that-does-not-exist"
	cfg.DB.Port = 9999
	cfg.DB.User = "baduser"
	cfg.DB.Password = "badpass"
	cfg.DB.Name = "baddb"
	cfg.DB.SSLMode = "disable"

	err := InitDB(cfg)
	if err == nil {
		t.Error("expected error for unreachable DB host")
	}
}

func TestCloseDB_WithNilPool(t *testing.T) {
	orig := Pool
	Pool = nil
	defer func() { Pool = orig }()
	// Should not panic
	CloseDB()
}
