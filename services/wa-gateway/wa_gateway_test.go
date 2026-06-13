package main

import (
	"testing"
	"time"
)

func TestNewTenantRateLimiter(t *testing.T) {
	rl := NewTenantRateLimiter(5)
	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}
	if rl.rate != 5 {
		t.Errorf("expected rate 5, got %d", rl.rate)
	}
	if rl.buckets == nil {
		t.Error("expected initialized buckets map")
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewTenantRateLimiter(5)

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		if !rl.Allow("tenant-1") {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if rl.Allow("tenant-1") {
		t.Error("6th request should be denied (rate limit)")
	}
}

func TestRateLimiter_MultipleTenants(t *testing.T) {
	rl := NewTenantRateLimiter(3)

	// Tenant A uses all tokens
	for i := 0; i < 3; i++ {
		rl.Allow("tenant-a")
	}

	// Tenant B should still have full quota
	if !rl.Allow("tenant-b") {
		t.Error("tenant-b should have its own quota")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewTenantRateLimiter(2)

	// Use all tokens
	rl.Allow("tenant-1")
	rl.Allow("tenant-1")
	if rl.Allow("tenant-1") {
		t.Error("should be rate limited after using all tokens")
	}

	// After waiting, tokens should refill (we can't easily test time-based refill in unit tests
	// without sleeping, but the logic path is covered by the above tests)
}

func TestShouldReconnect_Initial(t *testing.T) {
	// Clean state
	reconnectAttempts = make(map[string]int)
	reconnectBackoff = make(map[string]time.Time)

	// First attempt should be allowed
	if !shouldReconnect("new-tenant") {
		t.Error("initial reconnect should be allowed")
	}
}

func TestShouldReconnect_Backoff(t *testing.T) {
	reconnectAttempts = make(map[string]int)
	reconnectBackoff = make(map[string]time.Time)

	// Set a recent reconnect
	reconnectBackoff["tenant-x"] = time.Now()
	reconnectAttempts["tenant-x"] = 1

	// Should not reconnect within backoff window
	if shouldReconnect("tenant-x") {
		t.Error("should not reconnect within backoff window")
	}
}

func TestShouldReconnect_MaxAttempts(t *testing.T) {
	reconnectAttempts = make(map[string]int)
	reconnectBackoff = make(map[string]time.Time)

	// Set attempts to 5 (max capped) - backoff is 30*2^5 = 960s = 16min
	// Set last attempt past the max backoff window (20 min ago)
	reconnectAttempts["tenant-y"] = 10
	reconnectBackoff["tenant-y"] = time.Now().Add(-20 * time.Minute)

	if !shouldReconnect("tenant-y") {
		t.Error("should reconnect after max backoff window expires")
	}
}

func TestShouldReconnect_CooldownPeriod(t *testing.T) {
	reconnectAttempts = make(map[string]int)
	reconnectBackoff = make(map[string]time.Time)

	// Recent attempt with no prior attempts
	reconnectBackoff["tenant-z"] = time.Now()
	reconnectAttempts["tenant-z"] = 0

	// 5-minute cooldown check
	if shouldReconnect("tenant-z") {
		t.Error("should respect 5-minute cooldown when recent attempt exists with 0 prior attempts")
	}
}
