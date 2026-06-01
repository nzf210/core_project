// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"core_project/shared/sdk/config"
	"core_project/shared/sdk/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.LoadConfig(".env")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to PostgreSQL")

	// Test migration runner
	runner, err := migrate.NewRunnerFromPath(pool, "shared/migrations")
	if err != nil {
		log.Fatalf("Failed to create migration runner: %v", err)
	}

	// Show status
	statuses, err := runner.Status(context.Background())
	if err != nil {
		log.Fatalf("Failed to get migration status: %v", err)
	}

	fmt.Println("\n📋 Migration Status:")
	fmt.Println("Version | Name                          | Applied")
	fmt.Println("--------|-------------------------------|--------")
	for _, s := range statuses {
		applied := "❌"
		if s.Applied {
			applied = "✅"
		}
		fmt.Printf("%7d | %-29s | %s\n", s.Version, s.Name, applied)
	}

	// Ask user confirmation
	fmt.Print("\n🚀 Run pending migrations? (y/N): ")
	var answer string
	fmt.Scanln(&answer)

	if answer != "y" && answer != "Y" {
		fmt.Println("Aborted.")
		os.Exit(0)
	}

	// Run migrations
	if err := runner.Up(context.Background()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("\n✅ All migrations completed successfully!")
}
