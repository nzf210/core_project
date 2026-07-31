package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func handleWAOTPRequest(tenantID, senderJID, senderPhone string) {
	ctx := context.Background()

	toLocal := func(p string) string {
		if strings.HasPrefix(p, "62") {
			return "0" + p[2:]
		}
		return p
	}

	localPhone := toLocal(senderPhone)

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
