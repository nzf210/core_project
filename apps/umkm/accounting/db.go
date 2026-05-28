package main

import (
	"context"
	"fmt"
	"log/slog"

	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool
var isTest bool

func initDB(cfg *config.Config) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)
	
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	DB = pool
	slog.Info("✅ Connected to PostgreSQL")

	// Ensure the tenant_automations schema exists
	if err := ensureSchema(); err != nil {
		slog.Error("Failed to ensure tenant_automations schema", "error", err)
	}

	return nil
}

func ensureSchema() error {
	ctx := context.Background()
	query := `
	CREATE TABLE IF NOT EXISTS tenant_automations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
		type VARCHAR(50) NOT NULL,
		name VARCHAR(255) NOT NULL,
		enabled BOOLEAN DEFAULT true,
		cron_expression VARCHAR(100) NOT NULL,
		config JSONB DEFAULT '{}'::jsonb,
		target_wa VARCHAR(50),
		last_run_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_tenant_automations_tenant_id ON tenant_automations(tenant_id);
	`
	if _, err := DB.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create tenant_automations table: %w", err)
	}
	slog.Info("✅ Schema tenant_automations verified")
	return nil
}
