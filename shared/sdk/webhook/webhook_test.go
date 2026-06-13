package webhook

import (
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
	orig := os.Getenv("N8N_WEBHOOK_URL")
	os.Unsetenv("N8N_WEBHOOK_URL")
	defer os.Setenv("N8N_WEBHOOK_URL", orig)

	// Should not panic even without URL (just log error in goroutine)
	DispatchEvent("test.event", "tenant-1", map[string]string{"key": "val"})
	// Give goroutine time to run (it will fail to connect but should not panic)
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchEvent_InvalidURL(t *testing.T) {
	os.Setenv("N8N_WEBHOOK_URL", "http://invalid-host-that-does-not-exist:99999/webhook")
	defer os.Unsetenv("N8N_WEBHOOK_URL")

	// Should not panic, should log error in goroutine
	DispatchEvent("test.event", "tenant-1", nil)
	time.Sleep(100 * time.Millisecond)
}
