package config

import (
	"os"
	"testing"
)

func TestLoadConfigDefault(t *testing.T) {
	os.Clearenv()
	
	cfg := LoadConfig(".env.not_exist")
	
	if cfg.Env != "development" {
		t.Errorf("Expected Env to be development, got %s", cfg.Env)
	}
	if cfg.DB.Host != "127.0.0.1" {
		t.Errorf("Expected DB.Host to be 127.0.0.1, got %s", cfg.DB.Host)
	}
	if cfg.Redis.Host != "127.0.0.1" {
		t.Errorf("Expected Redis.Host to be 127.0.0.1, got %s", cfg.Redis.Host)
	}
}

func TestLoadConfigWithEnvVars(t *testing.T) {
	os.Setenv("ENV", "production")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5432")
	
	cfg := LoadConfig(".env.not_exist")
	
	if cfg.Env != "production" {
		t.Errorf("Expected Env to be production, got %s", cfg.Env)
	}
	if cfg.DB.Host != "db.example.com" {
		t.Errorf("Expected DB.Host to be db.example.com, got %s", cfg.DB.Host)
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("Expected DB.Port to be 5432, got %d", cfg.DB.Port)
	}
}
