package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

var (
	memDedupMu sync.Mutex
	memDedup   = make(map[string]time.Time)
)

func isDuplicateMessage(ctx context.Context, msgID string) bool {
	if msgID == "" {
		return false
	}
	if redisShared != nil {
		ok, err := redisShared.SetNX(ctx, "wa:msg-dedup:"+msgID, "1", 2*time.Minute).Result()
		if err == nil {
			return !ok
		}
	}
	// In-memory fallback (tests / when Redis is offline)
	memDedupMu.Lock()
	defer memDedupMu.Unlock()
	now := time.Now()
	if len(memDedup) > 500 {
		for id, expiry := range memDedup {
			if now.After(expiry) {
				delete(memDedup, id)
			}
		}
	}
	if expiry, exists := memDedup[msgID]; exists && now.Before(expiry) {
		return true
	}
	memDedup[msgID] = now.Add(2 * time.Minute)
	return false
}

func handleMessageEvent(tenantID string, v *events.Message) {
	if v.Info.IsFromMe {
		return
	}

	ctx := context.Background()
	msgID := v.Info.ID
	if msgID != "" && isDuplicateMessage(ctx, msgID) {
		slog.Debug("handleMessageEvent: duplicate message ignored", "tenant_id", tenantID, "msg_id", msgID)
		return
	}

	// Normalize sender JID: strip agent/device part supaya session key & reply
	// konsisten. WhatsApp bisa kirim JID sama dengan device berbeda (user@lid vs
	// user:9@lid) → tanpa normalize, session state hilang antar-step & reply gagal.
	senderJID := v.Info.Sender.ToNonAD().String()
	senderPhone := resolveSenderPhone(ctx, tenantID, v)
	messageText := extractMessageText(v)

	slog.Info("Message received", "tenant_id", tenantID, "sender", senderJID, "phone", senderPhone, "text", messageText)

	if senderJID != "" && senderPhone != "" {
		mapUserJIDIfNeeded(senderJID, senderPhone)
	}

	rawText := strings.TrimSpace(messageText)
	if rawText == "" {
		// Event tanpa pesan teks (misal protocol message, reaction, sync, atau media tanpa caption)
		// TIDAK BOLEH diproses sebagai input percakapan / step wizard.
		return
	}
	upperText := strings.ToUpper(rawText)

	// Isolasi jalur: Perintah registrasi, OTP, reset password, dan menu platform
	// HANYA diproses jika pesan masuk ke nomor WA System / Platform ("system", "platform", atau kosong).
	// Untuk nomor WA tenant UMKM (UUID), semua pesan adalah percakapan dengan pelanggan toko/klinik
	// dan harus ditangani langsung oleh AI CS Toko (tidak boleh di-intercept menu WCH Platform).
	if isSystemTenant(tenantID) {
		if handleActiveSession(tenantID, senderJID, rawText, upperText) {
			return
		}

		if handleCommandMessage(tenantID, senderJID, senderPhone, upperText) {
			return
		}
	}

	// Dispatch asynchronous to avoid blocking the whatsmeow message receiving loop
	// while waiting for N8N and LLM inference response.
	go forwardToN8NChatbot(tenantID, senderJID, senderPhone, messageText)
}

func isSystemTenant(tenantID string) bool {
	t := strings.ToLower(strings.TrimSpace(tenantID))
	return t == "" || t == "system" || t == "platform" || t == "wch"
}

func extractMessageText(v *events.Message) string {
	if v == nil || v.Message == nil {
		return ""
	}
	if v.Message.Conversation != nil {
		return *v.Message.Conversation
	}
	if v.Message.ExtendedTextMessage != nil && v.Message.ExtendedTextMessage.Text != nil {
		return *v.Message.ExtendedTextMessage.Text
	}
	if v.Message.ImageMessage != nil && v.Message.ImageMessage.Caption != nil {
		return *v.Message.ImageMessage.Caption
	}
	if v.Message.VideoMessage != nil && v.Message.VideoMessage.Caption != nil {
		return *v.Message.VideoMessage.Caption
	}
	if v.Message.DocumentMessage != nil && v.Message.DocumentMessage.Caption != nil {
		return *v.Message.DocumentMessage.Caption
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
	slog.Info("handleCommandMessage: received",
		"tenant_id", tenantID,
		"sender_jid", senderJID,
		"sender_phone", senderPhone,
		"upper_text", upperText)

	if strings.HasPrefix(upperText, "VERIF ") {
		code := strings.TrimSpace(upperText[6:])
		handleWAVerifyOTP(tenantID, senderJID, code)
		return true
	}

	if upperText == "OTP" {
		handleWAOTPRequest(tenantID, senderJID, senderPhone)
		return true
	}

	if strings.HasPrefix(upperText, "OTP ") {
		phoneArg := strings.TrimSpace(upperText[4:])
		handleWAOTPRequestWithPhone(tenantID, senderJID, senderPhone, phoneArg)
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

// n8nHTTPClient is a shared HTTP client with an optimized connection pool (Keep-Alive)
// to handle high-concurrency requests to N8N without socket exhaustion.
var n8nHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
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
	req, _ := http.NewRequestWithContext(context.Background(), "POST", n8nURL+"/webhook/chatbot/incoming", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentTypeJSON)

	resp, err := n8nHTTPClient.Do(req)
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
