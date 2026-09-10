package main

import (
	"os"
	"testing"
)

func TestGetDBURI_DirectBypass(t *testing.T) {
	// Save existing env
	origDirectHost := os.Getenv("DB_DIRECT_HOST")
	origDirectPort := os.Getenv("DB_DIRECT_PORT")
	origHost := os.Getenv("DB_HOST")
	origPort := os.Getenv("DB_PORT")
	origUser := os.Getenv("DB_USER")
	origPass := os.Getenv("DB_PASSWORD")
	origName := os.Getenv("DB_NAME")
	origSSL := os.Getenv("DB_SSLMODE")
	defer func() {
		os.Setenv("DB_DIRECT_HOST", origDirectHost)
		os.Setenv("DB_DIRECT_PORT", origDirectPort)
		os.Setenv("DB_HOST", origHost)
		os.Setenv("DB_PORT", origPort)
		os.Setenv("DB_USER", origUser)
		os.Setenv("DB_PASSWORD", origPass)
		os.Setenv("DB_NAME", origName)
		os.Setenv("DB_SSLMODE", origSSL)
	}()

	os.Setenv("DB_USER", "test_user")
	os.Setenv("DB_PASSWORD", "test_pass")
	os.Setenv("DB_NAME", "test_db")
	os.Setenv("DB_SSLMODE", "disable")

	// Scenario 1: DB_DIRECT_HOST and DB_DIRECT_PORT set (Staging/Prod Docker bypass)
	os.Setenv("DB_HOST", "pgbouncer")
	os.Setenv("DB_PORT", "6432")
	os.Setenv("DB_DIRECT_HOST", "postgres")
	os.Setenv("DB_DIRECT_PORT", "5432")

	uri := getDBURI()
	expected := "postgres://test_user:test_pass@postgres:5432/test_db?sslmode=disable"
	if uri != expected {
		t.Errorf("expected %q, got %q", expected, uri)
	}

	// Scenario 2: Only DB_DIRECT_PORT set (Dev native bypass, DB_DIRECT_PORT=15432, DB_HOST=127.0.0.1)
	os.Setenv("DB_DIRECT_HOST", "")
	os.Setenv("DB_DIRECT_PORT", "15432")
	os.Setenv("DB_HOST", "127.0.0.1")
	os.Setenv("DB_PORT", "10433")

	uri = getDBURI()
	expected = "postgres://test_user:test_pass@127.0.0.1:15432/test_db?sslmode=disable"
	if uri != expected {
		t.Errorf("expected %q, got %q", expected, uri)
	}

	// Scenario 3: Neither direct env set -> falls back to DB_HOST / DB_PORT
	os.Setenv("DB_DIRECT_HOST", "")
	os.Setenv("DB_DIRECT_PORT", "")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5432")

	uri = getDBURI()
	expected = "postgres://test_user:test_pass@db.example.com:5432/test_db?sslmode=disable"
	if uri != expected {
		t.Errorf("expected %q, got %q", expected, uri)
	}
}
