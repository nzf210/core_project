package webhook

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"core_project/shared/sdk/config"
)

func initTestConfig() {
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{}
	}
}

func TestValidateN8NSignature_EmptySecret(t *testing.T) {
	initTestConfig()
	config.GlobalConfig.N8N.WebhookSecret = ""
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Webhook-Secret", "anything")
	if ValidateN8NSignature(req) {
		t.Error("expected false when secret is empty")
	}
}

func TestValidateN8NSignature_NoHeader(t *testing.T) {
	initTestConfig()
	config.GlobalConfig.N8N.WebhookSecret = "mysecret"
	defer func() { config.GlobalConfig.N8N.WebhookSecret = "" }()
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	if ValidateN8NSignature(req) {
		t.Error("expected false when header is missing")
	}
}

func TestValidateN8NSignature_WrongSecret(t *testing.T) {
	initTestConfig()
	config.GlobalConfig.N8N.WebhookSecret = "mysecret"
	defer func() { config.GlobalConfig.N8N.WebhookSecret = "" }()
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Webhook-Secret", "wrongsecret")
	if ValidateN8NSignature(req) {
		t.Error("expected false for wrong secret")
	}
}

func TestValidateN8NSignature_CorrectSecret(t *testing.T) {
	initTestConfig()
	config.GlobalConfig.N8N.WebhookSecret = "mysecret"
	defer func() { config.GlobalConfig.N8N.WebhookSecret = "" }()
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Webhook-Secret", "mysecret")
	if !ValidateN8NSignature(req) {
		t.Error("expected true for correct secret")
	}
}

func TestRequireN8NSecret_Unauthorized(t *testing.T) {
	initTestConfig()
	config.GlobalConfig.N8N.WebhookSecret = "mysecret"
	defer func() { config.GlobalConfig.N8N.WebhookSecret = "" }()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })
	handler := RequireN8NSecret(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if called {
		t.Error("next handler should not be called when unauthorized")
	}
}

func TestRequireN8NSecret_Authorized(t *testing.T) {
	initTestConfig()
	config.GlobalConfig.N8N.WebhookSecret = "mysecret"
	defer func() { config.GlobalConfig.N8N.WebhookSecret = "" }()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := RequireN8NSecret(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-Webhook-Secret", "mysecret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !called {
		t.Error("next handler should be called when authorized")
	}
}
