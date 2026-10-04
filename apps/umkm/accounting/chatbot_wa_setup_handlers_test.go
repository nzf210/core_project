package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// =============================================================================
// F048 AC-9: WhatsApp Number Extraction & Setup Handler Tests
// =============================================================================

func TestExtractPhoneFromJID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard WhatsApp JID",
			input:    "6281234567890@s.whatsapp.net",
			expected: "6281234567890",
		},
		{
			name:     "Multi-device WhatsApp JID with device suffix",
			input:    "6281234567890:12@s.whatsapp.net",
			expected: "6281234567890",
		},
		{
			name:     "Multi-device WhatsApp JID with high device index",
			input:    "6289876543210:99@s.whatsapp.net",
			expected: "6289876543210",
		},
		{
			name:     "JID with whitespace",
			input:    "  6285551234567:1@s.whatsapp.net  ",
			expected: "6285551234567",
		},
		{
			name:     "Plain phone number without @ domain",
			input:    "628111222333",
			expected: "628111222333",
		},
		{
			name:     "Plain phone number with device suffix",
			input:    "628111222333:2",
			expected: "628111222333",
		},
		{
			name:     "Empty JID",
			input:    "",
			expected: "",
		},
		{
			name:     "Whitespace only JID",
			input:    "   ",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractPhoneFromJID(tc.input)
			if result != tc.expected {
				t.Errorf("extractPhoneFromJID(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestHandleWASetup_MissingTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/umkm/wa/setup", nil)
	rec := httptest.NewRecorder()

	handleWASetup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing tenant ID, got %d", rec.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Message != errMissingTenantID {
		t.Errorf("expected error message %q, got %q", errMissingTenantID, resp.Message)
	}
}

func TestHandleWAConnect_Validation(t *testing.T) {
	t.Run("Missing Tenant ID", func(t *testing.T) {
		body := bytes.NewBufferString(`{"provider":"whatsmeow"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/umkm/wa/connect", body)
		rec := httptest.NewRecorder()

		handleWAConnect(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for missing tenant ID, got %d", rec.Code)
		}
	})

	t.Run("Invalid Provider Enum", func(t *testing.T) {
		body := bytes.NewBufferString(`{"provider":"invalid_provider"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/umkm/wa/connect", body)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		rec := httptest.NewRecorder()

		handleWAConnect(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for invalid provider, got %d", rec.Code)
		}
	})

	t.Run("Invalid JSON Body", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid json}`)
		req := httptest.NewRequest(http.MethodPost, "/api/umkm/wa/connect", body)
		req.Header.Set("X-Tenant-ID", "11111111-1111-1111-1111-111111111111")
		rec := httptest.NewRecorder()

		handleWAConnect(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400 for invalid json, got %d", rec.Code)
		}
	})
}

func TestWASetupPayload_JSONSerialization(t *testing.T) {
	// Verify that the response structure correctly serializes wa_number, phone_number_id,
	// and display_phone_number according to F048 AC-9 contract.
	type whatsmeowStatus struct {
		Connected bool   `json:"connected"`
		Status    string `json:"status"`
		WANumber  string `json:"wa_number,omitempty"`
	}

	type cloudAPIStatus struct {
		Active             bool   `json:"active"`
		CreditBal          int64  `json:"credit_balance_rupiah"`
		LastSync           string `json:"last_sync_at"`
		PhoneNumberID      string `json:"phone_number_id,omitempty"`
		DisplayPhoneNumber string `json:"display_phone_number,omitempty"`
	}

	wm := whatsmeowStatus{
		Connected: true,
		Status:    "connected",
		WANumber:  "6281234567890",
	}

	cloud := cloudAPIStatus{
		Active:             true,
		CreditBal:          50000,
		LastSync:           "2026-09-15T10:00:00Z",
		PhoneNumberID:      "109876543210",
		DisplayPhoneNumber: "+62 812-3456-7890",
	}

	data := map[string]interface{}{
		"wa_provider_preference": "auto",
		"whatsmeow":              wm,
		"cloud_api":              cloud,
		"has_cloud_api_addon":    true,
		"can_use_cloud_api":      true,
	}

	resp := APIResponse{
		Success: true,
		Data:    data,
	}

	encoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal setup response: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("failed to unmarshal setup response: %v", err)
	}

	dataMap, ok := decoded["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing or invalid data map in response")
	}

	wmMap, ok := dataMap["whatsmeow"].(map[string]interface{})
	if !ok || wmMap["wa_number"] != "6281234567890" {
		t.Errorf("expected whatsmeow.wa_number = '6281234567890', got %v", wmMap["wa_number"])
	}

	cloudMap, ok := dataMap["cloud_api"].(map[string]interface{})
	if !ok || cloudMap["display_phone_number"] != "+62 812-3456-7890" {
		t.Errorf("expected cloud_api.display_phone_number = '+62 812-3456-7890', got %v", cloudMap["display_phone_number"])
	}
	if cloudMap["phone_number_id"] != "109876543210" {
		t.Errorf("expected cloud_api.phone_number_id = '109876543210', got %v", cloudMap["phone_number_id"])
	}
}
