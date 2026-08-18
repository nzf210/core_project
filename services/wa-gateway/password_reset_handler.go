package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"
)

type waPasswordResetSession struct {
	SenderJID   string
	PhoneNumber string
	Step        int
	CreatedAt   time.Time
}

const waPWResetSessionTTL = 30 * time.Minute

func pwResetSessionKey(senderJID string) string {
	return "wa:pw-reset-session:" + senderJID
}

func savePWResetSession(session *waPasswordResetSession) {
	if redisShared == nil {
		waPasswordResetSessions[session.SenderJID] = session
		return
	}
	data, err := json.Marshal(session)
	if err != nil {
		waPasswordResetSessions[session.SenderJID] = session
		return
	}
	ctx := context.Background()
	if err := redisShared.Set(ctx, pwResetSessionKey(session.SenderJID), data, waPWResetSessionTTL).Err(); err != nil {
		waPasswordResetSessions[session.SenderJID] = session
	}
}

func loadPWResetSession(senderJID string) (*waPasswordResetSession, bool) {
	if redisShared != nil {
		ctx := context.Background()
		data, err := redisShared.Get(ctx, pwResetSessionKey(senderJID)).Bytes()
		if err == nil {
			var session waPasswordResetSession
			if json.Unmarshal(data, &session) == nil {
				return &session, true
			}
		}
	}
	s, ok := waPasswordResetSessions[senderJID]
	return s, ok
}

func deletePWResetSession(senderJID string) {
	if redisShared != nil {
		ctx := context.Background()
		redisShared.Del(ctx, pwResetSessionKey(senderJID))
	}
	delete(waPasswordResetSessions, senderJID)
}

// in-memory fallback map (used when Redis is unavailable)
var waPasswordResetSessions = make(map[string]*waPasswordResetSession)

func startWAPasswordReset(tenantID, senderJID, senderPhone string) {
	savePWResetSession(&waPasswordResetSession{
		SenderJID:   senderJID,
		PhoneNumber: senderPhone,
		Step:        0,
		CreatedAt:   time.Now(),
	})
	sendWAMessage(tenantID, senderJID,
		"🔑 *Reset Password*\n\n"+
			"Kirim nomor HP Anda yang terdaftar di platform WCH.\n"+
			"Contoh: 0812xxxxxxxx\n\n"+
			"Ketik *batal* untuk membatalkan.")
}

func cancelPasswordReset(tenantID, senderJID string) {
	deletePWResetSession(senderJID)
	sendWAMessage(tenantID, senderJID, msgResetPasswordCanceled)
}

func handlePhoneInputStep(tenantID string, session *waPasswordResetSession, rawText string) bool {
	phone := normalizePhone(rawText)
	if len(phone) < 10 || !strings.HasPrefix(phone, "62") {
		sendWAMessage(tenantID, session.SenderJID, "❌ Format nomor HP tidak valid. Contoh: 0812xxxxxxxx")
		return true
	}

	if db != nil {
		var userID string
		phoneAlt := "0" + phone[2:]
		err := db.QueryRow("SELECT id FROM users WHERE phone_number = $1 OR phone_number = $2", phone, phoneAlt).Scan(&userID)
		if err != nil {
			sendWAMessage(tenantID, session.SenderJID,
				"📱 Nomor tidak ditemukan di sistem kami.\n\n"+
					"Ketik nomor HP lain, atau ketik *batal* untuk membatalkan.")
			return true
		}
		session.PhoneNumber = phone
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	otp := fmt.Sprintf("%06d", n.Int64())
	ctx := context.Background()
	otpKey := redisKeyPWResetOTP + session.PhoneNumber
	redisShared.Set(ctx, otpKey, otp, 1*time.Hour)

	session.Step = 1
	session.CreatedAt = time.Now()
	savePWResetSession(session)

	sendWAMessage(tenantID, session.SenderJID,
		"📩 Kode OTP telah dikirim ke chat ini.\n\n"+
			"Silakan balas dengan 6 digit kode OTP tersebut.\n\n"+
			msgBatalCommand)

	sendWAMessage(tenantID, session.SenderJID,
		"🔑 *Kode Reset Password WCH*\n\n"+
			"Kode OTP Anda: *"+otp+"*\n\n"+
			"📌 Masukkan kode ini untuk mereset password.\n"+
			"⚠️ Jangan bagikan kode ini kepada siapapun.\n"+
			"Berlaku selama 1 jam.")
	slog.Info("WA password reset OTP generated", "phone", session.PhoneNumber, "otp", otp)
	return true
}

func handleOTPVerificationStep(tenantID string, session *waPasswordResetSession, upperText string) bool {
	if !isSixDigitOTP(strings.TrimSpace(upperText)) {
		sendWAMessage(tenantID, session.SenderJID, "❌ Kode OTP harus 6 digit angka. Coba lagi.")
		return true
	}
	otpKey := redisKeyPWResetOTP + session.PhoneNumber
	ctx := context.Background()
	storedOTP, _ := redisShared.Get(ctx, otpKey).Result()
	if storedOTP != strings.TrimSpace(upperText) {
		sendWAMessage(tenantID, session.SenderJID, "❌ Kode OTP salah. Silakan coba lagi.")
		return true
	}

	session.Step = 2
	session.CreatedAt = time.Now()
	savePWResetSession(session)
	sendWAMessage(tenantID, session.SenderJID,
		"✅ Kode OTP verified!\n\n"+
			"Sekarang kirim password baru Anda (minimal 8 karakter).\n\n"+
			msgBatalCommand)
	return true
}

func handleNewPasswordStep(tenantID string, session *waPasswordResetSession, rawText string) bool {
	if len(rawText) < 8 {
		sendWAMessage(tenantID, session.SenderJID, "❌ Password minimal 8 karakter. Silakan coba lagi.")
		return true
	}

	authSvcURL := getAuthServiceURL()
	body, _ := json.Marshal(map[string]any{
		"phoneNumber": session.PhoneNumber,
		"newPassword": rawText,
	})
	req, _ := http.NewRequestWithContext(context.Background(), "POST", authSvcURL+"/reset-password-verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("X-OTP-Verified", "true")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp == nil {
		sendWAMessage(tenantID, session.SenderJID, "❌ Gagal mereset password. Silakan coba lagi.")
		deletePWResetSession(session.SenderJID)
		return true
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	deletePWResetSession(session.SenderJID)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		ctx := context.Background()
		otpKey := redisKeyPWResetOTP + session.PhoneNumber
		redisShared.Del(ctx, otpKey)

		sendWAMessage(tenantID, session.SenderJID,
			"✅ *Password berhasil direset!*\n\n"+
				"Silakan login dengan password baru Anda di aplikasi WCH.")
		slog.Info("WA password reset complete", "phone", session.PhoneNumber)
	} else {
		msg := "Terjadi kesalahan."
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		sendWAMessage(tenantID, session.SenderJID, "❌ Gagal: "+msg+"\n\nKetik RESET untuk mulai ulang.")
	}
	return true
}

func handleWAPasswordResetStep(tenantID string, session *waPasswordResetSession, rawText, upperText string) bool {
	if upperText == "BATAL" {
		cancelPasswordReset(tenantID, session.SenderJID)
		return true
	}

	switch session.Step {
	case 0:
		return handlePhoneInputStep(tenantID, session, rawText)
	case 1:
		return handleOTPVerificationStep(tenantID, session, upperText)
	case 2:
		return handleNewPasswordStep(tenantID, session, rawText)
	}
	return false
}

func normalizePhone(phone string) string {
	p := strings.TrimSpace(phone)
	p = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, p)
	if strings.HasPrefix(p, "0") {
		p = "62" + p[1:]
	} else if strings.HasPrefix(p, "+") {
		p = p[1:]
	}
	return p
}
