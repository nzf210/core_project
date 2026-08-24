package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"core_project/shared/observability"
	"core_project/shared/sdk/response"
	"core_project/shared/sdk/webhook"
)

// ─────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────

const (
	serviceName     = "notification-service"
	contentTypeJSON = "application/json"
	headerContentType = "Content-Type"
)

// ─────────────────────────────────────────────
// Business Metrics
// ─────────────────────────────────────────────

var (
	notificationsSentTotal = observability.NewCounter("notifications_sent_total", "Total notifications sent", []string{"channel", "status"})
)

// ─────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────

type NotificationRequest struct {
	Type    string `json:"type"` // 'wa', 'telegram', 'email'
	Target  string `json:"target"`
	Message string `json:"message"`
	Subject string `json:"subject,omitempty"`
}

// ─────────────────────────────────────────────
// Main
// ─────────────────────────────────────────────

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	mux := http.NewServeMux()

	// Metrics endpoint
	mux.Handle("/metrics", observability.PrometheusHandler())

	mux.HandleFunc("/api/notification/send", handleSendNotification)
	mux.HandleFunc("/health", handleHealth)
	
	// Campaign Internal Alerts
	mux.HandleFunc("/api/notification/campaign/conflict", handleConflictAlertTrigger)

	// N8N Webhooks
	mux.Handle("/webhook/n8n/whatsapp", webhook.RequireN8NSecret(http.HandlerFunc(handleN8NWhatsApp)))
	mux.Handle("/webhook/n8n/telegram", webhook.RequireN8NSecret(http.HandlerFunc(handleN8NTelegram)))

	port := "8005"
	slog.Info("Notification Service listening", "port", port)
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        observability.Middleware(serviceName)(mux),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

// ─────────────────────────────────────────────
// Handler
// ─────────────────────────────────────────────

func handleSendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, response.MethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		tenantID = "global"
	}

	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, response.InvalidRequest, http.StatusBadRequest)
		return
	}

	var err error
	switch req.Type {
	case "telegram":
		err = sendTelegram(req.Target, req.Message)
	case "wa", "whatsapp":
		err = sendWA(tenantID, req.Target, req.Message)
	case "email":
		err = sendEmail(req.Target, req.Subject, req.Message)
	default:
		// Try all channels
		go func() {
			sendTelegram(req.Target, req.Message)
		}()
		go func() {
			sendWA(tenantID, req.Target, req.Message)
		}()
		err = sendEmail(req.Target, req.Subject, req.Message)
	}

	if err != nil {
		slog.Error("Notification failed", "type", req.Type, "error", err)
		http.Error(w, fmt.Sprintf("Failed to send %s notification", req.Type), http.StatusInternalServerError)
		return
	}

	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"type":    req.Type,
		"target":  req.Target,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerContentType, contentTypeJSON)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": serviceName,
		"status":  "ok",
	})
}

// ─────────────────────────────────────────────
// Telegram
// ─────────────────────────────────────────────

func sendTelegram(chatID, message string) error {
	return sendTelegramMedia(chatID, message, "", "")
}

func sendTextMessage(botToken, chatID, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, contentTypeJSON, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned %d", resp.StatusCode)
	}
	return nil
}

func downloadMedia(mediaURL string) (*http.Response, error) {
	parsedMediaURL, parseErr := url.Parse(mediaURL)
	if parseErr != nil || (parsedMediaURL.Scheme != "https" && parsedMediaURL.Scheme != "http") {
		return nil, fmt.Errorf("invalid media URL")
	}
	resp, err := http.Get(parsedMediaURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}
	return resp, nil
}

func createMediaUploadBody(chatID, message, mediaName string, mediaResp *http.Response) (*bytes.Buffer, *multipart.Writer, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("chat_id", chatID)
	_ = writer.WriteField("caption", message)
	_ = writer.WriteField("parse_mode", "Markdown")

	part, err := writer.CreateFormFile("document", mediaName)
	if err != nil {
		return nil, nil, err
	}
	_, err = io.Copy(part, mediaResp.Body)
	if err != nil {
		return nil, nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, nil, err
	}

	return body, writer, nil
}

func sendTelegramMedia(chatID, message, mediaURL, mediaName string) error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		slog.Warn("TELEGRAM_BOT_TOKEN not set, skipping telegram")
		return nil
	}

	if mediaURL == "" {
		return sendTextMessage(botToken, chatID, message)
	}

	mediaResp, err := downloadMedia(mediaURL)
	if err != nil {
		return err
	}
	defer mediaResp.Body.Close()

	if mediaName == "" {
		mediaName = "document.file"
	}

	body, writer, err := createMediaUploadBody(chatID, message, mediaName, mediaResp)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botToken)
	req, err := http.NewRequestWithContext(context.Background(), "POST", apiURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	tgResp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer tgResp.Body.Close()

	if tgResp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram sendDocument returned %d", tgResp.StatusCode)
	}

	return nil
}

// ─────────────────────────────────────────────
// WhatsApp (via wa-gateway)
// ─────────────────────────────────────────────

func sendWA(tenantID, target, message string) error {
	return sendWAMedia(tenantID, target, message, "", "")
}

func sendWAMedia(tenantID, target, message, mediaURL, mediaName string) error {
	waGatewayBase := os.Getenv("WA_GATEWAY_URL")
	if waGatewayBase == "" {
		waGatewayBase = "http://wa-gateway:8202"
	}
	waGatewayURL := waGatewayBase + "/api/wa/send"

	targetJID := normalizeWAJID(target)

	data := url.Values{}
	data.Set("target", targetJID)
	data.Set("message", message)
	data.Set("tenant_id", tenantID)
	if mediaURL != "" {
		data.Set("media_url", mediaURL)
	}
	if mediaName != "" {
		data.Set("media_name", mediaName)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", waGatewayURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set(headerContentType, "application/x-www-form-urlencoded")
	req.Header.Set("X-Message-Type", "system")
	req.Header.Set("X-Source", serviceName)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wa-gateway returned %d", resp.StatusCode)
	}
	return nil
}

func normalizeWAJID(target string) string {
	jid := target
	if !strings.Contains(jid, "@s.whatsapp.net") {
		jid = strings.TrimPrefix(jid, "+")
		if strings.HasPrefix(jid, "0") {
			jid = "62" + jid[1:]
		}
		jid = jid + "@s.whatsapp.net"
	}
	return jid
}

// ─────────────────────────────────────────────
// Email (SMTP)
// ─────────────────────────────────────────────

func sendEmail(to, subject, body string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpHost == "" {
		slog.Warn("SMTP not configured, skipping email")
		return nil // Don't fail if email not configured
	}

	if smtpPort == "" {
		smtpPort = "587"
	}
	if smtpFrom == "" {
		smtpFrom = smtpUser
	}

	headers := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
			"\r\n", smtpFrom, to, subject)

	fullBody := headers + "<html><body>" +
		"<div style=\"font-family: Arial, sans-serif; max-width: 600px; margin: auto;\">" +
		"<h2 style=\"color: #2c3e50;\">🎫 WCH Platform</h2>" +
		"<div style=\"white-space: pre-wrap;\">" + body + "</div>" +
		"<hr style=\"margin-top:20px; color:#ddd;\"/>" +
		"<p style=\"color:#888; font-size:12px;\">WCH Platform — Smart Business Automation</p>" +
		"</div></body></html>"

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	var auth smtp.Auth
	if smtpUser != "" && smtpPass != "" {
		auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	}

	err := smtp.SendMail(addr, auth, smtpFrom, []string{to}, []byte(fullBody))
	if err != nil {
		return fmt.Errorf("smtp send failed: %w", err)
	}

	slog.Info("Email sent", "to", to, "subject", subject)
	return nil
}