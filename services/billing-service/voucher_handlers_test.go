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

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       *string
	}{
		{
			name:       "IPv4 with port (typical RemoteAddr)",
			remoteAddr: "127.0.0.1:43394",
			want:       ptrString("127.0.0.1"),
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:54321",
			want:       ptrString("::1"),
		},
		{
			name:       "Plain IPv4 without port",
			remoteAddr: "192.168.1.100",
			want:       ptrString("192.168.1.100"),
		},
		{
			name:       "Plain IPv6 without port",
			remoteAddr: "2001:db8::1",
			want:       ptrString("2001:db8::1"),
		},
		{
			name:       "X-Forwarded-For multiple IPs takes first",
			remoteAddr: "127.0.0.1:43394",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178",
			},
			want: ptrString("203.0.113.195"),
		},
		{
			name:       "X-Forwarded-For single IP with spaces",
			remoteAddr: "127.0.0.1:43394",
			headers: map[string]string{
				"X-Forwarded-For": "  198.51.100.50  ",
			},
			want: ptrString("198.51.100.50"),
		},
		{
			name:       "X-Real-IP takes precedence over RemoteAddr",
			remoteAddr: "127.0.0.1:43394",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.25",
			},
			want: ptrString("198.51.100.25"),
		},
		{
			name:       "Invalid IP returns nil (safe for postgres inet NULL)",
			remoteAddr: "invalid-ip-string",
			want:       nil,
		},
		{
			name:       "Empty RemoteAddr returns nil",
			remoteAddr: "",
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := extractClientIP(req)
			if tt.want == nil {
				if got != nil {
					t.Errorf("extractClientIP() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("extractClientIP() = nil, want %v", *tt.want)
				} else if *got != *tt.want {
					t.Errorf("extractClientIP() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}

	t.Run("Nil request returns nil", func(t *testing.T) {
		got := extractClientIP(nil)
		if got != nil {
			t.Errorf("extractClientIP(nil) = %v, want nil", *got)
		}
	})
}

func ptrString(s string) *string {
	return &s
}

func TestCalculateVoucherCharge(t *testing.T) {
	price := int64(10000000) // 100,000 IDR in sen

	tests := []struct {
		name          string
		voucherType   string
		discountValue int
		want          int64
	}{
		{
			name:          "free_months",
			voucherType:   "free_months",
			discountValue: 0,
			want:          0,
		},
		{
			name:          "discount_percent 20%",
			voucherType:   "discount_percent",
			discountValue: 20,
			want:          8000000,
		},
		{
			name:          "discount_percent 100%",
			voucherType:   "discount_percent",
			discountValue: 100,
			want:          0,
		},
		{
			name:          "discount_fixed",
			voucherType:   "discount_fixed",
			discountValue: 2000000,
			want:          8000000,
		},
		{
			name:          "discount_fixed exceeds price",
			voucherType:   "discount_fixed",
			discountValue: 15000000,
			want:          0,
		},
		{
			name:          "unknown type full price",
			voucherType:   "other",
			discountValue: 50,
			want:          price,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateVoucherCharge(price, tt.voucherType, tt.discountValue)
			if got != tt.want {
				t.Errorf("calculateVoucherCharge() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveVoucherValidityDays(t *testing.T) {
	tests := []struct {
		name                   string
		codeValidityDays       int
		programDurationMonths  int
		want                   int
	}{
		{
			name:                  "Code validity days takes precedence",
			codeValidityDays:      60,
			programDurationMonths: 1,
			want:                  60,
		},
		{
			name:                  "Fallback to program duration months",
			codeValidityDays:      0,
			programDurationMonths: 3,
			want:                  90,
		},
		{
			name:                  "Both zero defaults to 30",
			codeValidityDays:      0,
			programDurationMonths: 0,
			want:                  30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVoucherValidityDays(tt.codeValidityDays, tt.programDurationMonths)
			if got != tt.want {
				t.Errorf("resolveVoucherValidityDays() = %d, want %d", got, tt.want)
			}
		})
	}
}
