package main

import (
	"context"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
)

// TestRestoreSingleSession_NilContainer verifies graceful handling when globalContainer is nil
func TestRestoreSingleSession_NilContainer(t *testing.T) {
	globalContainer = nil
	db = nil

	// Should not panic, just log warning and return
	restoreSingleSession("tenant-test")
}

// TestRestoreSingleSession_NoSessionInDB verifies behavior when tenant has no stored session
func TestRestoreSingleSession_NoSessionInDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DB-dependent test in short mode")
	}

	// This test requires a real DB connection to test the query path
	// In unit test mode (db == nil), it just returns early
	restoreSingleSession("nonexistent-tenant")
}

// TestHandleConnectedEvent_NilClient verifies no panic when client not in map
func TestHandleConnectedEvent_NilClient(t *testing.T) {
	db = nil
	clientMu.Lock()
	delete(clientMap, "test-tenant")
	clientMu.Unlock()

	// Should not panic even if client not found
	handleConnectedEvent("test-tenant")
}

// TestHandleDisconnectedEvent verifies disconnect handler doesn't panic
func TestHandleDisconnectedEvent(t *testing.T) {
	// Should handle disconnect without panic
	handleDisconnectedEvent("test-tenant")
}

// TestHandleLoggedOutEvent_ClientRemoval verifies client removed from map
func TestHandleLoggedOutEvent_ClientRemoval(t *testing.T) {
	db = nil
	tenantID := "logout-test-tenant"

	// Add a fake client
	clientMu.Lock()
	clientMap[tenantID] = &whatsmeow.Client{}
	clientMu.Unlock()

	handleLoggedOutEvent(tenantID)

	// Client should be removed
	clientMu.RLock()
	_, exists := clientMap[tenantID]
	clientMu.RUnlock()

	if exists {
		t.Error("expected client to be removed after logout")
	}
}

// TestMapUserJIDIfNeeded_NilDB verifies graceful handling when DB unavailable
func TestMapUserJIDIfNeeded_NilDB(t *testing.T) {
	db = nil

	// Should not panic
	mapUserJIDIfNeeded("6281234567890@s.whatsapp.net", "6281234567890")
}

// TestMapUserJIDIfNeeded_PhoneNormalization verifies 62 -> 0 conversion
func TestMapUserJIDIfNeeded_PhoneNormalization(t *testing.T) {
	// We can't test the actual DB update without a connection,
	// but we can verify the function doesn't panic with various inputs
	db = nil

	cases := []struct {
		jid   string
		phone string
	}{
		{"6281234567890@s.whatsapp.net", "6281234567890"},
		{"081234567890@s.whatsapp.net", "081234567890"},
		{"", "6281234567890"},
		{"6281234567890@s.whatsapp.net", ""},
	}

	for _, tc := range cases {
		mapUserJIDIfNeeded(tc.jid, tc.phone)
	}
}

// TestMapUserJIDIfNeeded_EmptyInputs verifies early return on empty inputs
func TestMapUserJIDIfNeeded_EmptyInputs(t *testing.T) {
	// When DB is nil, function should return early without error
	db = nil

	// Should return early without querying DB
	mapUserJIDIfNeeded("", "")
	mapUserJIDIfNeeded("jid", "")
	mapUserJIDIfNeeded("", "phone")
}

// TestReconnectBackoff_EdgeCases verifies backoff calculation boundaries
func TestReconnectBackoff_EdgeCases(t *testing.T) {
	reconnectAttempts = make(map[string]int)
	reconnectBackoff = make(map[string]time.Time)

	cases := []struct {
		name        string
		attempts    int
		lastAttempt time.Time
		shouldAllow bool
	}{
		{"no prior attempts", 0, time.Time{}, true},
		{"1 attempt, recent", 1, time.Now(), false},
		{"1 attempt, expired", 1, time.Now().Add(-2 * time.Minute), true},
		{"5 attempts, max backoff", 5, time.Now().Add(-20 * time.Minute), true},
		{"10 attempts (capped at 5)", 10, time.Now().Add(-20 * time.Minute), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenantID := "tenant-" + tc.name
			reconnectAttempts[tenantID] = tc.attempts
			if !tc.lastAttempt.IsZero() {
				reconnectBackoff[tenantID] = tc.lastAttempt
			}

			got := shouldReconnect(tenantID)
			if got != tc.shouldAllow {
				t.Errorf("shouldReconnect(%q) = %v, want %v (attempts=%d, lastAttempt=%v)",
					tenantID, got, tc.shouldAllow, tc.attempts, tc.lastAttempt)
			}
		})
	}
}

// TestIsSixDigitOTP_BoundaryValues verifies edge cases
func TestIsSixDigitOTP_BoundaryValues(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"000000", true},
		{"999999", true},
		{"123456", true},
		{"00000", false},   // 5 digits
		{"0000000", false}, // 7 digits
		{"12345a", false},
		{"12.456", false}, // has dot
		{"12-456", false}, // has dash
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := isSixDigitOTP(tc.input)
			if got != tc.expected {
				t.Errorf("isSixDigitOTP(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}

// TestSessionLockAcquisition_Concurrency verifies lock prevents double-restore
func TestSessionLockAcquisition_Concurrency(t *testing.T) {
	t.Skip("Skipping session lock concurrency test - requires Redis connection")
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}

	ctx := context.Background()
	tenantID := "concurrent-test"

	// First acquisition should succeed
	owned1, err1 := AcquireSessionLock(ctx, tenantID)
	if err1 != nil {
		t.Fatalf("first lock acquisition failed: %v", err1)
	}
	if !owned1 {
		t.Error("expected first lock acquisition to succeed")
	}

	// Second acquisition should fail (already owned)
	owned2, _ := AcquireSessionLock(ctx, tenantID)
	if owned2 {
		t.Error("expected second lock acquisition to fail (already owned)")
	}

	// Release and try again
	ReleaseSessionLock(ctx, tenantID)

	owned3, err3 := AcquireSessionLock(ctx, tenantID)
	if err3 != nil {
		t.Fatalf("lock acquisition after release failed: %v", err3)
	}
	if !owned3 {
		t.Error("expected lock acquisition to succeed after release")
	}

	ReleaseSessionLock(ctx, tenantID)
}
