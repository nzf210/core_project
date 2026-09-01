package main

import (
	"context"
	"core_project/shared/sdk/migrate"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

// runMigrations executes all pending database migrations.
// Uses DB_DIRECT_HOST/DB_DIRECT_PORT if set to bypass pgbouncer
// (transaction mode is incompatible with DDL wrapped in explicit transactions).
func runMigrations(db *pgxpool.Pool) error {
	migrationsDir := "shared/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "../../shared/migrations"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			return fmt.Errorf("migrations directory not found")
		}
	}

	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve migrations path: %w", err)
	}

	slog.Info("Running database migrations", "path", absPath)

	migrateDB := db
	directHost := os.Getenv("DB_DIRECT_HOST")
	directPort := os.Getenv("DB_DIRECT_PORT")
	if directHost != "" || directPort != "" {
		host := directHost
		if host == "" {
			host = os.Getenv("DB_HOST")
		}
		if host == "" {
			host = "127.0.0.1"
		}
		port := directPort
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		sslMode := os.Getenv("DB_SSLMODE")
		if sslMode == "" {
			sslMode = "disable"
		}

		dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			user, password, host, port, dbName, sslMode)

		directPool, err := pgxpool.New(context.Background(), dsn)
		if err != nil {
			return fmt.Errorf("failed to connect directly to postgres for migrations: %w", err)
		}
		defer directPool.Close()

		if err := directPool.Ping(context.Background()); err != nil {
			return fmt.Errorf("failed to ping direct postgres for migrations: %w", err)
		}

		slog.Info("Using direct PostgreSQL connection for migrations (bypassing pgbouncer)", "host", host, "port", port)
		migrateDB = directPool
	}

	runner, err := migrate.NewRunnerFromPath(migrateDB, absPath)
	if err != nil {
		return err
	}

	return runner.Up(context.Background())
}
