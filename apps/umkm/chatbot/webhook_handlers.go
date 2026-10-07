package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"core_project/shared/sdk/response"
	"log/slog"
)

// handleWAWebhook processes incoming WhatsApp messages from wa-gateway internal (whatsmeow)
func handleWAWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, response.MethodNotAllowed, http.StatusMethodNotAllowed)
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
	job := ChatJob{Sender: sender, Message: message, TenantID: tenantID}
	if redisClient == nil {
		http.Error(w, "Chat queue unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := enqueueChatJob(r.Context(), job, func(ctx context.Context, payload []byte) error {
		return redisClient.LPush(ctx, redisQueueKey, payload).Err()
	}); err != nil {
		slog.Error("Failed to enqueue chat job to Redis", "sender", sender, "error", err)
		http.Error(w, "Chat queue unavailable", http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusAccepted, APIResponse{Success: true, Message: "queued"})
}

func enqueueChatJob(ctx context.Context, job ChatJob, push func(context.Context, []byte) error) error {
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return push(ctx, jobBytes)
}
