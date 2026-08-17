package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)


func setupSendHandler() {
	http.HandleFunc("/api/wa/send", handleSendRequest)
}

func handleSendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.FormValue("tenant_id")
	target := r.FormValue("target")
	message := r.FormValue("message")
	mediaURL := r.FormValue("media_url")
	mediaName := r.FormValue("media_name")

	if tenantID == "" || target == "" || (message == "" && mediaURL == "") {
		http.Error(w, `{"error":"tenant_id, target, and (message or media_url) are required"}`, http.StatusBadRequest)
		return
	}

	pref := resolveProviderPreference(r, tenantID)
	if handleCloudAPIRouting(w, r, tenantID, target, message, pref) {
		return
	}

	if !rateLimiter.Allow(tenantID) {
		slog.Warn("Rate limit exceeded for whatsmeow", "tenant_id", tenantID)
		waRateLimitedTotal.WithLabelValues().Inc()
		http.Error(w, `{"error":"Rate limit exceeded (max 5 msg/min). Use Cloud API for broadcasting."}`, http.StatusTooManyRequests)
		return
	}

	if !acquireSessionLock(w, tenantID) {
		return
	}
	defer ReleaseSessionLock(context.Background(), tenantID)

	client := getWAClient(tenantID)
	if client == nil {
		http.Error(w, `{"error":"Not connected to WhatsApp. Please scan QR first."}`, http.StatusUnauthorized)
		return
	}

	if !ensureConnection(w, client, tenantID) {
		return
	}

	jid, err := types.ParseJID(target)
	if err != nil {
		http.Error(w, `{"error":"Invalid target JID"}`, http.StatusBadRequest)
		return
	}

	msg, err := buildWhatsAppMessage(client, tenantID, message, mediaURL, mediaName)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
		return
	}

	_, err = client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		slog.Error("Failed to send message", "tenant_id", tenantID, "target", jid.String(), "error", err)
		waMessagesTotal.WithLabelValues("whatsmeow", "out", "failed").Inc()
		http.Error(w, `{"error":"Failed to send message"}`, http.StatusInternalServerError)
		return
	}

	waMessagesTotal.WithLabelValues("whatsmeow", "out", "sent").Inc()
	waRoutedTotal.WithLabelValues("whatsmeow").Inc()
	w.Header().Set(headerContentType, mimeApplicationJSON)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
}

func acquireSessionLock(w http.ResponseWriter, tenantID string) bool {
	acquired, _ := AcquireSessionLock(context.Background(), tenantID)
	if !acquired {
		http.Error(w, `{"error":"Session is currently locked by another operation"}`, http.StatusLocked)
		return false
	}
	return true
}

func getWAClient(tenantID string) *whatsmeow.Client {
	clientMu.RLock()
	defer clientMu.RUnlock()
	client, exists := clientMap[tenantID]
	if !exists || client == nil {
		return nil
	}
	return client
}

func ensureConnection(w http.ResponseWriter, client *whatsmeow.Client, tenantID string) bool {
	if client.IsConnected() {
		return true
	}
	if !shouldReconnect(tenantID) {
		writeServiceUnavailable(w, "WhatsApp disconnected. Reconnect backoff active.")
		return false
	}
	slog.Info("Attempting reconnect before sending", "tenant_id", tenantID)
	if err := client.Connect(); err != nil {
		writeServiceUnavailable(w, "Failed to reconnect to WA server")
		return false
	}
	return true
}

func writeServiceUnavailable(w http.ResponseWriter, msg string) {
	w.Header().Set(headerContentType, mimeApplicationJSON)
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func handleCloudAPIRouting(w http.ResponseWriter, r *http.Request, tenantID, target, message, pref string) bool {
	if pref == "whatsmeow" {
		return false
	}

	shouldTryCloud := pref == "cloud_api" || (pref == "auto" && isTransactional(r))

	if !shouldTryCloud {
		return false
	}

	msgType := r.Header.Get("X-Message-Type")
	if msgType == "" {
		msgType = "text"
	}
	waMsgID, err := routeToCloudAPI(tenantID, target, message, msgType)
	if err == nil {
		waRoutedTotal.WithLabelValues("cloud_api").Inc()
		w.Header().Set(headerContentType, mimeApplicationJSON)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":       true,
			"routed":        "cloud_api",
			"wa_message_id": waMsgID,
		})
		return true
	}

	// Cloud API failed
	if pref == "cloud_api" {
		// Forced Cloud API → no fallback, return error
		slog.Error("Cloud API forced but failed", "tenant_id", tenantID, "error", err)
		http.Error(w, `{"error":"Cloud API forced but failed"}`, http.StatusBadGateway)
		return true
	}

	// auto mode → fallback to whatsmeow
	slog.Warn("Cloud API fallback to whatsmeow", "tenant_id", tenantID, "error", err)
	waFallbackTotal.WithLabelValues().Inc()
	return false
}

func buildWhatsAppMessage(client *whatsmeow.Client, tenantID, message, mediaURL, mediaName string) (*waE2E.Message, error) {
	if mediaURL == "" {
		return &waE2E.Message{
			Conversation: proto.String(message),
		}, nil
	}

	// Validate URL scheme before fetch to prevent SSRF
	parsedURL, parseErr := url.Parse(mediaURL)
	if parseErr != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") {
		return nil, fmt.Errorf("invalid media URL scheme")
	}

	resp, err := http.Get(parsedURL.String())
	if err != nil {
		slog.Error("Failed to download media", "tenant_id", tenantID, "media_url", mediaURL, "error", err)
		return nil, fmt.Errorf("failed to download media") //nolint:staticcheck
	}
	defer resp.Body.Close()
	mediaData, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read media", "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("failed to read media") //nolint:staticcheck
	}
	mimetype := resp.Header.Get(headerContentType)
	if mimetype == "" {
		mimetype = "application/octet-stream"
	}
	if mediaName == "" {
		mediaName = "document"
	}

	uploadResp, err := client.Upload(context.Background(), mediaData, whatsmeow.MediaDocument)
	if err != nil {
		slog.Error("Failed to upload media to WA", "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("failed to upload media") //nolint:staticcheck
	}

	return &waE2E.Message{
		DocumentMessage: &waE2E.DocumentMessage{
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			Mimetype:      proto.String(mimetype),
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(mediaData))),
			Title:         proto.String(mediaName),
			FileName:      proto.String(mediaName),
			Caption:       proto.String(message),
		},
	}, nil
}
