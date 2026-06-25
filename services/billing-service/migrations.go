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

// runMigrations executes all pending database migrations
// Migrations are loaded from filesystem path (not embedded)
func runMigrations(db *pgxpool.Pool) error {
	// Get migrations directory path relative to working directory
	migrationsDir := "shared/migrations"

	// Check if running from service directory (e.g., services/billing-service)
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try from project root
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

	runner, err := migrate.NewRunnerFromPath(db, absPath)
	if err != nil {
		return err
	}

	if err := runner.Up(context.Background()); err != nil {
		return err
	}

	return nil
}
