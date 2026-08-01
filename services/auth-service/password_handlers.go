package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// ─────────────────────────────────────────────
// Chat-based Password Reset (WA + Telegram)
// F055 v2: User chat WA/Telegram → OTP → reset
// ─────────────────────────────────────────────

// ResetPasswordChatRequest is the payload for requesting a password reset OTP
// via WhatsApp or Telegram chat.
type ResetPasswordChatRequest struct {
	PhoneNumber string `json:"phoneNumber"` // required: registered phone number
	Channel     string `json:"channel"`    // "wa" or "telegram"
}

// ResetPasswordVerifyRequest is the payload for verifying the OTP and setting
// a new password.
type ResetPasswordVerifyRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	OTP         string `json:"otp"`
	NewPassword string `json:"newPassword"`
	Channel     string `json:"channel"` // "wa" or "telegram"
}

func checkExistingPasswordResetOTP(ctx context.Context, phone, channel string) (bool, string, error) {
	otpKey := redisKeyPWResetOTP + phone
	existingVal, err := Redis.Get(ctx, otpKey).Result()
	if err == nil && existingVal != "" {
		ttl, _ := Redis.TTL(ctx, otpKey).Result()
		slog.Info("Password reset OTP rate-limited", "phone", phone, "ttl_remaining_sec", int(ttl.Seconds()))
		msg := fmt.Sprintf("Kode OTP sudah dikirim sebelumnya. Berlaku selama %d menit. Silakan cek %s Anda.",
			int(ttl.Minutes())+1, mapChannelName(channel))
		return true, msg, nil
	}
	return false, "", nil
}

func generateAndStorePasswordResetOTP(ctx context.Context, phone string) (string, error) {
	otpNum, err := rand.Int(rand.Reader, new(big.Int).SetUint64(1000000))
	if err != nil {
		return "", err
	}
	otp := fmt.Sprintf("%06d", otpNum.Int64())

	otpKey := redisKeyPWResetOTP + phone
	if err := Redis.Set(ctx, otpKey, otp, 1*time.Hour).Err(); err != nil {
		slog.Error("Failed to store password reset OTP in Redis", "error", err)
		return "", err
	}
	return otp, nil
}

func sendPasswordResetOTPViaWA(phone, otp string) {
	go func() {
		waTarget := phone + "@s.whatsapp.net"
		if err := sendWAPasswordResetOTP("system", waTarget, otp); err != nil {
			slog.Error("Failed to send password reset OTP via WA", "phone", phone, "error", err)
		} else {
			slog.Info("Password reset OTP sent via WA", "phone", phone)
		}
	}()
}

func sendPasswordResetOTPViaTelegram(telegramChatID, otp string) {
	go func() {
		msg := fmt.Sprintf("🔑 *Kode Reset Password WCH*\n\nKode OTP Anda: *%s*\n\n📌 Masukkan kode ini untuk mereset password.\n\n⚠️ Jangan bagikan kode ini kepada siapapun.\n\nBerlaku selama 1 jam.", otp)
		if err := sendTelegramMessage(telegramChatID, msg); err != nil {
			slog.Error("Failed to send password reset OTP via Telegram", "chatID", telegramChatID, "error", err)
		} else {
			slog.Info("Password reset OTP sent via Telegram", "chatID", telegramChatID)
		}
	}()
}

func handleRequestResetPasswordOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req ResetPasswordChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	if req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "phoneNumber wajib diisi"})
		return
	}

	if req.Channel == "" {
		req.Channel = "wa"
	}
	if req.Channel != "wa" && req.Channel != "telegram" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "channel harus 'wa' atau 'telegram'"})
		return
	}

	phone := normalizePhone(req.PhoneNumber)
	ctx := context.Background()

	var userID, telegramChatID string
	err := DB.QueryRow(ctx,
		"SELECT id, COALESCE(telegram_chat_id, '') FROM users WHERE phone_number = $1 OR phone_number = $2",
		phone, "0"+phone[2:],
	).Scan(&userID, &telegramChatID)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Jika nomor terdaftar, kode OTP akan dikirim."})
		return
	}
	if err != nil {
		slog.Error("DB error looking up user for password reset", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	if exists, msg, _ := checkExistingPasswordResetOTP(ctx, phone, req.Channel); exists {
		writeJSON(w, http.StatusOK, Response{Success: true, Message: msg})
		return
	}

	otp, err := generateAndStorePasswordResetOTP(ctx, phone)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	channelName := mapChannelName(req.Channel)
	if req.Channel == "wa" {
		sendPasswordResetOTPViaWA(phone, otp)
	} else {
		if telegramChatID == "" {
			Redis.Del(ctx, redisKeyPWResetOTP+phone)
			writeJSON(w, http.StatusBadRequest, Response{
				Success: false,
				Message: "Akun Anda belum terhubung dengan Telegram. Gunakan chat WhatsApp untuk reset password.",
			})
			return
		}
		sendPasswordResetOTPViaTelegram(telegramChatID, otp)
	}

	slog.Info("Password reset OTP generated", "phone", phone, "channel", req.Channel)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: fmt.Sprintf("Kode OTP telah dikirim ke %s Anda. Berlaku selama 1 jam.", channelName),
	})
}

// handleVerifyResetPasswordOTP verifies the OTP and sets the new password.
func handleVerifyResetPasswordOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	var req ResetPasswordVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: msgInvalidPayload})
		return
	}

	if req.PhoneNumber == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "phoneNumber dan newPassword wajib diisi"})
		return
	}
	if req.OTP == "" && r.Header.Get("X-OTP-Verified") != "true" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "otp wajib diisi"})
		return
	}
	if len(req.NewPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Password baru minimal 8 karakter"})
		return
	}

	phone := normalizePhone(req.PhoneNumber)
	ctx := context.Background()

	var otpKey string
	// Verify OTP only if not bypassed (X-OTP-Verified set by internal services like wa-gateway)
	if r.Header.Get("X-OTP-Verified") != "true" {
		otpKey = "pw-reset-otp:" + phone
		storedOTP, err := Redis.Get(ctx, otpKey).Result()
		if err != nil || storedOTP != req.OTP {
			writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Kode OTP tidak valid atau sudah kadaluarsa."})
			return
		}
	}

	// Find user
	var userID string
	err := DB.QueryRow(ctx,
		"SELECT id FROM users WHERE phone_number = $1 OR phone_number = $2",
		phone, "0"+phone[2:],
	).Scan(&userID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Nomor tidak terdaftar."})
		return
	}

	// Hash and set new password (clear must_change_password if present)
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		slog.Error("Failed to hash new password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	_, err = DB.Exec(ctx,
		"UPDATE users SET password_hash = $1, must_change_password = false, updated_at = NOW() WHERE id = $2",
		string(hashed), userID,
	)
	if err != nil {
		slog.Error("Failed to update password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	// Consume OTP
	if otpKey != "" {
		Redis.Del(ctx, otpKey)
	}

	slog.Info("Password reset successful via chat", "user_id", userID, "phone", phone)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Password berhasil direset. Silakan login dengan password baru Anda.",
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

// sendWAPasswordResetOTP sends a password reset OTP via the wa-gateway.
func sendWAPasswordResetOTP(senderTenant, target, otp string) error {
	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = "http://wa-gateway:8202"
	}

	msg := fmt.Sprintf("🔑 *Kode Reset Password WCH*\n\nKode OTP Anda: *%s*\n\n📌 Masukkan kode ini untuk mereset password.\n\n⚠️ Jangan bagikan kode ini kepada siapapun.\n\nBerlaku selama 1 jam.", otp)

	formData := url.Values{}
	formData.Set("tenant_id", senderTenant)
	formData.Set("target", target)
	formData.Set("message", msg)

	payload := strings.NewReader(formData.Encode())
	req, err := http.NewRequest("POST", waURL+"/api/wa/send", payload)
	if err != nil {
		return err
	}
	req.Header.Set(headerContentType, contentTypeFormURLEncoded)
	req.Header.Set("X-Message-Type", "otp")
	req.Header.Set("X-Source", "auth-service")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wa-gateway returned %d", resp.StatusCode)
	}
	return nil
}

// normalizePhone converts a phone number to the "62" format.
func normalizePhone(phone string) string {
	p := strings.TrimSpace(phone)
	if strings.HasPrefix(p, "0") {
		p = "62" + p[1:]
	} else if strings.HasPrefix(p, "+") {
		p = p[1:]
	}
	return p
}

// mapChannelName returns a human-readable channel name.
func mapChannelName(channel string) string {
	if channel == "telegram" {
		return "Telegram"
	}
	return "WhatsApp"
}
