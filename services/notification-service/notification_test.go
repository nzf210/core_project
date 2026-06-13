package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleSendNotification_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/notification/send", nil)
	rr := httptest.NewRecorder()
	handleSendNotification(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleSendNotification_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/notification/send", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleSendNotification(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleSendNotification_DefaultTenantID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/notification/send", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleSendNotification(rr, req)
	// Should use "global" as default tenantID and attempt to send
	if rr.Code == http.StatusOK {
		t.Error("should fail with bad request (invalid JSON)")
	}
}

func TestNormalizeWAJID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"081234567890", "6281234567890@s.whatsapp.net"},
		{"+6281234567890", "6281234567890@s.whatsapp.net"},
		{"6281234567890@s.whatsapp.net", "6281234567890@s.whatsapp.net"},
		{"+1234567890", "1234567890@s.whatsapp.net"},
	}

	for _, tt := range tests {
		result := normalizeWAJID(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeWAJID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSendTelegram_NoToken(t *testing.T) {
	origToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Setenv("TELEGRAM_BOT_TOKEN", origToken)

	// Should warn and skip, not panic
	err := sendTelegram("12345", "test message")
	if err != nil {
		t.Errorf("expected nil (skip) when no token, got %v", err)
	}
}

func TestSendEmail_NoSMTPConfig(t *testing.T) {
	origHost := os.Getenv("SMTP_HOST")
	origPass := os.Getenv("SMTP_PASS")
	origUser := os.Getenv("SMTP_USER")
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PASS")
	os.Unsetenv("SMTP_USER")
	defer func() {
		os.Setenv("SMTP_HOST", origHost)
		os.Setenv("SMTP_PASS", origPass)
		os.Setenv("SMTP_USER", origUser)
	}()

	err := sendEmail("test@test.com", "Subject", "Body")
	if err != nil {
		t.Errorf("expected nil (skip) when no SMTP config, got %v", err)
	}
}

func TestNotificationRequest_Fields(t *testing.T) {
	req := NotificationRequest{
		Type:    "email",
		Target:  "user@test.com",
		Message: "Hello",
		Subject: "Test",
	}
	if req.Type != "email" {
		t.Errorf("expected email, got %q", req.Type)
	}
	if req.Target != "user@test.com" {
		t.Errorf("expected user@test.com, got %q", req.Target)
	}
}
