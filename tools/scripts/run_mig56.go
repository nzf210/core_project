package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := "postgres://postgres:postgres@localhost:5433/wch_core?sslmode=disable"
	if u := os.Getenv("DATABASE_URL"); u != "" {
		dbURL = u
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}
	defer pool.Close()

	bUp, err := os.ReadFile("shared/migrations/000056_affiliate_and_leaderboard.up.sql")
	if err != nil {
		log.Fatalf("Failed to read up.sql: %v", err)
	}

	_, err = pool.Exec(context.Background(), string(bUp))
	if err != nil {
		log.Fatalf("Failed to execute up.sql: %v", err)
	}

	_, err = pool.Exec(context.Background(), "INSERT INTO schema_migrations (version, name) VALUES (56, 'affiliate_and_leaderboard') ON CONFLICT DO NOTHING")
	if err != nil {
		log.Fatalf("Failed to update schema_migrations: %v", err)
	}

	fmt.Println("Migration 56 applied successfully!")
}