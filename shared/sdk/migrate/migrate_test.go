package migrate

import (
	"testing"
)

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		filename      string
		wantVersion   int
		wantName      string
		wantDirection string
		wantErr       bool
	}{
		{"000001_init.up.sql", 1, "init", "up", false},
		{"000001_init.down.sql", 1, "init", "down", false},
		{"000025_add_voucher_system.up.sql", 25, "add_voucher_system", "up", false},
		{"000025_add_voucher_system.down.sql", 25, "add_voucher_system", "down", false},
		{"000000_bootstrap.up.sql", 0, "bootstrap", "up", false},
		// Invalid cases
		{"not_a_migration.sql", 0, "", "", true},
		{"0001_name.txt", 0, "", "", true},
		{"name.up.sql", 0, "", "", true},
		{"0001name.up.sql", 0, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			version, name, direction, err := parseMigrationFilename(tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.filename)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if version != tt.wantVersion {
				t.Errorf("version: got %d, want %d", version, tt.wantVersion)
			}
			if name != tt.wantName {
				t.Errorf("name: got %q, want %q", name, tt.wantName)
			}
			if direction != tt.wantDirection {
				t.Errorf("direction: got %q, want %q", direction, tt.wantDirection)
			}
		})
	}
}

func TestParseMigrationFilename_Negative(t *testing.T) {
	_, _, _, err := parseMigrationFilename("0001_invalid.txt")
	if err == nil {
		t.Error("expected error for non-sql file")
	}
	_, _, _, err = parseMigrationFilename("not_a_number_name.up.sql")
	if err == nil {
		t.Error("expected error for invalid version number")
	}
	_, err = NewRunnerFromPath(nil, "/nonexistent")
	if err == nil {
		t.Error("expected error for nil db")
	}
}

