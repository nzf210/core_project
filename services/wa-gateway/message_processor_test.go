package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.mau.fi/whatsmeow/types/events"
)

// TestGetN8NWebhookURL_Default verifies default URL
func TestGetN8NWebhookURL_Default(t *testing.T) {
	orig := os.Getenv("N8N_WEBHOOK_URL")
	os.Unsetenv("N8N_WEBHOOK_URL")
	defer func() {
		if orig != "" {
			os.Setenv("N8N_WEBHOOK_URL", orig)
		}
	}()

	got := getN8NWebhookURL()
	expected := "http://n8n-main:5678"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestGetN8NWebhookURL_CustomWithTrailingSlash verifies trailing slash removal
func TestGetN8NWebhookURL_CustomWithTrailingSlash(t *testing.T) {
	orig := os.Getenv("N8N_WEBHOOK_URL")
	os.Setenv("N8N_WEBHOOK_URL", "https://n8n.example.com/")
	defer func() {
		if orig == "" {
			os.Unsetenv("N8N_WEBHOOK_URL")
		} else {
			os.Setenv("N8N_WEBHOOK_URL", orig)
		}
	}()

	got := getN8NWebhookURL()
	expected := "https://n8n.example.com"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestGetAuthServiceURL_Default verifies default auth service URL
func TestGetAuthServiceURL_Default(t *testing.T) {
	orig := os.Getenv("AUTH_SERVICE_URL")
	os.Unsetenv("AUTH_SERVICE_URL")
	defer func() {
		if orig != "" {
			os.Setenv("AUTH_SERVICE_URL", orig)
		}
	}()

	got := getAuthServiceURL()
	expected := "http://auth-service:8001"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// TestGetAuthServiceURL_Custom verifies custom auth service URL from env
func TestGetAuthServiceURL_Custom(t *testing.T) {
	orig := os.Getenv("AUTH_SERVICE_URL")
	custom := "https://auth.example.com"
	os.Setenv("AUTH_SERVICE_URL", custom)
	defer func() {
		if orig == "" {
			os.Unsetenv("AUTH_SERVICE_URL")
		} else {
			os.Setenv("AUTH_SERVICE_URL", orig)
		}
	}()

	got := getAuthServiceURL()
	if got != custom {
		t.Errorf("expected %q, got %q", custom, got)
	}
}

// TestForwardToN8NChatbot_HTTPError verifies HTTP error handling
func TestForwardToN8NChatbot_HTTPError(t *testing.T) {
	// Start mock N8N server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Override N8N URL
	orig := os.Getenv("N8N_WEBHOOK_URL")
	os.Setenv("N8N_WEBHOOK_URL", server.URL)
	defer func() {
		if orig == "" {
			os.Unsetenv("N8N_WEBHOOK_URL")
		} else {
			os.Setenv("N8N_WEBHOOK_URL", orig)
		}
	}()

	// Should handle error without panic
	forwardToN8NChatbot("test-tenant", "test-jid", "628123456", "test message")
}

// TestForwardToN8NChatbot_ValidResponse verifies successful forwarding
func TestForwardToN8NChatbot_ValidResponse(t *testing.T) {
	// Start mock N8N server that returns valid response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/webhook/chatbot/incoming" {
			t.Errorf("expected path /webhook/chatbot/incoming, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response": "AI reply here"}`))
	}))
	defer server.Close()

	orig := os.Getenv("N8N_WEBHOOK_URL")
	os.Setenv("N8N_WEBHOOK_URL", server.URL)
	defer func() {
		if orig == "" {
			os.Unsetenv("N8N_WEBHOOK_URL")
		} else {
			os.Setenv("N8N_WEBHOOK_URL", orig)
		}
	}()

	forwardToN8NChatbot("test-tenant", "test-jid", "628123456", "test message")
}

// TestSendHelpMenu verifies help menu doesn't panic without client
func TestSendHelpMenu(t *testing.T) {
	// Mock WA client to avoid actual send
	tenantID := "help-test"
	senderJID := "6281234567890@s.whatsapp.net"

	// Should not panic even without client
	sendHelpMenu(tenantID, senderJID)
}

func TestToLocalPhoneAndToIntlPhone(t *testing.T) {
	if got := toLocalPhone("6281355492003"); got != "081355492003" {
		t.Errorf("expected 081355492003, got %s", got)
	}
	if got := toLocalPhone("081355492003"); got != "081355492003" {
		t.Errorf("expected 081355492003, got %s", got)
	}
	if got := toIntlPhone("081355492003"); got != "6281355492003" {
		t.Errorf("expected 6281355492003, got %s", got)
	}
	if got := toIntlPhone("6281355492003"); got != "6281355492003" {
		t.Errorf("expected 6281355492003, got %s", got)
	}
}

func TestIsDuplicateMessage(t *testing.T) {
	ctx := context.Background()
	msgID := "unique-test-msg-123"

	// First time: not duplicate
	if isDuplicateMessage(ctx, msgID) {
		t.Errorf("expected first check of %q to be false (not duplicate)", msgID)
	}

	// Second time: must be duplicate
	if !isDuplicateMessage(ctx, msgID) {
		t.Errorf("expected second check of %q to be true (duplicate)", msgID)
	}

	// Empty message ID: never duplicate
	if isDuplicateMessage(ctx, "") {
		t.Error("expected empty msgID to return false")
	}
}

func TestExtractMessageText_NilAndEmpty(t *testing.T) {
	if got := extractMessageText(nil); got != "" {
		t.Errorf("expected empty string for nil event, got %q", got)
	}
	if got := extractMessageText(&events.Message{}); got != "" {
		t.Errorf("expected empty string for empty message, got %q", got)
	}
}

func TestIsSystemTenant(t *testing.T) {
	tests := []struct {
		tenantID string
		expected bool
	}{
		{"system", true},
		{"SYSTEM", true},
		{"platform", true},
		{"Platform", true},
		{"wch", true},
		{"", true},
		{"   ", true},
		{"11111111-1111-1111-1111-111111111111", false},
		{"umkm-toko-berkah", false},
		{"tenant-123", false},
	}

	for _, tt := range tests {
		got := isSystemTenant(tt.tenantID)
		if got != tt.expected {
			t.Errorf("isSystemTenant(%q) = %v, want %v", tt.tenantID, got, tt.expected)
		}
	}
}

func TestGetChatbotServiceURL_EnvironmentOverrides(t *testing.T) {
	origChatbot := os.Getenv("UMKM_CHATBOT_URL")
	origAppEnv := os.Getenv("APP_ENV")
	origDBHost := os.Getenv("DB_HOST")
	defer func() {
		os.Setenv("UMKM_CHATBOT_URL", origChatbot)
		os.Setenv("APP_ENV", origAppEnv)
		os.Setenv("DB_HOST", origDBHost)
	}()

	os.Setenv("UMKM_CHATBOT_URL", "http://my-chatbot:8203/")
	if url := getChatbotServiceURL(); url != "http://my-chatbot:8203" {
		t.Errorf("expected trimmed custom URL, got %s", url)
	}

	os.Unsetenv("UMKM_CHATBOT_URL")
	os.Setenv("APP_ENV", "production")
	if url := getChatbotServiceURL(); url != "http://umkm-chatbot:8203" {
		t.Errorf("expected production default umkm-chatbot:8203, got %s", url)
	}

	os.Setenv("APP_ENV", "development")
	os.Setenv("DB_HOST", "127.0.0.1")
	if url := getChatbotServiceURL(); url != "http://localhost:8203" {
		t.Errorf("expected dev default localhost:8203, got %s", url)
	}
}

func TestFallbackToInternalChatbot(t *testing.T) {
	var receivedBody map[string]string
	var receivedTenant string

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/webhook/wa" {
			http.NotFound(w, r)
			return
		}
		receivedTenant = r.URL.Query().Get("tenant_id")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":true,"message":"queued"}`))
	}))
	defer mockServer.Close()

	origChatbot := os.Getenv("UMKM_CHATBOT_URL")
	defer os.Setenv("UMKM_CHATBOT_URL", origChatbot)
	os.Setenv("UMKM_CHATBOT_URL", mockServer.URL)

	success := fallbackToInternalChatbot("tenant-123", "628111222@s.whatsapp.net", "628111222", "Halo CS")
	if !success {
		t.Fatalf("expected fallbackToInternalChatbot to succeed")
	}

	if receivedTenant != "tenant-123" {
		t.Errorf("expected tenant_id 'tenant-123', got %q", receivedTenant)
	}
	if receivedBody["sender"] != "628111222@s.whatsapp.net" {
		t.Errorf("expected sender '628111222@s.whatsapp.net', got %q", receivedBody["sender"])
	}
	if receivedBody["message"] != "Halo CS" {
		t.Errorf("expected message 'Halo CS', got %q", receivedBody["message"])
	}
}
