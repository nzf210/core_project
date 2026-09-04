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

func toLocalPhone(p string) string {
	if strings.HasPrefix(p, "62") {
		return "0" + p[2:]
	}
	return p
}

func checkAuthPendingKeys(ctx context.Context, senderPhone, localPhone string) bool {
	for _, key := range []string{
		redisKeyAuthPending + senderPhone,
		redisKeyAuthPending + localPhone,
	} {
		v, e := redisShared.Get(ctx, key).Result()
		if e == nil && v != "" {
			return true
		}
	}
	return false
}

func scanAuthPendingByValue(ctx context.Context, senderPhone, localPhone string) bool {
	keys, _ := redisShared.Keys(ctx, redisKeyAuthPending+"*").Result()
	for _, key := range keys {
		v, e := redisShared.Get(ctx, key).Result()
		if e == nil && (v == senderPhone || v == localPhone) {
			slog.Info("handleWAOTPRequest: found via value scan",
				"senderPhone", senderPhone, "matchedKey", key)
			return true
		}
	}
	return false
}

func checkRegisteredPhoneByJID(ctx context.Context, senderJID string) (string, bool) {
	if db == nil {
		return "", false
	}

	var registeredPhone string
	err := db.QueryRow("SELECT phone_number FROM users WHERE wa_jid = $1", senderJID).Scan(&registeredPhone)
	if err != nil {
		shortJID := strings.Split(senderJID, ":")[0] + "@s.whatsapp.net"
		err = db.QueryRow("SELECT phone_number FROM users WHERE wa_jid = $1", shortJID).Scan(&registeredPhone)
	}

	if registeredPhone != "" {
		key := redisKeyAuthPending + toLocalPhone(registeredPhone)
		if v, e := redisShared.Get(ctx, key).Result(); e == nil && v != "" {
			slog.Info("handleWAOTPRequest: found via wa_jid lookup",
				"senderJID", senderJID, "registeredPhone", registeredPhone)
			return registeredPhone, true
		}
	}
	return "", false
}

func handleWAOTPRequest(tenantID, senderJID, senderPhone string) {
	ctx := context.Background()
	localPhone := toLocalPhone(senderPhone)

	slog.Info("handleWAOTPRequest: checking OTP request",
		"tenant_id", tenantID,
		"sender_jid", senderJID,
		"sender_phone", senderPhone,
		"local_phone", localPhone)

	if checkAuthPendingKeys(ctx, senderPhone, localPhone) {
		slog.Info("handleWAOTPRequest: found via auth pending keys")
		generateAndSendOTP(ctx, tenantID, senderJID, senderPhone)
		return
	}

	if scanAuthPendingByValue(ctx, senderPhone, localPhone) {
		slog.Info("handleWAOTPRequest: found via value scan")
		generateAndSendOTP(ctx, tenantID, senderJID, senderPhone)
		return
	}

	if _, found := checkRegisteredPhoneByJID(ctx, senderJID); found {
		slog.Info("handleWAOTPRequest: found via registered phone by JID")
		generateAndSendOTP(ctx, tenantID, senderJID, senderPhone)
		return
	}

	slog.Warn("OTP request without pending login", "tenant_id", tenantID, "sender_phone", senderPhone, "local_phone", localPhone)

	// Debug: list all pending keys in Redis
	keys, _ := redisShared.Keys(ctx, "auth:pending:*").Result()
	slog.Info("handleWAOTPRequest: available auth:pending keys", "keys", keys)

	// Debug: check JID mapping
	if db != nil {
		var phone string
		err := db.QueryRow("SELECT phone_number FROM users WHERE wa_jid = $1", senderJID).Scan(&phone)
		if err == nil {
			slog.Info("handleWAOTPRequest: found user by JID", "phone", phone)
		} else {
			// Try short JID
			shortJID := strings.Split(senderJID, ":")[0] + "@s.whatsapp.net"
			err = db.QueryRow("SELECT phone_number FROM users WHERE wa_jid = $1", shortJID).Scan(&phone)
			if err == nil {
				slog.Info("handleWAOTPRequest: found user by short JID", "phone", phone)
			}
		}
	}

	sendWAMessage(tenantID, senderJID, "❌ Tidak ada permintaan login. Silakan coba login di website terlebih dahulu.")
}

func generateAndSendOTP(ctx context.Context, tenantID, senderJID, senderPhone string) {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	otp := fmt.Sprintf("%06d", n.Int64())
	normPhone := senderPhone
	if strings.HasPrefix(normPhone, "62") {
		normPhone = "0" + normPhone[2:]
	}
	otpKey := "phone-login-otp:" + normPhone
	redisShared.Set(ctx, otpKey, otp+"|"+normPhone, 1*time.Hour)
	sendWAMessage(tenantID, senderJID, "📩 Kode OTP Anda: *"+otp+"*\n\nBalas pesan ini dengan 6 digit kode OTP tersebut.\n\nContoh: 123456")
	slog.Info("OTP generated & sent via WA Center", "phone", normPhone, "otp", otp)
}

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
