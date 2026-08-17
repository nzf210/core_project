package webhook

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestEventPayload_Fields(t *testing.T) {
	payload := EventPayload{
		EventName: "test.event",
		TenantID:  "tenant-1",
		Timestamp: time.Now(),
		Data:      map[string]string{"key": "value"},
	}
	if payload.EventName != "test.event" {
		t.Errorf("expected test.event, got %q", payload.EventName)
	}
	if payload.TenantID != "tenant-1" {
		t.Errorf("expected tenant-1, got %q", payload.TenantID)
	}
	if payload.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestDispatchEvent_NoURL(t *testing.T) {
	os.Setenv("N8N_WEBHOOK_URL", "http://127.0.0.1:65535/webhook")
	defer os.Unsetenv("N8N_WEBHOOK_URL")
	DispatchEvent("test.event", "tenant-1", map[string]string{"key": "val"})
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchEvent_InvalidURL(t *testing.T) {
	os.Setenv("N8N_WEBHOOK_URL", "http://invalid-host-that-does-not-exist:99999/webhook")
	defer os.Unsetenv("N8N_WEBHOOK_URL")
	DispatchEvent("test.event", "tenant-1", nil)
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchEvent_SuccessResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	os.Setenv("N8N_WEBHOOK_URL", srv.URL)
	defer os.Unsetenv("N8N_WEBHOOK_URL")

	DispatchEvent("order.created", "tenant-1", map[string]string{"order_id": "123"})
	time.Sleep(200 * time.Millisecond)
}

func TestDispatchEvent_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	os.Setenv("N8N_WEBHOOK_URL", srv.URL)
	defer os.Unsetenv("N8N_WEBHOOK_URL")

	DispatchEvent("order.failed", "tenant-1", nil)
	time.Sleep(200 * time.Millisecond)
}

func TestDispatchEvent_DefaultURL(t *testing.T) {
	os.Unsetenv("N8N_WEBHOOK_URL")
	// Default URL (n8n:5678) won't be reachable — just verify no panic
	DispatchEvent("test.event", "tenant-1", nil)
	time.Sleep(100 * time.Millisecond)
}

