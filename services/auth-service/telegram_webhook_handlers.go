package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

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

// handleTelegramWebhookCommand dispatches Telegram commands with a multi-step
// password-reset state machine.
// Flow: /reset_password → lookup by chat_id → confirm (ya/tidak) → new password → done.
func handleTelegramWebhookCommand(chatID, text string) {
	text = strings.TrimSpace(text)
	if idx := strings.Index(text, "@"); idx != -1 {
		text = text[:idx]
	}

	ctx := context.Background()
	stepKey := "pw-reset:step:" + chatID
	currentStep, _ := Redis.Get(ctx, stepKey).Result()

	if handleTelegramCommandRouter(ctx, chatID, text, stepKey) {
		return
	}

	handleTelegramFlowStep(ctx, chatID, text, stepKey, currentStep)
}

func handleTelegramCommandRouter(ctx context.Context, chatID, text, stepKey string) bool {
	switch text {
	case "/start":
		return handleTelegramStart(ctx, chatID, stepKey)
	case "/reset_password", "/resetpassword":
		return handleTelegramResetPasswordInit(ctx, chatID, stepKey)
	default:
		return false
	}
}

func handleTelegramStart(ctx context.Context, chatID, stepKey string) bool {
	Redis.Del(ctx, stepKey, "pw-reset:data:"+chatID)
	sendTelegramMessage(chatID,
		"👋 *Selamat datang di WCH Platform!*\n\n"+
			"✅ Bot berhasil terhubung!\n"+
			"Chat ID Anda: `"+chatID+"`\n\n"+
			"Buka aplikasi WCH dan pilih *Masuk dengan Telegram* untuk mendaftar atau login.\n\n"+
			"Bot ini akan mengirimkan notifikasi penting seperti:\n"+
			"• Update langganan & tagihan\n"+
			"• Kode OTP untuk verifikasi\n"+
			"• Pengingat automate\n\n"+
			"Hubungi admin jika butuh bantuan.",
	)
	return true
}

func handleTelegramResetPasswordInit(ctx context.Context, chatID, stepKey string) bool {
	var userID, phone string
	err := DB.QueryRow(ctx,
		"SELECT id, phone_number FROM users WHERE telegram_chat_id = $1",
		chatID,
	).Scan(&userID, &phone)

	if err == pgx.ErrNoRows {
		sendTelegramMessage(chatID,
			"❌ Akun Telegram ini belum terhubung ke akun WCH manapun.\n\n"+
				"Buka aplikasi WCH, lalu gunakan menu *Masuk dengan Telegram* untuk menghubungkan akun Anda.\n\n"+
				"Hubungi admin jika butuh bantuan.",
		)
		return true
	}

	if err != nil {
		slog.Error("DB error in telegram reset step 0", "error", err)
		sendTelegramMessage(chatID, "⚠️ Terjadi kesalahan. Silakan coba lagi.")
		return true
	}

	masked := phone[:4] + "xxx" + phone[len(phone)-4:]
	Redis.Set(ctx, stepKey, "await_confirm", 0)
	Redis.Set(ctx, "pw-reset:data:"+chatID, userID+"|"+phone, 10*time.Minute)

	sendTelegramMessage(chatID,
		"🔑 *Reset Password*\n\n"+
			"Apakah Anda yakin ingin mereset password untuk akun:\n"+
			"`"+masked+"`\n\n"+
			"Ketik *ya* untuk melanjutkan atau *tidak* untuk membatalkan.",
	)
	return true
}

func handleTelegramFlowStep(ctx context.Context, chatID, text, stepKey, currentStep string) {
	switch currentStep {
	case "await_confirm":
		handleTelegramAwaitConfirm(ctx, chatID, text, stepKey)
	case "await_password":
		handleTelegramAwaitPassword(ctx, chatID, text, stepKey)
	default:
		sendTelegramMessage(chatID,
			"✅ Bot aktif!\n\n"+
				"Chat ID Anda: `"+chatID+"`\n\n"+
				"Kirim `/start` untuk melihat panduan.\n"+
				"Gunakan `/reset_password` jika lupa password.",
		)
	}
}

func handleTelegramAwaitConfirm(ctx context.Context, chatID, text, stepKey string) {
	lowerText := strings.ToLower(text)
	if lowerText == "tidak" || lowerText == "batal" {
		Redis.Del(ctx, stepKey, "pw-reset:data:"+chatID)
		sendTelegramMessage(chatID, "✅ Reset password dibatalkan.")
		return
	}

	if lowerText != "ya" {
		sendTelegramMessage(chatID, "❌ Ketik *ya* untuk melanjutkan atau *tidak* untuk membatalkan.")
		return
	}

	Redis.Set(ctx, stepKey, "await_password", 0)
	sendTelegramMessage(chatID,
		"✅ Baik!\n\n"+
			"Kirim password baru Anda.\n"+
			"📌 Minimal 8 karakter.\n\n"+
			"Ketik *tidak* untuk membatalkan.",
	)
}

func handleTelegramAwaitPassword(ctx context.Context, chatID, text, stepKey string) {
	lowerText := strings.ToLower(text)
	if lowerText == "tidak" || lowerText == "batal" {
		Redis.Del(ctx, stepKey, "pw-reset:data:"+chatID)
		sendTelegramMessage(chatID, "✅ Reset password dibatalkan.")
		return
	}

	if len(text) < 8 {
		sendTelegramMessage(chatID, "❌ Password minimal 8 karakter. Silakan coba lagi.")
		return
	}

	data, _ := Redis.Get(ctx, "pw-reset:data:"+chatID).Result()
	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		Redis.Del(ctx, stepKey)
		sendTelegramMessage(chatID, "⚠️ Sesi expired. Ketik /reset_password untuk mulai ulang.")
		return
	}

	userID, phone := parts[0], parts[1]
	hashed, err := bcrypt.GenerateFromPassword([]byte(text), 12)
	if err != nil {
		sendTelegramMessage(chatID, "⚠️ Terjadi kesalahan. Ketik /reset_password untuk mulai ulang.")
		Redis.Del(ctx, stepKey, "pw-reset:data:"+chatID)
		return
	}

	_, err = DB.Exec(ctx,
		"UPDATE users SET password_hash = $1, must_change_password = false, updated_at = NOW() WHERE id = $2",
		string(hashed), userID,
	)
	if err != nil {
		slog.Error("Telegram password reset: failed to update", "user_id", userID, "error", err)
		sendTelegramMessage(chatID, "⚠️ Terjadi kesalahan. Ketik /reset_password untuk mulai ulang.")
		Redis.Del(ctx, stepKey, "pw-reset:data:"+chatID)
		return
	}

	Redis.Del(ctx, stepKey, "pw-reset:data:"+chatID)
	slog.Info("Password reset via Telegram complete", "user_id", userID, "phone", phone)
	sendTelegramMessage(chatID,
		"✅ *Password berhasil direset!*\n\n"+
			"Silakan login dengan password baru Anda di aplikasi WCH.",
	)
}
