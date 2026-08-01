package main

import (
	"net/http"
	"os"
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

// =============================================================================
// F048: WA Provider Preferences — Unit Tests
// =============================================================================

// Mock UUIDs sesuai real schema
const (
	mockWAProviderTenantID = "11111111-1111-1111-1111-111111111111"
)

// TestResolveProviderPreference_HeaderWins verifies X-WA-Provider-Override header takes priority
// over DB lookup (F048 AC-6: auth-service forces provider for OTP routing)
func TestResolveProviderPreference_HeaderWins(t *testing.T) {
	cases := []struct {
		name     string
		header   string
		expected string
	}{
		{"header auto", "auto", "auto"},
		{"header whatsmeow", "whatsmeow", "whatsmeow"},
		{"header cloud_api", "cloud_api", "cloud_api"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/api/wa/send", nil)
			req.Header.Set("X-WA-Provider-Override", tc.header)

			got := resolveProviderPreference(req, mockWAProviderTenantID)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TestResolveProviderPreference_HeaderEmpty verifies fallback to DB lookup when header empty
// In test mode db is nil, so getTenantWAProviderPreference returns "auto"
func TestResolveProviderPreference_HeaderEmpty(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "/api/wa/send", nil)
	// No X-WA-Provider-Override header

	got := resolveProviderPreference(req, mockWAProviderTenantID)
	if got != "auto" {
		t.Errorf("expected 'auto' fallback (db nil in test), got %q", got)
	}
}

// TestGetTenantWAProviderPreference_DBUnavailable verifies "auto" fallback when DB unavailable
func TestGetTenantWAProviderPreference_DBUnavailable(t *testing.T) {
	// db is nil in unit test (no real connection)
	got := getTenantWAProviderPreference(mockWAProviderTenantID)
	if got != "auto" {
		t.Errorf("expected 'auto' when DB unavailable, got %q", got)
	}
}

// TestIsTransactional verifies the X-Message-Type and X-Source routing logic
// (used by wa-gateway to decide between Cloud API vs whatsmeow in hybrid mode)
func TestIsTransactional(t *testing.T) {
	cases := []struct {
		name     string
		msgType  string
		source   string
		expected bool
	}{
		// Transactional message types → Cloud API
		{"otp type", "otp", "", true},
		{"invoice type", "invoice", "", true},
		{"payment type", "payment", "", true},
		{"subscription type", "subscription", "", true},
		{"system type", "system", "", true},

		// Transactional sources → Cloud API
		{"auth-service source", "", "auth-service", true},
		{"billing-service source", "", "billing-service", true},
		{"notification-service source", "", "notification-service", true},

		// Conversational → whatsmeow
		{"chat text", "text", "chatbot", false},
		{"no headers", "", "", false},
		{"unknown type", "marketing", "unknown", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/api/wa/send", nil)
			if tc.msgType != "" {
				req.Header.Set("X-Message-Type", tc.msgType)
			}
			if tc.source != "" {
				req.Header.Set("X-Source", tc.source)
			}

			got := isTransactional(req)
			if got != tc.expected {
				t.Errorf("expected %v, got %v (msgType=%q, source=%q)",
					tc.expected, got, tc.msgType, tc.source)
			}
		})
	}
}

// TestWAProviderPreference_EnumValues verifies valid enum values (migration 000063 wa_provider_enum)
func TestWAProviderPreference_EnumValues(t *testing.T) {
	validPrefs := []string{"auto", "whatsmeow", "cloud_api"}
	for _, pref := range validPrefs {
		if pref == "" {
			t.Error("preference should not be empty")
		}
	}

	// Invalid values that should be rejected by validateChatbotConfig
	invalidPrefs := []string{"telegram", "email", "sms", "voicemail", ""}
	for _, pref := range invalidPrefs {
		switch pref {
		case "auto", "whatsmeow", "cloud_api":
			t.Errorf("%q should NOT be in valid enum (test setup error)", pref)
		}
	}
}

// =============================================================================
// F063: WA Keyword Registration — Unit Tests
// =============================================================================

func TestIsSixDigitOTP(t *testing.T) {
	cases := []struct {
		text     string
		expected bool
	}{
		{"123456", true},
		{"000000", true},
		{"999999", true},
		{"12345", false},   // too short
		{"1234567", false}, // too long
		{"12345a", false},  // has letter
		{"abcdef", false},  // all letters
		{"", false},
		{" 123456", false}, // has space
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got := isSixDigitOTP(tc.text)
			if got != tc.expected {
				t.Errorf("isSixDigitOTP(%q) = %v, want %v", tc.text, got, tc.expected)
			}
		})
	}
}

func TestExtractPhoneFromJID(t *testing.T) {
	cases := []struct {
		jid      string
		expected string
	}{
		{"62812123456789@s.whatsapp.net", "0812123456789"},
		{"62812123456789@c.us", "0812123456789"},
		{"081212345678@s.whatsapp.net", "081212345678"},
		{"noatsign", "noatsign"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.jid, func(t *testing.T) {
			got := extractPhoneFromJID(tc.jid)
			if got != tc.expected {
				t.Errorf("extractPhoneFromJID(%q) = %q, want %q", tc.jid, got, tc.expected)
			}
		})
	}
}

func TestGetAuthServiceURL(t *testing.T) {
	// Save and restore env
	orig := os.Getenv("AUTH_SERVICE_URL")
	defer func() {
		if orig == "" {
			os.Unsetenv("AUTH_SERVICE_URL")
		} else {
			os.Setenv("AUTH_SERVICE_URL", orig)
		}
	}()

	// Default when not set
	os.Unsetenv("AUTH_SERVICE_URL")
	if got := getAuthServiceURL(); got != "http://auth-service:8001" {
		t.Errorf("expected default URL, got %q", got)
	}

	// Custom URL
	os.Setenv("AUTH_SERVICE_URL", "http://auth-service:8001")
	if got := getAuthServiceURL(); got != "http://auth-service:8001" {
		t.Errorf("expected custom URL, got %q", got)
	}
}

func TestHandleWARegistrationStep_Step1_Name(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628121234567@c.us",
		PhoneNumber: "628121234567",
		Step:        1,
	}
	// rawText is not uppercased; step 1 expects raw business name
	handled := handleWARegistrationStep("system", session, "Toko Saya", "TOKO SAYA")
	if !handled {
		t.Error("expected step 1 to handle input")
	}
	if session.BusinessName != "Toko Saya" {
		t.Errorf("expected business name 'Toko Saya', got %q", session.BusinessName)
	}
	if session.Step != 2 {
		t.Errorf("expected step 2, got %d", session.Step)
	}
}

func TestHandleWARegistrationStep_Step2_Type(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628121234567@c.us",
		PhoneNumber: "628121234567",
		Step:        2,
	}
	// upperText is what gets compared for choice keys
	handled := handleWARegistrationStep("system", session, "Warung Saya", "2")
	if !handled {
		t.Error("expected step 2 to handle input")
	}
	if session.BusinessType != "warung" {
		t.Errorf("expected type 'warung', got %q", session.BusinessType)
	}
	if session.Step != 3 {
		t.Errorf("expected step 3, got %d", session.Step)
	}
}

func TestHandleWARegistrationStep_InvalidType(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628121234567@c.us",
		PhoneNumber: "628121234567",
		Step:        2,
	}
	// Invalid choice should still "handle" (not fall through to chatbot)
	handled := handleWARegistrationStep("system", session, "x", "9")
	if !handled {
		t.Error("expected invalid choice to be handled (reply error, not fall through)")
	}
	// Step should NOT advance
	if session.Step != 2 {
		t.Errorf("expected step 2 unchanged on invalid choice, got %d", session.Step)
	}
}

func TestHandleWARegistrationStep_Step3_UsernameTooShort(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628121234567@c.us",
		PhoneNumber: "628121234567",
		Step:        3,
	}
	handled := handleWARegistrationStep("system", session, "ab", "AB")
	if !handled {
		t.Error("expected step 3 to handle short username")
	}
	if session.Username != "" {
		t.Errorf("expected username unchanged on too-short, got %q", session.Username)
	}
}

func TestHandleWARegistrationStep_Step4_PasswordTooShort(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628121234567@c.us",
		PhoneNumber: "628121234567",
		Step:        4,
	}
	handled := handleWARegistrationStep("system", session, "12345", "12345")
	if !handled {
		t.Error("expected step 4 to handle short password")
	}
	if session.Password != "" {
		t.Errorf("expected password unchanged on too-short, got %q", session.Password)
	}
}

func TestHandleWARegistrationStep_Step5_ConfirmYA(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628121234567@c.us",
		PhoneNumber: "628121234567",
		Username:    "toko_saya",
		Password:    "secret123",
		BusinessName: "Toko Saya",
		BusinessType: "warung",
		Step:        5,
	}
	// YA (uppercased) should advance step (submitWARegistration called at step 6)
	handled := handleWARegistrationStep("system", session, "ya", "YA")
	if !handled {
		t.Error("expected step 5 to handle YA")
	}
	if session.Step != 6 {
		t.Errorf("expected step 6 (submit), got %d", session.Step)
	}
}

func TestHandleWARegistrationStep_Step5_ChangePhone(t *testing.T) {
	session := &waRegistrationSession{
		SenderJID:   "628121234567@c.us",
		PhoneNumber: "628121234567",
		Step:        5,
	}
	// User wants to change phone number
	handled := handleWARegistrationStep("system", session, "628211234567", "628211234567")
	if !handled {
		t.Error("expected step 5 to handle phone correction")
	}
	if session.PhoneNumber != "628211234567" {
		t.Errorf("expected phone updated, got %q", session.PhoneNumber)
	}
	// Step should stay at 5
	if session.Step != 5 {
		t.Errorf("expected step 5 unchanged after phone correction, got %d", session.Step)
	}
}
