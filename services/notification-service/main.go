package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	"core_project/shared/sdk/webhook"
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
	mux.HandleFunc("/api/notification/send", handleSendNotification)
	mux.HandleFunc("/health", handleHealth)
	
	// N8N Webhooks
	mux.Handle("/webhook/n8n/whatsapp", webhook.RequireN8NSecret(http.HandlerFunc(handleN8NWhatsApp)))

	port := "8005"
	slog.Info("Notification Service listening", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("Failed to start server", "error", err)
	}
}

// ─────────────────────────────────────────────
// Handler
// ─────────────────────────────────────────────

func handleSendNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = "global"
	}

	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"type":    req.Type,
		"target":  req.Target,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "notification-service",
		"status":  "ok",
	})
}

// ─────────────────────────────────────────────
// Telegram
// ─────────────────────────────────────────────

func sendTelegram(chatID, message string) error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		slog.Warn("TELEGRAM_BOT_TOKEN not set, skipping telegram")
		return nil
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned %d", resp.StatusCode)
	}
	return nil
}

// ─────────────────────────────────────────────
// WhatsApp (via wa-gateway)
// ─────────────────────────────────────────────

func sendWA(tenantID, target, message string) error {
	waGatewayURL := "http://localhost:8202/api/wa/send"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		waGatewayURL = "http://wa-gateway:8202/api/wa/send"
	}

	targetJID := normalizeWAJID(target)

	data := url.Values{}
	data.Set("target", targetJID)
	data.Set("message", message)
	data.Set("tenant_id", tenantID)

	req, err := http.NewRequest("POST", waGatewayURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Message-Type", "system")
	req.Header.Set("X-Source", "notification-service")

	resp, err := http.DefaultClient.Do(req)
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