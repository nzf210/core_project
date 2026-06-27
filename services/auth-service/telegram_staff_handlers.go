package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"core_project/shared/sdk/config"

	"github.com/jackc/pgx/v5"
)

func handleTelegramRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req TelegramRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	slog.Info("[TELEGRAM:REGISTER] Incoming request", "chatID", req.TelegramChatID, "phone", req.PhoneNumber, "username", req.Username, "businessName", req.BusinessName)

	if req.PhoneNumber == "" || req.Password == "" || req.TelegramChatID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "telegramChatId, phoneNumber, and password are required"})
		return
	}

	ctx := context.Background()
	otpKey := "otp:" + req.PhoneNumber

	if existingVal, err := Redis.Get(ctx, otpKey).Result(); err == nil && existingVal != "" {
		parts := strings.Split(existingVal, ":")
		existingOTP := parts[len(parts)-1]
		ttl, _ := Redis.TTL(ctx, otpKey).Result()
		slog.Info("OTP still active, reusing existing (Telegram)", "phone", req.PhoneNumber, "otp", existingOTP, "ttl_remaining_sec", int(ttl.Seconds()))
		writeJSON(w, http.StatusOK, Response{
			Success: true,
			Message: "OTP already sent. Valid for 1 hour. Please check your Telegram.",
		})
		return
	}

	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)

	regData := map[string]any{
		"username":       req.Username,
		"password":       req.Password,
		"email":          req.Email,
		"phoneNumber":    req.PhoneNumber,
		"role":           req.Role,
		"tenantId":       req.TenantID,
		"businessName":   req.BusinessName,
		"telegramChatId": req.TelegramChatID,
	}
	regJSON, _ := json.Marshal(regData)
	err := Redis.Set(ctx, otpKey, string(regJSON)+":"+otp, 1*time.Hour).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process registration"})
		return
	}

	go func() {
		msg := fmt.Sprintf("🔐 *Kode OTP Registrasi WCH*\n\nKode OTP Anda: *%s*\n\n📌 Masukkan kode ini di aplikasi WCH untuk menyelesaikan pendaftaran.\n\n⚠️ Jangan bagikan kode ini kepada siapapun.\n\nBerlaku selama 1 jam.", otp)
		if err := sendTelegramOTP(req.TelegramChatID, msg); err != nil {
			slog.Error("Failed to send register OTP via Telegram", "chatID", req.TelegramChatID, "error", err)
		} else {
			slog.Info("Telegram register OTP sent successfully", "chatID", req.TelegramChatID)
		}
	}()

	slog.Info("Telegram register OTP generated", "phone", req.PhoneNumber, "chatID", req.TelegramChatID, "otp", otp)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "OTP has been sent to your Telegram. Please verify.",
	})
}

func handleTelegramLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req TelegramLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.PhoneNumber == "" || req.TelegramChatID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "telegramChatId and phoneNumber are required"})
		return
	}

	ctx := context.Background()

	var userID string
	err := DB.QueryRow(ctx, "SELECT id FROM users WHERE phone_number = $1", req.PhoneNumber).Scan(&userID)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Phone number not registered"})
		return
	} else if err != nil {
		slog.Error("DB query failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	otp := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	otpKey := "phone-login-otp:" + req.PhoneNumber
	err = Redis.Set(ctx, otpKey, otp, 1*time.Hour).Err()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to process login"})
		return
	}

	_, err = DB.Exec(ctx, "UPDATE users SET telegram_chat_id = $1 WHERE phone_number = $2", req.TelegramChatID, req.PhoneNumber)
	if err != nil {
		slog.Warn("Failed to update telegram_chat_id", "phone", req.PhoneNumber, "error", err)
	}

	go func() {
		msg := fmt.Sprintf("🔐 *Kode OTP Login WCH*\n\nKode OTP Anda: *%s*\n\n📌 Masukkan kode ini di aplikasi WCH untuk masuk ke akun Anda.\n\n⚠️ Jangan bagikan kode ini kepada siapapun.\n\nBerlaku selama 1 jam.", otp)
		if err := sendTelegramOTP(req.TelegramChatID, msg); err != nil {
			slog.Error("Failed to send login OTP via Telegram", "chatID", req.TelegramChatID, "error", err)
		} else {
			slog.Info("Telegram login OTP sent successfully", "chatID", req.TelegramChatID)
		}
	}()

	slog.Info("Telegram login OTP sent", "phone", req.PhoneNumber, "chatID", req.TelegramChatID, "otp", otp)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "OTP telah dikirim ke Telegram Anda. Silakan verifikasi.",
	})
}

func isTelegramWebhookSet(client *http.Client, baseURL string) bool {
	resp, err := client.Get(baseURL + "/getWebhookInfo")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			URL string `json:"url"`
		} `json:"result"`
	}
	if json.NewDecoder(resp.Body).Decode(&result) == nil && result.Result.URL != "" {
		return true
	}
	return false
}

func processTelegramUpdates(updates []map[string]any, nextUpdateID *int64) {
	for _, update := range updates {
		updateIDFloat, ok := update["update_id"].(float64)
		if ok {
			*nextUpdateID = int64(updateIDFloat) + 1
		}

		message, ok := update["message"].(map[string]any)
		if !ok {
			continue
		}
		chat, ok := message["chat"].(map[string]any)
		if !ok {
			continue
		}
		chatIDFloat, ok := chat["id"].(float64)
		if !ok {
			continue
		}
		chatID := fmt.Sprintf("%.0f", chatIDFloat)
		text, _ := message["text"].(string)

		handleTelegramCommand(chatID, text)
	}
}

func handleTelegramCommand(chatID, text string) {
	if text == "/start" {
		welcomeMsg := fmt.Sprintf(
			"👋 *Selamat datang di WCH Platform!*\n\n"+
				"✅ Bot berhasil terhubung!\n"+
				"Chat ID Anda: `%s`\n\n"+
				"Buka aplikasi WCH dan pilih *Masuk dengan Telegram* untuk mendaftar atau login.\n\n"+
				"Bot ini akan mengirimkan notifikasi penting seperti:\n"+
				"• Update langganan & tagihan\n"+
				"• Kode OTP untuk verifikasi\n"+
				"• Pengingat automate\n\n"+
				"Hubungi admin jika butuh bantuan.",
			chatID,
		)
		sendTelegramMessage(chatID, welcomeMsg)
	} else {
		sendTelegramMessage(chatID, fmt.Sprintf("✅ Bot aktif!\n\nChat ID Anda: `%s`\n\nKirim `/start` untuk melihat panduan.", chatID))
	}
}

func startTelegramPolling(cfg *config.Config) {
	botToken := cfg.Telegram.BotToken
	if botToken == "" {
		slog.Warn("TELEGRAM_BOT_TOKEN not set, skipping Telegram polling")
		return
	}

	baseURL := fmt.Sprintf("https://api.telegram.org/bot%s", botToken)
	client := &http.Client{Timeout: 10 * time.Second}
	nextUpdateID := int64(0)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	slog.Info("Telegram polling started", "bot", "Core_tesbot")

	for range ticker.C {
		if isTelegramWebhookSet(client, baseURL) {
			slog.Info("Telegram webhook is set, stopping polling")
			return
		}
		pollTelegramOnce(client, baseURL, &nextUpdateID)
	}
}

func pollTelegramOnce(client *http.Client, baseURL string, nextUpdateID *int64) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d", baseURL, *nextUpdateID)
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var updates struct {
		OK      bool             `json:"ok"`
		Result  []map[string]any `json:"result"`
		ErrCode int              `json:"error_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
		return
	}

	if len(updates.Result) > 0 {
		slog.Info("Telegram updates received", "count", len(updates.Result), "next_offset", *nextUpdateID)
	} else {
		slog.Info("Telegram poll cycle", "offset", *nextUpdateID, "updates", 0)
	}

	processTelegramUpdates(updates.Result, nextUpdateID)
}

func handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var update map[string]any
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid update payload"})
		return
	}

	message, ok := update["message"].(map[string]any)
	if !ok {
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "OK"})
		return
	}

	chat, ok := message["chat"].(map[string]any)
	if !ok {
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "OK"})
		return
	}

	chatIDFloat, ok := chat["id"].(float64)
	if !ok {
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "OK"})
		return
	}
	chatID := fmt.Sprintf("%.0f", chatIDFloat)

	text, _ := message["text"].(string)
	handleTelegramWebhookCommand(chatID, text)

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "OK"})
}

func handleTelegramWebhookCommand(chatID, text string) {
	if text == "/start" {
		welcomeMsg := fmt.Sprintf(
			"👋 *Selamat datang di WCH Platform!*\n\n"+
				"✅ Bot berhasil terhubung!\n"+
				"Chat ID Anda: `%s`\n\n"+
				"Buka aplikasi WCH dan pilih *Masuk dengan Telegram* untuk mendaftar atau login.\n\n"+
				"Bot ini akan mengirimkan notifikasi penting seperti:\n"+
				"• Update langganan & tagihan\n"+
				"• Kode OTP untuk verifikasi\n"+
				"• Pengingat automate\n\n"+
				"Hubungi admin jika butuh bantuan.",
			chatID,
		)
		sendTelegramOTP(chatID, welcomeMsg)
	} else {
		sendTelegramOTP(chatID, fmt.Sprintf("✅ Bot aktif!\n\nChat ID Anda: `%s`\n\nKirim `/start` untuk melihat panduan.", chatID))
	}
}

func formatPhoneToWAJID(phone string) string {
	if strings.HasSuffix(phone, "@s.whatsapp.net") {
		return phone
	}
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	} else if strings.HasPrefix(phone, "+") {
		phone = phone[1:]
	}
	return phone + "@s.whatsapp.net"
}
