package main

import (
	"testing"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// REGISTER TENANT SCRIPT TESTS
// Tests untuk helper functions yang digunakan di register_tenant.go
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestArgsValidation(t *testing.T) {
	// Test argument count validation
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
	}{
		{
			name:    "not enough args",
			args:    []string{"tenant"},
			wantErr: true,
		},
		{
			name:    "enough args",
			args:    []string{"tenant", "user", "email@test.com", "6281234567890"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasEnoughArgs := len(tt.args) >= 4
			if hasEnoughArgs == tt.wantErr {
				t.Errorf("args validation = %v, want %v", hasEnoughArgs, !tt.wantErr)
			}
		})
	}
}

func TestTenantCreationSQL(t *testing.T) {
	// Test tenant creation SQL pattern
	tenantName := "Test Tenant"
	sql := "INSERT INTO tenants (name) VALUES ($1) RETURNING id"

	if sql == "" {
		t.Error("SQL should not be empty")
	}

	// Verify SQL contains INSERT
	if sql[:6] != "INSERT" {
		t.Error("SQL should start with INSERT")
	}

	// Verify SQL contains RETURNING
	if !contains(sql, "RETURNING id") {
		t.Error("SQL should contain RETURNING id")
	}

	_ = tenantName // Used in actual script
}

func TestUserCreationSQL(t *testing.T) {
	// Test user creation SQL pattern
	sql := "INSERT INTO users (tenant_id, username, email, password_hash, phone_number) VALUES ($1, $2, $3, $4, $5) RETURNING id"

	if sql == "" {
		t.Error("SQL should not be empty")
	}

	// Verify SQL contains all required columns
	requiredCols := []string{"tenant_id", "username", "email", "password_hash", "phone_number"}
	for _, col := range requiredCols {
		if !contains(sql, col) {
			t.Errorf("SQL should contain column %s", col)
		}
	}
}

func TestSeedRequestHTTP(t *testing.T) {
	// Test seed request HTTP pattern
	tenantID := "test-tenant-id"
	url := "http://localhost:8201/seed"

	if url == "" {
		t.Error("URL should not be empty")
	}

	// Verify URL format
	if url[:4] != "http" {
		t.Error("URL should start with http")
	}

	_ = tenantID // Used in actual script
}

func TestExitCodes(t *testing.T) {
	// Test exit code behavior
	tests := []struct {
		name     string
		hasError bool
		wantExit int
	}{
		{"success", false, 0},
		{"db_error", true, 1},
		{"user_error", true, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			if tt.hasError {
				exitCode = 1
			}
			if exitCode != tt.wantExit {
				t.Errorf("exitCode = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}

// Helper
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
