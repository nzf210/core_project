package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

func handleMessageEvent(tenantID string, v *events.Message) {
	if v.Info.IsFromMe {
		return
	}

	senderJID := v.Info.Sender.String()
	senderPhone := v.Info.Sender.User
	messageText := extractMessageText(v)

	slog.Info("Message received", "tenant_id", tenantID, "sender", senderJID, "text", messageText)

	if senderJID != "" && senderPhone != "" {
		mapUserJIDIfNeeded(senderJID, senderPhone)
	}

	rawText := strings.TrimSpace(messageText)
	upperText := strings.ToUpper(rawText)

	if handleActiveSession(tenantID, senderJID, rawText, upperText) {
		return
	}

	if handleCommandMessage(tenantID, senderJID, senderPhone, upperText) {
		return
	}

	forwardToN8NChatbot(tenantID, senderJID, senderPhone, messageText)
}

func extractMessageText(v *events.Message) string {
	if v.Message.Conversation != nil {
		return *v.Message.Conversation
	}
	if v.Message.ExtendedTextMessage != nil && v.Message.ExtendedTextMessage.Text != nil {
		return *v.Message.ExtendedTextMessage.Text
	}
	return ""
}

func handleActiveSession(tenantID, senderJID, rawText, upperText string) bool {
	if session, exists := loadRegSession(senderJID); exists {
		return handleWARegistrationStep(tenantID, session, rawText, upperText)
	}
	if session, exists := loadPWResetSession(senderJID); exists {
		return handleWAPasswordResetStep(tenantID, session, rawText, upperText)
	}
	return false
}

func handleCommandMessage(tenantID, senderJID, senderPhone, upperText string) bool {
	if strings.HasPrefix(upperText, "VERIF ") {
		code := strings.TrimSpace(upperText[6:])
		handleWAVerifyOTP(tenantID, senderJID, code)
		return true
	}

	if upperText == "OTP" {
		handleWAOTPRequest(tenantID, senderJID, senderPhone)
		return true
	}

	if isSixDigitOTP(upperText) {
		handleWALoginOTPReply(tenantID, senderJID, senderPhone, upperText)
		return true
	}

	if isRegistrationCommand(upperText) {
		startWARegistration(tenantID, senderJID, senderPhone)
		return true
	}

	if isPasswordResetCommand(upperText) {
		startWAPasswordReset(tenantID, senderJID, senderPhone)
		return true
	}

	if isHelpCommand(upperText) {
		sendHelpMenu(tenantID, senderJID)
		return true
	}

	return false
}

func isRegistrationCommand(text string) bool {
	return text == "REG" || text == "REGISTER" || text == "DAFTAR"
}

func isPasswordResetCommand(text string) bool {
	return text == "RESET" || text == "LUPA PASSWORD"
}

func isHelpCommand(text string) bool {
	return text == "HELP" || text == "BANTUAN" || text == "MENU"
}

func sendHelpMenu(tenantID, senderJID string) {
	menu := `🤖 *WCH Platform - Menu Bantuan*

📝 *Pendaftaran & Login:*
• REG - Daftar akun baru
• OTP - Minta kode login
• RESET - Reset password

💬 *Chatbot:*
Kirim pesan apa saja untuk berbicara dengan AI assistant

📞 *Bantuan:*
• HELP - Menu ini

Ketik perintah yang Anda butuhkan!`

	sendWAMessage(tenantID, senderJID, menu)
}

func forwardToN8NChatbot(tenantID, senderJID, senderPhone, messageText string) {
	n8nURL := getN8NWebhookURL()
	if n8nURL == "" {
		slog.Warn("N8N webhook URL not configured")
		sendWAMessage(tenantID, senderJID, "❌ Chatbot service tidak tersedia saat ini.")
		return
	}

	payload := map[string]interface{}{
		"tenant_id":    tenantID,
		"sender_jid":   senderJID,
		"sender_phone": senderPhone,
		"message":      messageText,
		"timestamp":    time.Now().Unix(),
		"platform":     "whatsapp",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", n8nURL+"/webhook/chatbot", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentTypeJSON)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Failed to forward message to N8N", "error", err)
		sendWAMessage(tenantID, senderJID, "❌ Maaf, terjadi kesalahan. Silakan coba lagi.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("N8N returned error", "status", resp.StatusCode)
		sendWAMessage(tenantID, senderJID, "❌ Maaf, layanan sedang sibuk. Silakan coba lagi.")
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		// N8N universal_chatbot workflow returns "response" field
		for _, key := range []string{"response", "reply"} {
			if msg, ok := result[key].(string); ok && msg != "" {
				sendWAMessage(tenantID, senderJID, msg)
				return
			}
		}
	}

	slog.Info("Message forwarded to N8N chatbot", "tenant_id", tenantID, "sender", senderJID)
}

func getN8NWebhookURL() string {
	u := os.Getenv("N8N_WEBHOOK_URL")
	if u == "" {
		u = "http://n8n-main:5678"
	}
	return strings.TrimRight(u, "/")
}

func getAuthServiceURL() string {
	if url := os.Getenv("AUTH_SERVICE_URL"); url != "" {
		return url
	}
	return "http://auth-service:8001"
}
