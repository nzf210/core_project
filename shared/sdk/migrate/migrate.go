package migrate

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration represents a single database migration file
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

// Runner handles database migrations
type Runner struct {
	db         *pgxpool.Pool
	migrations []Migration
}

// NewRunner creates a new migration runner
func NewRunner(db *pgxpool.Pool, migrationsFS embed.FS, migrationsDir string) (*Runner, error) {
	migrations, err := loadMigrations(migrationsFS, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	return &Runner{
		db:         db,
		migrations: migrations,
	}, nil
}

// NewRunnerFromPath creates a new migration runner from filesystem path
func NewRunnerFromPath(db *pgxpool.Pool, migrationsPath string) (*Runner, error) {
	migrations, err := loadMigrationsFromPath(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	return &Runner{
		db:         db,
		migrations: migrations,
	}, nil
}

// Up runs all pending migrations
func (r *Runner) Up(ctx context.Context) error {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	appliedVersions, err := r.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	pendingCount := 0
	for _, m := range r.migrations {
		if appliedVersions[m.Version] {
			continue
		}

		slog.Info("Running migration", "version", m.Version, "name", m.Name)

		tx, err := r.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", m.Version, err)
		}

		// Execute migration SQL
		if _, err := tx.Exec(ctx, m.UpSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %d (%s): %w", m.Version, m.Name, err)
		}

		// Record migration
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)",
			m.Version, m.Name, time.Now().UTC(),
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.Version, err)
		}

		slog.Info("Migration applied", "version", m.Version, "name", m.Name)
		pendingCount++
	}

	if pendingCount == 0 {
		slog.Info("No pending migrations")
	} else {
		slog.Info("Migrations completed", "applied", pendingCount)
	}

	return nil
}

// Down rolls back the last N migrations
func (r *Runner) Down(ctx context.Context, steps int) error {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	appliedVersions, err := r.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	// Get applied migrations in reverse order
	var toRollback []Migration
	for i := len(r.migrations) - 1; i >= 0 && len(toRollback) < steps; i-- {
		m := r.migrations[i]
		if appliedVersions[m.Version] {
			toRollback = append(toRollback, m)
		}
	}

	if len(toRollback) == 0 {
		slog.Info("No migrations to rollback")
		return nil
	}

	for _, m := range toRollback {
		slog.Info("Rolling back migration", "version", m.Version, "name", m.Name)

		tx, err := r.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for rollback %d: %w", m.Version, err)
		}

		// Execute down SQL
		if _, err := tx.Exec(ctx, m.DownSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to rollback migration %d (%s): %w", m.Version, m.Name, err)
		}

		// Remove migration record
		if _, err := tx.Exec(ctx,
			"DELETE FROM schema_migrations WHERE version = $1",
			m.Version,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to remove migration record %d: %w", m.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit rollback %d: %w", m.Version, err)
		}

		slog.Info("Migration rolled back", "version", m.Version, "name", m.Name)
	}

	return nil
}

// Status returns current migration status
func (r *Runner) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	appliedVersions, err := r.getAppliedVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied versions: %w", err)
	}

	var statuses []MigrationStatus
	for _, m := range r.migrations {
		status := MigrationStatus{
			Version: m.Version,
			Name:    m.Name,
			Applied: appliedVersions[m.Version],
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version int
	Name    string
	Applied bool
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist
func (r *Runner) ensureMigrationsTable(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

// getAppliedVersions returns a map of applied migration versions
func (r *Runner) getAppliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := r.db.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// loadMigrations loads all migration files from embedded FS
func loadMigrations(migrationsFS embed.FS, migrationsDir string) ([]Migration, error) {
	entries, err := migrationsFS.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrationsMap := make(map[int]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		version, migrationName, direction, err := parseMigrationFilename(name)
		if err != nil {
			slog.Warn("Skipping invalid migration file", "file", name, "error", err)
			continue
		}

		content, err := migrationsFS.ReadFile(migrationsDir + "/" + name)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		if migrationsMap[version] == nil {
			migrationsMap[version] = &Migration{
				Version: version,
				Name:    migrationName,
			}
		}

		if direction == "up" {
			migrationsMap[version].UpSQL = string(content)
		} else {
			migrationsMap[version].DownSQL = string(content)
		}
	}

	// Convert map to sorted slice
	var migrations []Migration
	for _, m := range migrationsMap {
		if m.UpSQL == "" {
			slog.Warn("Migration missing up.sql", "version", m.Version, "name", m.Name)
			continue
		}
		migrations = append(migrations, *m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFilename parses migration filename format: 000001_name.up.sql
func parseMigrationFilename(filename string) (version int, name string, direction string, err error) {
	// Remove .sql extension
	filename = strings.TrimSuffix(filename, ".sql")

	// Split by last dot to get direction
	parts := strings.Split(filename, ".")
	if len(parts) != 2 {
		return 0, "", "", fmt.Errorf("invalid format: expected format 000001_name.up.sql")
	}

	direction = parts[1]
	if direction != "up" && direction != "down" {
		return 0, "", "", fmt.Errorf("invalid direction: must be 'up' or 'down'")
	}

	// Split by underscore to get version and name
	nameParts := strings.SplitN(parts[0], "_", 2)
	if len(nameParts) != 2 {
		return 0, "", "", fmt.Errorf("invalid format: expected 000001_name")
	}

	_, err = fmt.Sscanf(nameParts[0], "%d", &version)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid version number: %w", err)
	}

	name = nameParts[1]

	return version, name, direction, nil
}

// loadMigrationsFromPath loads all migration files from filesystem path
func loadMigrationsFromPath(migrationsPath string) ([]Migration, error) {
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrationsMap := make(map[int]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		version, migrationName, direction, err := parseMigrationFilename(name)
		if err != nil {
			slog.Warn("Skipping invalid migration file", "file", name, "error", err)
			continue
		}

		content, err := os.ReadFile(migrationsPath + "/" + name)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		if migrationsMap[version] == nil {
			migrationsMap[version] = &Migration{
				Version: version,
				Name:    migrationName,
			}
		}

		if direction == "up" {
			migrationsMap[version].UpSQL = string(content)
		} else {
			migrationsMap[version].DownSQL = string(content)
		}
	}

	// Convert map to sorted slice
	var migrations []Migration
	for _, m := range migrationsMap {
		if m.UpSQL == "" {
			slog.Warn("Migration missing up.sql", "version", m.Version, "name", m.Name)
			continue
		}
		migrations = append(migrations, *m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}
