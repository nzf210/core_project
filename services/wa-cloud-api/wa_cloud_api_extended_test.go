package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── handleSend ───────────────────────────────────────────────────────────────

func TestHandleSendCloudAPI_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/send", nil)
	rr := httptest.NewRecorder()
	handleSend(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleSend_MissingTenantHeader(t *testing.T) {
	body := strings.NewReader(`{"to":"6281234","type":"text","text":"hi"}`)
	req := httptest.NewRequest(http.MethodPost, "/send", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handleSend(rr, req)

	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 400 or 401 for missing tenant, got %d", rr.Code)
	}
}

func TestHandleSend_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-123")
	rr := httptest.NewRecorder()
	handleSend(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

// ─── handleWebhook ────────────────────────────────────────────────────────────

func TestHandleWebhook_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/webhook", nil)
	rr := httptest.NewRecorder()
	handleWebhook(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

// ─── handleValidateCredential ─────────────────────────────────────────────────

func TestHandleValidateCredential_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/validate", nil)
	req.Header.Set("X-User-Role", "superadmin")
	rr := httptest.NewRecorder()
	handleValidateCredential(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleValidateCredential_NotSuperadmin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/validate", nil)
	req.Header.Set("X-User-Role", "owner")
	rr := httptest.NewRecorder()
	handleValidateCredential(rr, req)

	if rr.Code != http.StatusForbidden && rr.Code != http.StatusBadRequest {
		t.Errorf("expected 403 or 400 for non-superadmin, got %d", rr.Code)
	}
}

func TestHandleValidateCredential_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader("bad"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "superadmin")
	rr := httptest.NewRecorder()
	handleValidateCredential(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

// ─── normalizeTo edge cases ───────────────────────────────────────────────────

func TestNormalizeTo_WithSpacesAndDashes(t *testing.T) {
	result := normalizeTo("+62 812-3456-7890")
	if result != "6281234567890" {
		t.Errorf("expected 6281234567890, got %q", result)
	}
}

func TestNormalizeTo_AlreadyNormalized(t *testing.T) {
	result := normalizeTo("6281234567890")
	if result != "6281234567890" {
		t.Errorf("expected unchanged, got %q", result)
	}
}

func TestNormalizeTo_ZeroPrefix(t *testing.T) {
	result := normalizeTo("081234567890")
	if result != "6281234567890" {
		t.Errorf("expected 6281234567890, got %q", result)
	}
}
