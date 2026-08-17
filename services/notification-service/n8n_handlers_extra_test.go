package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleN8NWhatsApp_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/n8n/whatsapp", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()
	handleN8NWhatsApp(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleN8NWhatsApp_MissingFields(t *testing.T) {
	payload := map[string]interface{}{"tenant_id": "t1"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/n8n/whatsapp", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handleN8NWhatsApp(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing fields, got %d", w.Code)
	}
}

func TestHandleN8NTelegram_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/n8n/telegram", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	handleN8NTelegram(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleN8NTelegram_MissingTenantID(t *testing.T) {
	payload := map[string]interface{}{"template": "hello"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/n8n/telegram", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handleN8NTelegram(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing tenant_id, got %d", w.Code)
	}
}

func TestHandleN8NTelegram_MissingChatID(t *testing.T) {
	payload := map[string]interface{}{
		"tenant_id": "t1",
		"template":  "Hello {{name}}",
		"data":      map[string]interface{}{"name": "Budi"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/n8n/telegram", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handleN8NTelegram(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing chat_id, got %d", w.Code)
	}
}
