package repository

import (
	"context"
	"fmt"
	"log/slog"

	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitDB(cfg *config.Config) error {
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
	slog.Info("✅ Connected to PostgreSQL (Campaign API via Repository)")
	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

// GetHierarchyCTE returns a Recursive CTE to fetch all descendant user IDs for RBAC.
func GetHierarchyCTE(paramIndex int) string {
	return fmt.Sprintf(`
	WITH RECURSIVE subordinates AS (
		SELECT id FROM users WHERE id = $%d
		UNION
		SELECT u.id FROM users u INNER JOIN subordinates s ON s.id = u.parent_id
	)
	`, paramIndex)
}
