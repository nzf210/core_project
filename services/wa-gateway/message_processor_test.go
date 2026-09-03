package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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
