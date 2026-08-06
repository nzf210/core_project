package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderTemplate_StringSubstitution(t *testing.T) {
	tmpl := "Halo {{name}}, selamat datang di {{platform}}!"
	result := RenderTemplate(tmpl, map[string]interface{}{
		"name":     "Budi",
		"platform": "WCH",
	})
	if result != "Halo Budi, selamat datang di WCH!" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestRenderTemplate_NumericValue(t *testing.T) {
	tmpl := "Saldo: Rp {{amount}}"
	result := RenderTemplate(tmpl, map[string]interface{}{
		"amount": float64(50000),
	})
	if !strings.Contains(result, "50000") {
		t.Errorf("expected numeric substitution, got: %q", result)
	}
}

func TestRenderTemplate_MissingKey(t *testing.T) {
	tmpl := "Hello {{name}}, your code is {{code}}"
	result := RenderTemplate(tmpl, map[string]interface{}{
		"name": "Ani",
	})
	if !strings.Contains(result, "Ani") {
		t.Error("name should be substituted")
	}
	if !strings.Contains(result, "{{code}}") {
		t.Error("missing key placeholder should remain")
	}
}

func TestRenderTemplate_EmptyTemplate(t *testing.T) {
	result := RenderTemplate("", map[string]interface{}{"key": "val"})
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestRenderTemplate_EmptyData(t *testing.T) {
	tmpl := "No substitution here"
	result := RenderTemplate(tmpl, map[string]interface{}{})
	if result != tmpl {
		t.Errorf("template should be unchanged, got %q", result)
	}
}

func TestHandleN8NWhatsApp_MissingBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/n8n/whatsapp", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleN8NWhatsApp(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleN8NWhatsApp_MissingRequiredFields(t *testing.T) {
	body := strings.NewReader(`{"tenant_id":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/n8n/whatsapp", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleN8NWhatsApp(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", rr.Code)
	}
}

func TestHandleN8NTelegram_MissingBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/n8n/telegram", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleN8NTelegram(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleConflictAlertTrigger_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/alerts/conflict", nil)
	rr := httptest.NewRecorder()
	handleConflictAlertTrigger(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleConflictAlertTrigger_MissingBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/alerts/conflict", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleConflictAlertTrigger(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
