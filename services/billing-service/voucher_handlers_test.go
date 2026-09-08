package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"core_project/shared/sdk/config"
)

func TestExtractTokenFromVoucherInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Plain alphanumeric code",
			input:    "LITE-83921-A1B2",
			expected: "",
		},
		{
			name:     "Direct JWT token",
			input:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIifQ.sig123",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJmb28iOiJiYXIifQ.sig123",
		},
		{
			name:     "Full URL with token query param",
			input:    "https://app.wch.id/redeem?token=eySampleToken123",
			expected: "eySampleToken123",
		},
		{
			name:     "Full URL with multiple query params",
			input:    "https://app.wch.id/redeem?source=wa&token=eySampleToken123&plan=lite",
			expected: "eySampleToken123",
		},
		{
			name:     "Fragment with token",
			input:    "app.wch.id/redeem#token=eySampleToken123",
			expected: "eySampleToken123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTokenFromVoucherInput(tt.input)
			if got != tt.expected {
				t.Errorf("extractTokenFromVoucherInput(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSignAndParseVoucherToken(t *testing.T) {
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{}
	}
	config.GlobalConfig.JWTSecret = "test-secret-key-32-characters-long!"

	programID := "prog-123"
	planID := "lite"
	durationMonths := 3
	expiresAt := time.Now().Add(24 * time.Hour)

	token, err := signVoucherToken(programID, planID, durationMonths, expiresAt)
	if err != nil {
		t.Fatalf("signVoucherToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := parseVoucherLinkToken(token, config.GlobalConfig.JWTSecret)
	if err != nil {
		t.Fatalf("parseVoucherLinkToken failed: %v", err)
	}

	if claims.ProgramID != programID {
		t.Errorf("claims.ProgramID = %q, want %q", claims.ProgramID, programID)
	}
	if claims.PlanID != planID {
		t.Errorf("claims.PlanID = %q, want %q", claims.PlanID, planID)
	}
	if claims.DurationMonths != durationMonths {
		t.Errorf("claims.DurationMonths = %d, want %d", claims.DurationMonths, durationMonths)
	}
}

func TestParseVoucherToken_Expired(t *testing.T) {
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{}
	}
	config.GlobalConfig.JWTSecret = "test-secret-key-32-characters-long!"

	programID := "prog-expired"
	planID := "pro"
	durationMonths := 1
	expiresAt := time.Now().Add(-1 * time.Hour) // in the past

	token, err := signVoucherToken(programID, planID, durationMonths, expiresAt)
	if err != nil {
		t.Fatalf("signVoucherToken failed: %v", err)
	}

	_, err = parseVoucherLinkToken(token, config.GlobalConfig.JWTSecret)
	if err == nil {
		t.Fatal("expected error parsing expired voucher token, got nil")
	}
}

func TestHandleRedeemVoucherLink_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/voucher/redeem-link", nil)
	rr := httptest.NewRecorder()

	handleRedeemVoucherLink(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestHandleRedeemVoucherLink_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/voucher/redeem-link", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handleRedeemVoucherLink(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestHandleRedeemVoucherLink_MissingToken(t *testing.T) {
	body, _ := json.Marshal(VoucherLinkRedeemReq{
		Token:    "",
		TenantID: "tenant-123",
	})
	req := httptest.NewRequest(http.MethodPost, "/voucher/redeem-link", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	handleRedeemVoucherLink(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
