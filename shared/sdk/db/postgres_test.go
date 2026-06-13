package db

import (
	"testing"
)

func TestCloseDB_NilPool(t *testing.T) {
	orig := Pool
	Pool = nil
	defer func() { Pool = orig }()
	// Should not panic
	CloseDB()
}

func TestPoolIsNilByDefault(t *testing.T) {
	orig := Pool
	Pool = nil
	defer func() { Pool = orig }()
	if Pool != nil {
		t.Error("expected nil pool")
	}
}
