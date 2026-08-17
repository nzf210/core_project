package db

import (
	"testing"

	"core_project/shared/sdk/config"
)

func TestInitDB_BadConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.Host = "127.0.0.1"
	cfg.DB.Port = 1
	cfg.DB.User = "u"
	cfg.DB.Password = "p"
	cfg.DB.Name = "d"
	cfg.DB.SSLMode = "disable"

	err := InitDB(cfg)
	if err == nil {
		t.Error("expected error for unreachable DB")
	}
	// InitDB sets Pool before Ping — use the non-nil Pool to cover CloseDB's live branch
	if Pool != nil {
		CloseDB()
	}
}

