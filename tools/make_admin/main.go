package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <username>")
		fmt.Println("Example: go run main.go owner_kopi")
		os.Exit(1)
	}

	username := os.Args[1]

	// Load configuration
	cfg := config.LoadConfig("../../.env")

	dbUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

	// Connect to database
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	// Update user role to superadmin
	cmdTag, err := dbpool.Exec(ctx, "UPDATE users SET role = 'superadmin' WHERE username = $1", username)
	if err != nil {
		log.Fatalf("Failed to update user: %v\n", err)
	}

	if cmdTag.RowsAffected() == 0 {
		fmt.Printf("User '%s' not found.\n", username)
		os.Exit(1)
	}

	fmt.Printf("✅ Success! User '%s' is now a superadmin.\n", username)
}
