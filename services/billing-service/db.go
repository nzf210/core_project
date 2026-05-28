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
	queries := []string{
		`CREATE TABLE IF NOT EXISTS plans (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			price_idr DECIMAL(15, 2) NOT NULL,
			max_bots INT DEFAULT 1,
			max_ai_requests INT DEFAULT 100,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO plans (id, name, price_idr, max_bots, max_ai_requests) VALUES
		('free', 'Free Tier', 0, 1, 5),
		('lite', 'Lite', 150000, 3, 500),
		('pro', 'Pro', 450000, 10, 5000)
		ON CONFLICT DO NOTHING`,
		`CREATE TABLE IF NOT EXISTS tenant_subscriptions (
			tenant_id VARCHAR(100) PRIMARY KEY,
			plan_id VARCHAR(50) REFERENCES plans(id),
			status VARCHAR(20) DEFAULT 'active',
			current_period_end TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS invoices (
			id VARCHAR(100) PRIMARY KEY,
			tenant_id VARCHAR(100) NOT NULL,
			plan_id VARCHAR(50) NOT NULL,
			amount DECIMAL(15,2) NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			payment_url VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			paid_at TIMESTAMP
		)`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed to run schema query: %w", err)
		}
	}
	slog.Info("✅ Database schema verified for Billing")
	return nil
}
