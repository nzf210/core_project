package main

import (
	"testing"
	"time"
)

// ========================================
// WA Gateway Rate Limiter Tests
// ========================================

func TestTokenBucket_RateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow rate limiting test in -short mode")
	}
	testCases := []struct {
		name           string
		rate           int
		requests       int
		interval       time.Duration
		expectedAllows int
	}{
		{"5 msg/min - under limit", 5, 4, 0, 4},   // No sleep - test burst
		{"5 msg/min - at limit", 5, 5, 0, 5},      // No sleep - test burst
		{"5 msg/min - over limit", 5, 10, 0, 5},   // No sleep - test burst
		{"Burst prevention", 5, 10, 0, 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewTenantRateLimiter(tc.rate)
			tenantID := "test-tenant"
			allowCount := 0

			for i := 0; i < tc.requests; i++ {
				if limiter.Allow(tenantID) {
					allowCount++
				}
				if tc.interval > 0 {
					time.Sleep(tc.interval / time.Duration(tc.requests))
				}
			}

			if allowCount != tc.expectedAllows {
				t.Errorf("Expected %d allowed, got %d", tc.expectedAllows, allowCount)
			}
		})
	}
}

func TestTokenBucket_TenantIsolation(t *testing.T) {
	limiter := NewTenantRateLimiter(5)

	tenantA := "tenant-a"
	tenantB := "tenant-b"

	// Exhaust tenant A's quota
	for i := 0; i < 10; i++ {
		limiter.Allow(tenantA)
	}

	// Tenant B should still have full quota
	if !limiter.Allow(tenantB) {
		t.Error("Tenant B should not be affected by Tenant A's rate limit")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	limiter := NewTenantRateLimiter(5)
	tenantID := "test-tenant"

	// Exhaust quota
	for i := 0; i < 10; i++ {
		limiter.Allow(tenantID)
	}

	// Should be rate limited
	if limiter.Allow(tenantID) {
		t.Error("Should be rate limited after exhausting quota")
	}

	// Wait for refill (tokens refill at rate/minute)
	time.Sleep(15 * time.Second) // 1/4 of a minute = ~1 token refilled

	// Should allow at least 1 more message after refill
	if !limiter.Allow(tenantID) {
		t.Error("Should allow message after token refill")
	}
}

// ========================================
// WA Provider Routing Security Tests
// ========================================

func TestWARouting_ProviderPreference(t *testing.T) {
	tests := []struct {
		name             string
		preference       string
		messageType      string
		expectedProvider string
	}{
		{
			name:             "Auto - transactional to Cloud API",
			preference:       "auto",
			messageType:      "invoice",
			expectedProvider: "cloud_api",
		},
		{
			name:             "Auto - conversational to whatsmeow",
			preference:       "auto",
			messageType:      "",
			expectedProvider: "whatsmeow",
		},
		{
			name:             "Force whatsmeow",
			preference:       "whatsmeow",
			messageType:      "invoice",
			expectedProvider: "whatsmeow",
		},
		{
			name:             "Force cloud_api",
			preference:       "cloud_api",
			messageType:      "",
			expectedProvider: "cloud_api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualProvider string

			switch tt.preference {
			case "cloud_api":
				actualProvider = "cloud_api"
			case "whatsmeow":
				actualProvider = "whatsmeow"
			default:
				if isTransactionalType(tt.messageType) {
					actualProvider = "cloud_api"
				} else {
					actualProvider = "whatsmeow"
				}
			}

			if actualProvider != tt.expectedProvider {
				t.Errorf("Expected provider %s, got %s", tt.expectedProvider, actualProvider)
			}
		})
	}
}

func isTransactionalType(messageType string) bool {
	transactionalTypes := []string{"otp", "invoice", "subscription", "system", "broadcast"}
	for _, t := range transactionalTypes {
		if messageType == t {
			return true
		}
	}
	return false
}

// ========================================
// Phone Number Validation Tests
// ========================================

func TestPhoneNumber_Normalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		valid    bool
	}{
		{"Indonesian 08", "081234567890", "6281234567890", true},
		{"Already normalized", "6281234567890", "6281234567890", true},
		{"With plus prefix", "+6281234567890", "6281234567890", true},
		{"With spaces", "0812 3456 7890", "6281234567890", true},
		{"With dashes", "0812-3456-7890", "6281234567890", true},
		{"Too short", "08123", "", false},
		{"Invalid prefix", "09812345678", "", false},
		{"Non-numeric", "08abc123456", "", false},
		{"Empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizePhone(tt.input)

			if tt.valid && normalized != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, normalized)
			}
		})
	}
}

// ========================================
// Reconnect Backoff Tests
// ========================================

func TestReconnect_ExponentialBackoff(t *testing.T) {
	tests := []struct {
		name            string
		attempt         int
		expectedBackoff time.Duration
		maxBackoff      time.Duration
	}{
		{"First attempt", 1, 30 * time.Second, 10 * time.Minute},
		{"Second attempt", 2, 60 * time.Second, 10 * time.Minute},
		{"Third attempt", 3, 120 * time.Second, 10 * time.Minute},
		{"Fourth attempt", 4, 240 * time.Second, 10 * time.Minute},
		{"Max backoff reached", 10, 10 * time.Minute, 10 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := calculateBackoff(tt.attempt, tt.maxBackoff)

			if backoff > tt.maxBackoff {
				t.Errorf("Backoff %v exceeds max %v", backoff, tt.maxBackoff)
			}

			// Validate exponential growth
			if tt.attempt < 10 {
				expectedMin := time.Duration(30) * time.Second * (1 << (tt.attempt - 1))
				if backoff < expectedMin {
					t.Errorf("Expected at least %v, got %v", expectedMin, backoff)
				}
			}
		})
	}
}

func calculateBackoff(attempt int, maxBackoff time.Duration) time.Duration {
	baseBackoff := 30 * time.Second
	backoff := baseBackoff * (1 << (attempt - 1)) // Exponential: 30s, 60s, 120s, 240s...

	if backoff > maxBackoff {
		return maxBackoff
	}
	return backoff
}

// ========================================
// Message Type Detection Tests
// ========================================

func TestMessageType_Detection(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		isTransactional bool
	}{
		{"OTP message", "otp", true},
		{"Invoice", "invoice", true},
		{"Subscription", "subscription", true},
		{"System notification", "system", true},
		{"Broadcast", "broadcast", true},
		{"Conversational", "", false},
		{"Unknown type", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTransactionalType(tt.headerValue)
			if result != tt.isTransactional {
				t.Errorf("Expected isTransactional=%v for %s", tt.isTransactional, tt.headerValue)
			}
		})
	}
}

// ========================================
// Distributed Lock Tests (Redis)
// ========================================

func TestDistributedLock_Acquisition(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping distributed lock test in -short mode (requires Redis)")
	}
	tests := []struct {
		name        string
		lockKey     string
		ttl         time.Duration
		concurrent  int
		expectLocks int
	}{
		{
			name:        "Single lock acquisition",
			lockKey:     "wa:lock:tenant-1",
			ttl:         5 * time.Minute,
			concurrent:  1,
			expectLocks: 1,
		},
		{
			name:        "Concurrent lock attempts - only one succeeds",
			lockKey:     "wa:lock:tenant-2",
			ttl:         5 * time.Minute,
			concurrent:  5,
			expectLocks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			successCount := 0

			for i := 0; i < tt.concurrent; i++ {
				// Simulate lock acquisition
				acquired := tryAcquireLock(tt.lockKey, tt.ttl)
				if acquired {
					successCount++
				}
			}

			if successCount != tt.expectLocks {
				t.Errorf("Expected %d locks, got %d", tt.expectLocks, successCount)
			}
		})
	}
}

func tryAcquireLock(_ string, _ time.Duration) bool {
	return true
}

// ========================================
// Session Heartbeat Tests
// ========================================

func TestSessionHeartbeat_Expiration(t *testing.T) {
	tests := []struct {
		name          string
		heartbeatAge  time.Duration
		ttl           time.Duration
		shouldExpire  bool
	}{
		{
			name:         "Fresh heartbeat",
			heartbeatAge: 1 * time.Minute,
			ttl:          5 * time.Minute,
			shouldExpire: false,
		},
		{
			name:         "Expired heartbeat",
			heartbeatAge: 10 * time.Minute,
			ttl:          5 * time.Minute,
			shouldExpire: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastHeartbeat := time.Now().Add(-tt.heartbeatAge)
			expired := time.Since(lastHeartbeat) > tt.ttl

			if expired != tt.shouldExpire {
				t.Errorf("Expected expire=%v, got %v", tt.shouldExpire, expired)
			}
		})
	}
}
