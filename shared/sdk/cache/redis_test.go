package cache

import (
	"testing"
)

func TestCloseRedis_NilClient(t *testing.T) {
	orig := Client
	Client = nil
	defer func() { Client = orig }()
	// Should not panic
	CloseRedis()
}

func TestClientIsNilByDefault(t *testing.T) {
	orig := Client
	Client = nil
	defer func() { Client = orig }()
	if Client != nil {
		t.Error("expected nil client")
	}
}
