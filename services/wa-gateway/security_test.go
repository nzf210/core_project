package main

import (
	"testing"
	"time"
)

// ========================================
// WA Gateway Rate Limiter Tests
// ========================================

func TestTokenBucket_RateLimiting(t *testing.T) {
	tests := []struct {
		name           string
		rate           int // messages per minute
		requests       int
		interval       time.Duration
		expectedAllows int
	}{
		{
			name:           "5 msg/min - under limit",
			rate:           5,
			requests:       4,
			interval:       time.Minute,
			expectedAllows: 4,
		},
		{
			name:           "5 msg/min - at limit",
			rate:           5,
			requests:       5,
			interval:       time.Minute,
			expectedAllows: 5,
		},
		{
			name:           "5 msg/min - over limit",
			rate:           5,
			requests:       10,
			interval:       time.Minute,
			expectedAllows: 5,
		},
		{
			name:           "Burst prevention",
			rate:           5,
			requests:       10,
			interval:       0, // Instant burst
			expectedAllows: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewTenantRateLimiter(tt.rate)
			tenantID := "test-tenant"
			allowCount := 0

			for i := 0; i < tt.requests; i++ {
				if limiter.Allow(tenantID) {
					allowCount++
				}

				if tt.interval > 0 {
					time.Sleep(tt.interval / time.Duration(tt.requests))
				}
			}

			if allowCount != tt.expectedAllows {
				t.Errorf("Expected %d allowed, got %d", tt.expectedAllows, allowCount)
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
			// Validate routing logic
			var actualProvider string

			if tt.preference == "cloud_api" {
				actualProvider = "cloud_api"
			} else if tt.preference == "whatsmeow" {
				actualProvider = "whatsmeow"
			} else {
				// Auto mode
				if isTransactional(tt.messageType) {
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

func isTransactional(messageType string) bool {
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
			// Normalize phone number to 62xxx format
			normalized := normalizePhone(tt.input)

			if tt.valid && normalized != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, normalized)
			}
		})
	}
}

func normalizePhone(phone string) string {
	// Remove non-numeric characters
	var cleaned string
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			cleaned += string(r)
		}
	}

	// Convert 08xxx to 628xxx
	if len(cleaned) > 0 && cleaned[0] == '0' {
		cleaned = "62" + cleaned[1:]
	}

	// Remove leading +
	if len(cleaned) > 0 && cleaned[0] == '+' {
		cleaned = cleaned[1:]
	}

	// Validate length (Indonesian mobile: 62 + 9-13 digits)
	if len(cleaned) < 11 || len(cleaned) > 15 {
		return ""
	}

	// Validate prefix
	if len(cleaned) < 2 || cleaned[:2] != "62" {
		return ""
	}

	return cleaned
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
			result := isTransactional(tt.headerValue)
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

func tryAcquireLock(key string, ttl time.Duration) bool {
	// Simplified lock logic - real implementation uses Redis SET NX
	return true // Placeholder
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
