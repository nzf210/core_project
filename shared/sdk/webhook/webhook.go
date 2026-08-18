package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type EventPayload struct {
	EventName string      `json:"event_name"`
	TenantID  string      `json:"tenant_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// DispatchEvent sends an async HTTP POST to n8n webhook.
func DispatchEvent(eventName, tenantID string, data interface{}) {
	// Typically, n8n webhook URL is set via env variable or hardcoded for internal network
	webhookURL := os.Getenv("N8N_WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "http://n8n:5678/webhook/saas-events"
	}

	payload := EventPayload{
		EventName: eventName,
		TenantID:  tenantID,
		Timestamp: time.Now(),
		Data:      data,
	}

	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			slog.Error("Failed to marshal webhook payload", "error", err, "event", eventName)
			return
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookURL, bytes.NewBuffer(body))
		if err != nil {
			slog.Error("Failed to create webhook request", "error", err, "event", eventName)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("Failed to dispatch webhook to n8n", "error", err, "event", eventName)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			slog.Warn("n8n webhook returned non-success status", "status", resp.StatusCode, "event", eventName)
		} else {
			slog.Info("Successfully dispatched webhook to n8n", "event", eventName, "tenant", tenantID)
		}
	}()
}
