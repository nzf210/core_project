package cache

import (
	"testing"

	"core_project/shared/sdk/config"
	"github.com/redis/go-redis/v9"
)

func TestInitRedis_UnreachableHost(t *testing.T) {
	cfg := &config.Config{}
	cfg.Redis.Host = "127.0.0.1"
	cfg.Redis.Port = 1 // unreachable
	cfg.Redis.Password = ""

	err := InitRedis(cfg)
	if err == nil {
		t.Error("expected error for unreachable Redis host")
	}
	// Restore
	Client = nil
}

func TestCloseRedis_WithClient(t *testing.T) {
	// Create a real client object (not connected) just to test the Close path
	orig := Client
	Client = redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { Client = orig }()
	// Should not panic even if connection was never established
	CloseRedis()
}
