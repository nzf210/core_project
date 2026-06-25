package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"core_project/shared/sdk/config"
)

func TestNormalizeTo(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+6281234567890", "6281234567890"},
		{"6281234567890", "6281234567890"},
		{"+62 812-3456-7890", "6281234567890"},
		{"081234567890", "6281234567890"},
	}

	for _, tt := range tests {
		result := normalizeTo(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeTo(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSendRequest_TypeText(t *testing.T) {
	req := SendRequest{
		To:   "6281234567890",
		Type: "text",
		Text: "Hello World",
	}
	if req.To == "" || req.Type != "text" || req.Text != "Hello World" {
		t.Error("SendRequest fields mismatch")
	}
}

func TestSendRequest_TypeTemplate(t *testing.T) {
	req := SendRequest{
		To:       "6281234567890",
		Type:     "template",
		Template: "welcome_message",
		Params:   []string{"John", "WCH Platform"},
	}
	if len(req.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(req.Params))
	}
}

func TestCorsMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := corsMiddleware(next)

	t.Run("sets CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected Access-Control-Allow-Origin: *")
		}
		if rr.Header().Get("Access-Control-Allow-Methods") != "GET, POST, PUT, DELETE, OPTIONS" {
			t.Errorf("wrong Allow-Methods: %s", rr.Header().Get("Access-Control-Allow-Methods"))
		}
	})

	t.Run("handles OPTIONS preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for OPTIONS, got %d", rr.Code)
		}
	})
}

func TestCloudAPICredential_Fields(t *testing.T) {
	cred := CloudAPICredential{
		ID:            "cred-1",
		TenantID:      "tenant-1",
		PhoneNumberID: "123456789",
		WABAID:        "waba-1",
		AccessToken:   "secret-token",
		VerifyToken:   "verify-token-123",
		IsActive:      true,
	}
	if cred.ID != "cred-1" {
		t.Error("credential ID mismatch")
	}
	if cred.AccessToken == "" {
		t.Error("access token should not be empty")
	}
}

func TestMetaSendPayload_Text(t *testing.T) {
	payload := MetaSendPayload{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               "6281234567890",
		Type:             "text",
		Text:             &MetaText{Body: "Hello"},
	}
	if payload.To == "" {
		t.Error("To field is required")
	}
	if payload.Text.Body != "Hello" {
		t.Errorf("expected 'Hello', got %q", payload.Text.Body)
	}
}

func TestMetaSendPayload_Template(t *testing.T) {
	payload := MetaSendPayload{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               "6281234567890",
		Type:             "template",
		Template: &MetaTemplate{
			Name: "hello_world",
			Language: MetaTemplateLanguage{Code: "id"},
		},
	}
	if payload.Template.Name != "hello_world" {
		t.Errorf("expected 'hello_world', got %q", payload.Template.Name)
	}
	if payload.Template.Language.Code != "id" {
		t.Errorf("expected 'id', got %q", payload.Template.Language.Code)
	}
}

func TestMetaError_Fields(t *testing.T) {
	err := MetaError{
		Code:    400,
		Message: "Bad Request",
		Type:    "OAuthException",
	}
	if err.Code != 400 {
		t.Errorf("expected 400, got %d", err.Code)
	}
}

func TestSendResponse_Fields(t *testing.T) {
	resp := SendResponse{
		Success: true,
		Message: "Message sent",
		WAMsgID: "wamid.12345",
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.WAMsgID == "" {
		t.Error("expected WA message ID")
	}
}

func TestHandleSend_MissingTenant(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"to":"628123","text":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleSend(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing X-Tenant-ID, got %d", rr.Code)
	}
}

func TestHandleSend_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/send", nil)
	rr := httptest.NewRecorder()
	handleSend(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestBuildDBURI(t *testing.T) {
	cfg := &config.Config{}
	cfg.DB.User = "test"
	cfg.DB.Password = "pass"
	cfg.DB.Host = "localhost"
	cfg.DB.Port = 5432
	cfg.DB.Name = "testdb"
	cfg.DB.SSLMode = "disable"

	uri := buildDBURI(cfg)
	if uri == "" {
		t.Error("DB URI should not be empty")
	}
	if !strings.Contains(uri, "testdb") {
		t.Errorf("URI should contain db name: %s", uri)
	}
	if !strings.Contains(uri, "localhost") {
		t.Errorf("URI should contain host: %s", uri)
	}
}

func TestHandleAdminCredentials_NotSuperadmin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/credentials", nil)
	req.Header.Set("X-User-Role", "user")
	rr := httptest.NewRecorder()
	handleAdminCredentials(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestHandleAdminCredentialsItem_NotSuperadmin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/admin/credentials/abc", nil)
	req.Header.Set("X-User-Role", "user")
	rr := httptest.NewRecorder()
	handleAdminCredentialsItem(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
