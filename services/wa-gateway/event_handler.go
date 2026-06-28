package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var globalContainer *sqlstore.Container

func setContainer(c *sqlstore.Container) {
	globalContainer = c
}

// restoreSingleSession restores a specific tenant's WA session in-memory.
// Used as fallback when sendWAMessage finds clientMap empty.
func restoreSingleSession(tenantID string) {
	if globalContainer == nil || db == nil {
		slog.Warn("restoreSingleSession: cannot restore, container or db nil")
		return
	}

	var jidStr string
	err := db.QueryRow(`SELECT jid FROM wa_tenant_sessions WHERE tenant_id = $1`, tenantID).Scan(&jidStr)
	if err != nil || jidStr == "" {
		slog.Warn("restoreSingleSession: no session in DB", "tenant_id", tenantID)
		return
	}

	ctx := context.Background()
	if owned, _ := AcquireSessionLock(ctx, tenantID); !owned {
		slog.Info("restoreSingleSession: session owned by another instance, skipping", "tenant_id", tenantID)
		return
	}

	jid, _ := types.ParseJID(jidStr)
	device, _ := globalContainer.GetDevice(ctx, jid)
	if device == nil {
		slog.Warn("restoreSingleSession: no device in whatsmeow store", "tenant_id", tenantID)
		ReleaseSessionLock(ctx, tenantID)
		return
	}

	client := whatsmeow.NewClient(device, waLog.Stdout("Client-"+tenantID, "INFO", true))
	client.AddEventHandler(func(evt interface{}) { eventHandler(tenantID, evt) })
	if err := client.Connect(); err == nil {
		clientMu.Lock()
		clientMap[tenantID] = client
		clientMu.Unlock()
		slog.Info("restoreSingleSession: session restored", "tenant_id", tenantID)
	} else {
		slog.Error("restoreSingleSession: failed to connect", "tenant_id", tenantID, "error", err)
		ReleaseSessionLock(ctx, tenantID)
	}
}

// eventHandler handles WhatsApp events for a tenant
func eventHandler(tenantID string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		handleMessageEvent(tenantID, v)
	case *events.Connected:
		handleConnectedEvent(tenantID)
	}
}

const (
	authServiceURLDefault = "http://auth-service:8001"
	contentTypeJSON       = "application/json"
)

// waRegistrationSession stores in-progress WA registration conversation state.
// ponytail: in-memory map, per-instance only. Restart clears sessions.
// Add Redis persistence if HA/scale needed.
var waRegistrationSessions = make(map[string]*waRegistrationSession)

type waRegistrationSession struct {
	SenderJID    string
	PhoneNumber  string
	Step         int
	BusinessName string
	BusinessType string
	Password     string
	Username     string
	CreatedAt    time.Time
}

func handleMessageEvent(tenantID string, v *events.Message) {
	if v.Info.IsGroup || v.Info.IsFromMe || time.Since(v.Info.Timestamp) > 5*time.Minute {
		slog.Debug("handleMessageEvent: filtered message",
			"is_group", v.Info.IsGroup,
			"is_from_me", v.Info.IsFromMe,
			"timestamp_age_sec", time.Since(v.Info.Timestamp).Seconds(),
			"sender", v.Info.Sender.ToNonAD().String())
		return
	}

	rawText := extractMessageText(v)
	if rawText == "" {
		slog.Debug("handleMessageEvent: empty message text", "sender", v.Info.Sender.ToNonAD().String())
		return
	}

	slog.Info("handleMessageEvent: incoming message",
		"sender", v.Info.Sender.ToNonAD().String(),
		"text", rawText,
		"tenant_id", tenantID)
	upperText := strings.TrimSpace(strings.ToUpper(rawText))

	senderJID := v.Info.Sender.ToNonAD().String()
	senderPhone := extractPhoneFromJID(senderJID)

	// Map sender JID → registered phone so future OTP requests can find this user
	mapUserJIDIfNeeded(senderJID, senderPhone)

	// ─── Keyword-based routing ───────────────────────────────────────
	// REG or DAFTAR → start WA-only registration flow
	if upperText == "REG" || upperText == "DAFTAR" {
		startWARegistration(tenantID, senderJID, senderPhone)
		return
	}

	// OTP [phone] → trigger login OTP via auth-service
	// If phone included: use it directly (handles LID masking / different WA phone)
	// If just "OTP": try value-scan fallback
	if upperText == "OTP" {
		handleWAOTPRequest(tenantID, senderJID, senderPhone)
		return
	}
	if strings.HasPrefix(upperText, "OTP ") {
		phoneInMsg := strings.TrimSpace(strings.TrimPrefix(upperText, "OTP "))
		phoneInMsg = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, phoneInMsg)
		if len(phoneInMsg) >= 8 {
			handleWAOTPRequest(tenantID, senderJID, phoneInMsg)
			return
		}
	}

	// VERIF {code} → verify OTP for web-based registration
	if strings.HasPrefix(upperText, "VERIF ") {
		code := strings.TrimSpace(strings.ToUpper(strings.TrimPrefix(upperText, "VERIF")))
		handleWAVerifyOTP(tenantID, senderJID, code)
		return
	}

	// 6-digit OTP reply → verify phone login OTP
	if isSixDigitOTP(upperText) {
		handleWALoginOTPReply(tenantID, senderJID, senderPhone, upperText)
		return
	}

	// WA registration conversation step reply
	if session, ok := waRegistrationSessions[senderJID]; ok {
		if handleWARegistrationStep(tenantID, session, rawText, upperText) {
			return
		}
	}

	// ─── Default: forward to N8N / chatbot ──────────────────────────
	jsonBody, _ := json.Marshal(map[string]interface{}{
		"tenant_id":   tenantID,
		"sender_jid":  senderJID,
		"sender_name": v.Info.PushName,
		"message":     rawText,
		"timestamp":   v.Info.Timestamp.Unix(),
		"source":      "whatsmeow",
	})

	if tryForwardToN8N(tenantID, jsonBody) {
		return
	}
	forwardToChatbot(tenantID, jsonBody)
}

func extractPhoneFromJID(jid string) string {
	phone := jid
	if idx := strings.Index(jid, "@"); idx > 0 {
		phone = jid[:idx]
	}
	// Normalize to local format (DB stores 08xx, JID is 62xx)
	if strings.HasPrefix(phone, "62") {
		phone = "0" + phone[2:]
	}
	return phone
}

func isSixDigitOTP(text string) bool {
	return len(text) == 6 && len(strings.TrimLeft(text, "0123456789")) == 0
}

// ─── REG / DAFTAR → Full WA Registration ──────────────────────────

func startWARegistration(tenantID, senderJID, senderPhone string) {
	// Check if already has account
	if db != nil {
		var exists bool
		db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1)", senderPhone).Scan(&exists)
		if exists {
			sendWAMessage(tenantID, senderJID, "Nomor ini sudah terdaftar. Gunakan menu LOGIN atau hubungi admin.")
			return
		}
	}

	// Also check Redis for pending web registration with this phone
	if redisShared != nil {
		ctx := context.Background()
		otpKey := "otp:" + senderPhone
		if val, _ := redisShared.Get(ctx, otpKey).Result(); val != "" {
			sendWAMessage(tenantID, senderJID, "Nomor ini sedang dalam proses pendaftaran di website. Selesaikan verifikasi di website terlebih dahulu, atau tunggu 1 jam.")
			return
		}
	}

	// Start session
	waRegistrationSessions[senderJID] = &waRegistrationSession{
		SenderJID:   senderJID,
		PhoneNumber: senderPhone,
		Step:        1,
		CreatedAt:   time.Now(),
	}

	sendWAMessage(tenantID, senderJID, "Halo! Selamat datang di WCH Platform.\n\nUntuk mendaftar, silakan jawab pertanyaan berikut:\n\n1️⃣ Nama bisnis Anda叫什么? (ketik nama toko/usaha Anda)")
}

func handleWARegistrationStep(tenantID string, session *waRegistrationSession, rawText, upperText string) bool {
	switch session.Step {
	case 1:
		session.BusinessName = strings.TrimSpace(rawText)
		session.Step = 2
		sendWAMessage(tenantID, session.SenderJID, "✅ Nama bisnis: "+session.BusinessName+"\n\n2️⃣ Tipe bisnis Anda?\n1. Umum\n2. Warung/Kedai\n3. Klinik\n\nKetik nomor (1-3):")
		return true
	case 2:
		bt := map[string]string{"1": "umum", "2": "warung", "3": "clinic"}
		btDesc := map[string]string{"1": "Umum", "2": "Warung/Kedai", "3": "Klinik"}
		btVal, ok := bt[upperText]
		if !ok {
			sendWAMessage(tenantID, session.SenderJID, "❌ Pilih angka 1, 2, atau 3 saja.")
			return true
		}
		session.BusinessType = btVal
		session.Step = 3
		sendWAMessage(tenantID, session.SenderJID, "✅ Tipe bisnis: "+btDesc[upperText]+"\n\n3️⃣ Buat username untuk login (huruf, angka, underscore, min 3 karakter):")
		return true
	case 3:
		if len(rawText) < 3 || len(strings.Trim(rawText, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")) != 0 {
			sendWAMessage(tenantID, session.SenderJID, "❌ Username minimal 3 karakter, hanya huruf, angka, dan underscore.")
			return true
		}
		if db != nil {
			var exists bool
			db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", rawText).Scan(&exists)
			if exists {
				sendWAMessage(tenantID, session.SenderJID, "❌ Username sudah digunakan. Coba yang lain:")
				return true
			}
		}
		session.Username = rawText
		session.Step = 4
		sendWAMessage(tenantID, session.SenderJID, "✅ Username: "+session.Username+"\n\n4️⃣ Buat password (minimal 6 karakter):")
		return true
	case 4:
		if len(rawText) < 6 {
			sendWAMessage(tenantID, session.SenderJID, "❌ Password minimal 6 karakter.")
			return true
		}
		session.Password = rawText
		session.Step = 5
		sendWAMessage(tenantID, session.SenderJID, "✅ Password tersimpan.\n\n5️⃣ Konfirmasi nomor HP: "+session.PhoneNumber+"\n\nKetik YA jika benar, atau ketik ulang nomor Anda:")
		return true
	case 5:
		if upperText != "YA" {
			if strings.HasPrefix(rawText, "62") && len(rawText) >= 10 {
				session.PhoneNumber = rawText
				sendWAMessage(tenantID, session.SenderJID, "✅ Nomor HP diperbarui: "+session.PhoneNumber+"\n\nKetik YA untuk konfirmasi:")
			} else {
				sendWAMessage(tenantID, session.SenderJID, "Ketik YA untuk konfirmasi, atau ketik nomor HP baru (format: 62812xxx):")
			}
			return true
		}
		session.Step = 6
		submitWARegistration(tenantID, session)
		return true
	}
	return false
}

func submitWARegistration(tenantID string, session *waRegistrationSession) {
	// Call auth-service internal registration endpoint
	authSvcURL := getAuthServiceURL()
	payload := map[string]interface{}{
		"phoneNumber":  session.PhoneNumber,
		"username":     session.Username,
		"password":     session.Password,
		"businessName": session.BusinessName,
		"businessType": session.BusinessType,
		"role":         "owner",
		"wa_jid":       session.SenderJID,
		"source":       "wa_registration",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(authSvcURL+"/register-wa", contentTypeJSON, bytes.NewReader(body))
	if err != nil || resp == nil {
		sendWAMessage(tenantID, session.SenderJID, "❌ Gagal mengirim data. Silakan coba lagi nanti atau hubungi admin.")
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sendWAMessage(tenantID, session.SenderJID, "🎉 Pendaftaran berhasil!\n\n📱 Username: "+session.Username+"\n🔐 Password: "+session.Password+"\n🏪 Bisnis: "+session.BusinessName+"\n\nSilakan login di aplikasi dengan nomor HP "+session.PhoneNumber)
	} else {
		msg := "Terjadi kesalahan"
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		sendWAMessage(tenantID, session.SenderJID, "❌ Gagal: "+msg+"\n\nKetik REG untuk mulai ulang.")
	}

	delete(waRegistrationSessions, session.SenderJID)
}

// ─── OTP → Trigger Login OTP ───────────────────────────────────────

func handleWAOTPRequest(tenantID, senderJID, senderPhone string) {
	ctx := context.Background()

	toLocal := func(p string) string {
		if strings.HasPrefix(p, "62") {
			return "0" + p[2:]
		}
		return p
	}

	localPhone := toLocal(senderPhone)

	// Try direct key lookup first
	for _, key := range []string{
		"auth:pending:" + senderPhone,
		"auth:pending:" + localPhone,
	} {
		v, e := redisShared.Get(ctx, key).Result()
		if e == nil && v != "" {
			generateAndSendOTP(ctx, tenantID, senderJID, senderPhone)
			return
		}
	}

	// Scan all auth:pending keys and match by VALUE (stored phone number).
	// This handles when user's login phone differs from their WhatsApp sender phone.
	keys, _ := redisShared.Keys(ctx, "auth:pending:*").Result()
	for _, key := range keys {
		v, e := redisShared.Get(ctx, key).Result()
		if e == nil && (v == senderPhone || v == localPhone) {
			slog.Info("handleWAOTPRequest: found via value scan",
				"senderPhone", senderPhone, "matchedKey", key)
			generateAndSendOTP(ctx, tenantID, senderJID, senderPhone)
			return
		}
	}

	// Fallback: look up registered phone by sender's WhatsApp JID
	if db != nil {
		var registeredPhone string
		err := db.QueryRow("SELECT phone_number FROM users WHERE wa_jid = $1", senderJID).Scan(&registeredPhone)
		if err != nil {
			shortJID := strings.Split(senderJID, ":")[0] + "@s.whatsapp.net"
			err = db.QueryRow("SELECT phone_number FROM users WHERE wa_jid = $1", shortJID).Scan(&registeredPhone)
		}
		if registeredPhone != "" {
			key := "auth:pending:" + toLocal(registeredPhone)
			if v, e := redisShared.Get(ctx, key).Result(); e == nil && v != "" {
				slog.Info("handleWAOTPRequest: found via wa_jid lookup",
					"senderJID", senderJID, "registeredPhone", registeredPhone)
				generateAndSendOTP(ctx, tenantID, senderJID, senderPhone)
				return
			}
		}
	}

	slog.Warn("OTP request without pending login", "phone", senderPhone)
	sendWAMessage(tenantID, senderJID, "❌ Tidak ada permintaan login. Silakan coba login di website terlebih dahulu.")
}

func generateAndSendOTP(ctx context.Context, tenantID, senderJID, senderPhone string) {
	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	// Normalize to 0xxx so auth-service verify lookup is consistent
	normPhone := senderPhone
	if strings.HasPrefix(normPhone, "62") {
		normPhone = "0" + normPhone[2:]
	}
	otpKey := "phone-login-otp:" + normPhone
	redisShared.Set(ctx, otpKey, otp+"|"+normPhone, 1*time.Hour)
	sendWAMessage(tenantID, senderJID, "📩 Kode OTP Anda: *"+otp+"*\n\nBalas pesan ini dengan 6 digit kode OTP tersebut.\n\nContoh: 123456")
	slog.Info("OTP generated & sent via WA Center", "phone", normPhone, "otp", otp)
}

// ─── VERIF {code} → Verify Web Registration OTP ───────────────────

func handleWAVerifyOTP(tenantID, senderJID, code string) {
	if len(code) != 6 {
		sendWAMessage(tenantID, senderJID, "❌ Format salah. Ketik: VERIF <6-digit-kode>\n\nContoh: VERIF 123456")
		return
	}

	authSvcURL := getAuthServiceURL()
	payload := map[string]interface{}{
		"code":       code,
		"source":     "wa",
		"sender_jid": senderJID,
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(authSvcURL+"/verify-otp-wa", contentTypeJSON, bytes.NewReader(body))
	if err != nil || resp == nil {
		sendWAMessage(tenantID, senderJID, "❌ Gagal memverifikasi. Silakan coba lagi.")
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sendWAMessage(tenantID, senderJID, "🎉 Pendaftaran berhasil! Silakan login dengan nomor HP Anda.")
	} else {
		msg := "Kode verifikasi salah atau expired."
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		sendWAMessage(tenantID, senderJID, "❌ "+msg)
	}
}

// ─── 6-digit reply → Verify Login OTP ─────────────────────────────

func handleWALoginOTPReply(tenantID, senderJID, senderPhone, code string) {
	authSvcURL := getAuthServiceURL()
	payload := map[string]interface{}{
		"phoneNumber": senderPhone,
		"otp":         code,
		"source":      "wa",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(authSvcURL+"/verify-phone-login-wa", contentTypeJSON, bytes.NewReader(body))
	if err != nil || resp == nil {
		sendWAMessage(tenantID, senderJID, "❌ Gagal verifikasi. Silakan coba lagi.")
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sendWAMessage(tenantID, senderJID, "🎉 Login berhasil! Silakan buka aplikasi untuk melanjutkan.")
	} else {
		msg := "Kode OTP salah atau expired."
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		sendWAMessage(tenantID, senderJID, "❌ "+msg+"\n\nKetik OTP untuk mengirim ulang kode.")
	}
}

// ─── Send WA Message via whatsmeow ────────────────────────────────

func sendWAMessage(tenantID, targetJID, message string) {
	clientMu.RLock()
	client := clientMap[tenantID]
	clientMu.RUnlock()

	if client == nil || !client.IsConnected() {
		slog.Warn("sendWAMessage: client not connected, attempting restore", "tenant_id", tenantID)
		go restoreSingleSession(tenantID)
		return
	}

	jid, err := types.ParseJID(targetJID)
	if err != nil {
		slog.Error("Invalid JID for sendWAMessage", "jid", targetJID, "error", err)
		return
	}

	msg := &waE2E.Message{
		Conversation: proto.String(message),
	}

	_, err = client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		slog.Error("Failed to send WA reply", "tenant_id", tenantID, "target", targetJID, "error", err)
	} else {
		slog.Info("Sent WA reply", "tenant_id", tenantID, "target", targetJID, "len", len(message))
	}
}

func getAuthServiceURL() string {
	if url := os.Getenv("AUTH_SERVICE_URL"); url != "" {
		return url
	}
	// Native dev uses localhost, Docker uses service name
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		return authServiceURLDefault
	}
	return "http://localhost:8001"
}

func extractMessageText(v *events.Message) string {
	if v.Message.GetConversation() != "" {
		return v.Message.GetConversation()
	}
	if v.Message.GetExtendedTextMessage() != nil {
		return v.Message.GetExtendedTextMessage().GetText()
	}
	return ""
}

func tryForwardToN8N(tenantID string, jsonBody []byte) bool {
	if db == nil {
		return false
	}
	var n8nURL string
	err := db.QueryRow(`SELECT n8n_webhook_url FROM tenant_chatbot_configs WHERE tenant_id = $1 AND is_active = true`, tenantID).Scan(&n8nURL)
	if err != nil || n8nURL == "" {
		return false
	}
	resp, err := http.Post(n8nURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		slog.Error("Failed to forward message to N8N", "error", err)
		return true // still handled, don't fall through
	}
	resp.Body.Close()
	return true
}

func forwardToChatbot(tenantID string, jsonBody []byte) {
	chatbotURL := os.Getenv("UMKM_CHATBOT_URL")
	if chatbotURL == "" {
		if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
			chatbotURL = "http://umkm-chatbot:8203"
		} else {
			chatbotURL = "http://localhost:8203"
		}
	}
	resp, err := http.Post(chatbotURL+"/webhook/wa?tenant_id="+tenantID, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		slog.Error("Failed to forward message to chatbot", "error", err)
	} else {
		resp.Body.Close()
	}
}

// invalidatePlatformWAProviderCache removes the Redis override so auto-detect picks up the new state.
// AC-8: called on connect/disconnect so auth-service sees the correct provider immediately.
func invalidatePlatformWAProviderCache() {
	if redisShared == nil {
		return
	}
	ctx := context.Background()
	// Deleting forces getPlatformWAProvider to re-detect fresh (no cache to read)
	redisShared.Del(ctx, "platform:wa:provider")
	slog.Info("platform:wa:provider cache invalidated")
}

// mapUserJIDIfNeeded stores senderJID → phone_number in users.wa_jid
// so OTP requests from this WA device can find the registered user.
func mapUserJIDIfNeeded(senderJID, senderPhone string) {
	if db == nil || senderJID == "" || senderPhone == "" {
		return
	}
	// Normalize to DB format (0xxx)
	phone := senderPhone
	if strings.HasPrefix(phone, "62") {
		phone = "0" + phone[2:]
	}
	db.Exec(`UPDATE users SET wa_jid = $1 WHERE phone_number = $2 AND (wa_jid IS NULL OR wa_jid = '')`,
		senderJID, phone)
}

func handleConnectedEvent(tenantID string) {
	slog.Info("Connected to WhatsApp", "tenant_id", tenantID)
	invalidatePlatformWAProviderCache()
	clientMu.RLock()
	c := clientMap[tenantID]
	clientMu.RUnlock()
	if c != nil && c.Store.ID != nil && db != nil {
		db.Exec(`INSERT INTO wa_tenant_sessions (tenant_id, jid) VALUES ($1, $2) ON CONFLICT (tenant_id) DO UPDATE SET jid = EXCLUDED.jid`, tenantID, c.Store.ID.String())
	}
}
