package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"core_project/shared/sdk/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run register_tenant.go <tenant_name> <username> <email> <phone_number>")
		os.Exit(1)
	}

	tenantName := os.Args[1]
	username := os.Args[2]
	email := os.Args[3]
	phone := os.Args[4]

	cfg := config.LoadConfig("../../../../.env")
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode)

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Create Tenant
	var tenantID string
	err = db.QueryRow(ctx, "INSERT INTO tenants (name) VALUES ($1) RETURNING id", tenantName).Scan(&tenantID)
	if err != nil {
		fmt.Printf("Failed to create tenant: %v\n", err)
		os.Exit(1)
	}

	// 2. Create User
	// Mock password hash for now
	passwordHash := "bcrypt_hash_placeholder"
	var userID string
	err = db.QueryRow(ctx, "INSERT INTO users (tenant_id, username, email, password_hash, phone_number) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		tenantID, username, email, passwordHash, phone).Scan(&userID)
	if err != nil {
		fmt.Printf("Failed to create user: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Success!\nTenant ID: %s\nUser ID: %s\n", tenantID, userID)

	// 3. Seed Accounting
	req, _ := http.NewRequest("POST", "http://localhost:8201/seed", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to seed accounting: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Chart of Accounts (COA) seeded successfully for this tenant!")
	} else {
		fmt.Printf("⚠️ Warning: Seed accounting returned status %d\n", resp.StatusCode)
	}
}
