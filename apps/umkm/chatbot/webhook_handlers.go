package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"log/slog"
)

// handleWAWebhook processes incoming WhatsApp messages from wa-gateway internal (whatsmeow)
func handleWAWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read raw body to parse JSON
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	slog.Info("Raw Webhook Body", "body", string(bodyBytes))
	// Restore body for any subsequent reader if needed
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var payload struct {
		Sender  string `json:"sender"`
		Message string `json:"message"`
	}

	sender := ""
	message := ""

	// Try parsing as JSON first
	if err := json.Unmarshal(bodyBytes, &payload); err == nil && payload.Sender != "" {
		sender = payload.Sender
		message = payload.Message
	} else {
		// Fallback to FormValue if not JSON
		r.ParseMultipartForm(10 << 20)
		sender = r.FormValue("sender")
		message = r.FormValue("message")
	}

	slog.Info("Received WA Webhook", "sender", sender, "message", message)

	tenantID := r.URL.Query().Get("tenant_id")

	// Respond immediately to avoid timeout from webhook provider
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":true,"message":"queued"}`))

	// Enqueue the job for async processing via Redis
	job := ChatJob{Sender: sender, Message: message, TenantID: tenantID}
	jobBytes, err := json.Marshal(job)
	if err == nil {
		errRedis := redisClient.LPush(r.Context(), redisQueueKey, jobBytes).Err()
		if errRedis != nil {
			slog.Error("Failed to enqueue job to Redis", "sender", sender, "error", errRedis)
		} else {
			slog.Info("Job queued to Redis successfully", "sender", sender)
		}
	} else {
		slog.Error("Failed to marshal chat job", "error", err)
	}
}