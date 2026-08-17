package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMigrationsFromPath_Valid(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "000001_init.up.sql"), []byte("CREATE TABLE test (id INT);"), 0644)
	os.WriteFile(filepath.Join(dir, "000001_init.down.sql"), []byte("DROP TABLE test;"), 0644)
	os.WriteFile(filepath.Join(dir, "000002_add_col.up.sql"), []byte("ALTER TABLE test ADD col TEXT;"), 0644)

	migrations, err := loadMigrationsFromPath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migrations) != 2 {
		t.Errorf("expected 2 migrations, got %d", len(migrations))
	}
	if migrations[0].Version != 1 {
		t.Errorf("expected version 1 first, got %d", migrations[0].Version)
	}
	if migrations[0].DownSQL == "" {
		t.Error("expected DownSQL to be set")
	}
	if migrations[1].Version != 2 {
		t.Errorf("expected version 2 second, got %d", migrations[1].Version)
	}
}

func TestLoadMigrationsFromPath_InvalidPath(t *testing.T) {
	_, err := loadMigrationsFromPath("/nonexistent/path/to/migrations")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestLoadMigrationsFromPath_SkipsNonSQL(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "000001_init.up.sql"), []byte("CREATE TABLE test (id INT);"), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Migrations"), 0644)
	os.WriteFile(filepath.Join(dir, "not_a_migration.sql"), []byte("SELECT 1;"), 0644) // invalid name

	migrations, err := loadMigrationsFromPath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migrations) != 1 {
		t.Errorf("expected 1 valid migration, got %d", len(migrations))
	}
}

func TestLoadMigrationsFromPath_SkipsSubdir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "000001_init.up.sql"), []byte("CREATE TABLE test (id INT);"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "subdir", "000002_sub.up.sql"), []byte("SELECT 1;"), 0644)

	migrations, err := loadMigrationsFromPath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migrations) != 1 {
		t.Errorf("expected 1 migration (subdirs skipped), got %d", len(migrations))
	}
}

func TestLoadMigrationsFromPath_MissingUpSQL(t *testing.T) {
	dir := t.TempDir()
	// Only down.sql, no up.sql → migration should be skipped (logged as warning)
	os.WriteFile(filepath.Join(dir, "000001_init.down.sql"), []byte("DROP TABLE test;"), 0644)

	migrations, err := loadMigrationsFromPath(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(migrations) != 0 {
		t.Errorf("expected 0 migrations (missing up.sql), got %d", len(migrations))
	}
}

func TestNewRunnerFromPath_ValidPath(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "000001_init.up.sql"), []byte("CREATE TABLE test (id INT);"), 0644)

	// nil db is accepted — error only surfaces when Up()/Down() is called
	runner, err := NewRunnerFromPath(nil, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner == nil {
		t.Error("expected non-nil runner")
	}
}

func TestNewRunnerFromPath_InvalidPath(t *testing.T) {
	_, err := NewRunnerFromPath(nil, "/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}
