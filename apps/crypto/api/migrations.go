package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	sdkdb "core_project/shared/sdk/db"
	"core_project/shared/sdk/migrate"
)

// runMigrations executes all pending database migrations using shared db.Pool
func runMigrations() error {
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

	runner, err := migrate.NewRunnerFromPath(sdkdb.Pool, absPath)
	if err != nil {
		return err
	}

	return runner.Up(context.Background())
}
