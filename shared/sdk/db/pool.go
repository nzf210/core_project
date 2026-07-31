package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"core_project/shared/sdk/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds database connection pool configuration
type PoolConfig struct {
	MaxConns          int32         // Maximum connections in pool
	MinConns          int32         // Minimum idle connections
	MaxConnLifetime   time.Duration // Maximum connection lifetime
	MaxConnIdleTime   time.Duration // Maximum idle time before closing
	HealthCheckPeriod time.Duration // Health check interval
}

// DefaultPoolConfig returns optimized pool settings for PgBouncer deployment
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          5,              // Small pool per service (PgBouncer handles pooling)
		MinConns:          2,              // Keep 2 connections warm
		MaxConnLifetime:   1 * time.Hour,  // Recycle connections hourly
		MaxConnIdleTime:   10 * time.Minute, // Close idle after 10min
		HealthCheckPeriod: 30 * time.Second, // Health check every 30s
	}
}

// LegacyPoolConfig returns larger pool settings for direct PostgreSQL connection
func LegacyPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConns:          20,             // Larger pool for direct connection
		MinConns:          5,              // More idle connections
		MaxConnLifetime:   2 * time.Hour,
		MaxConnIdleTime:   15 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// Connect creates a connection pool with optimized settings
func Connect(cfg *config.Config, poolCfg PoolConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DSN: %w", err)
	}

	// Apply pool configuration
	config.MaxConns = poolCfg.MaxConns
	config.MinConns = poolCfg.MinConns
	config.MaxConnLifetime = poolCfg.MaxConnLifetime
	config.MaxConnIdleTime = poolCfg.MaxConnIdleTime
	config.HealthCheckPeriod = poolCfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	// Log pool configuration for observability
	slog.Info("✅ Connected to PostgreSQL",
		"host", cfg.DB.Host,
		"port", cfg.DB.Port,
		"database", cfg.DB.Name,
		"max_conns", poolCfg.MaxConns,
		"min_conns", poolCfg.MinConns,
	)

	return pool, nil
}

// ConnectWithDefaults creates a connection pool with default PgBouncer-optimized settings
func ConnectWithDefaults(cfg *config.Config) (*pgxpool.Pool, error) {
	return Connect(cfg, DefaultPoolConfig())
}
