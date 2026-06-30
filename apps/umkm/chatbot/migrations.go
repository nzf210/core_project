package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"core_project/shared/sdk/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// runMigrations executes all pending database migrations
// Migrations are loaded from filesystem path (not embedded)
func runMigrations(db *pgxpool.Pool) error {
	// Get migrations directory path relative to working directory
	migrationsDir := "shared/migrations"

	// Try paths from different working directory depths:
	// - services/* run from services/*/ -> ../../shared/migrations
	// - apps/* run from apps/*/ -> ../../../shared/migrations
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = "../../shared/migrations"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			migrationsDir = "../../../shared/migrations"
			if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
				return fmt.Errorf("migrations directory not found")
			}
		}
	}

	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to resolve migrations path: %w", err)
	}

	slog.Info("Running database migrations", "path", absPath)

	runner, err := migrate.NewRunnerFromPath(db, absPath)
	if err != nil {
		return err
	}

	if err := runner.Up(context.Background()); err != nil {
		return err
	}

	return nil
}
