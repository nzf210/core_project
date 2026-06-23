package main

import (
	"context"
	"fmt"
	"log/slog"

	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

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
	slog.Info("✅ Connected to PostgreSQL for Billing")
	return nil
}

func ensureSchema() error {
	ctx := context.Background()

	// Only create legacy tables if they don't exist.
	// SaaS v2 tables (saas_plans, plan_features, voucher_programs, etc.)
	// are managed via migration 000025.
	queries := []string{
		`CREATE TABLE IF NOT EXISTS invoices (
			id VARCHAR(100) PRIMARY KEY,
			tenant_id VARCHAR(100) NOT NULL,
			plan_id VARCHAR(50) NOT NULL,
			amount BIGINT NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			payment_url VARCHAR(500),
			voucher_code VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			paid_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS tenant_subscriptions (
			tenant_id VARCHAR(100) PRIMARY KEY,
			plan_id VARCHAR(20) NOT NULL,
			status VARCHAR(20) DEFAULT 'active',
			current_period_end TIMESTAMP,
			plan_tier VARCHAR(20) DEFAULT 'lite',
			period_days INT DEFAULT 30,
			ticket_id UUID,
			voucher_code_id UUID,
			activated_by VARCHAR(50) DEFAULT 'payment',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(ctx, q); err != nil {
			slog.Warn("Schema query warning (may already exist from migration)", "query", q[:50], "error", err)
		}
	}
	slog.Info("✅ Database schema verified for Billing")
	return nil
}
