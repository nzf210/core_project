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

func toIntlPhone(p string) string {
	if strings.HasPrefix(p, "0") {
		return "62" + p[1:]
	}
	return p
}

func checkAuthPendingKeys(ctx context.Context, senderPhone, localPhone string) bool {
	intlPhone := toIntlPhone(senderPhone)
	for _, key := range []string{
		redisKeyAuthPending + senderPhone,
		redisKeyAuthPending + localPhone,
		redisKeyAuthPending + intlPhone,
	} {
		v, e := redisShared.Get(ctx, key).Result()
		if e == nil && v != "" {
			return true
		}
	}
	return false
}

func scanAuthPendingByValue(ctx context.Context, senderPhone, localPhone string) bool {
	intlPhone := toIntlPhone(senderPhone)
	keys, _ := redisShared.Keys(ctx, redisKeyAuthPending+"*").Result()
	for _, key := range keys {
		v, e := redisShared.Get(ctx, key).Result()
		if e == nil && (v == senderPhone || v == localPhone || v == intlPhone) {
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

	// Also check whatsmeow_lid_map if senderJID is @lid
	if registeredPhone == "" && strings.Contains(senderJID, "@lid") {
		lid := strings.Split(strings.TrimSuffix(senderJID, "@lid"), ":")[0]
		_ = db.QueryRow("SELECT pn FROM whatsmeow_lid_map WHERE lid = $1", lid).Scan(&registeredPhone)
	}

	if registeredPhone != "" {
		keyLocal := redisKeyAuthPending + toLocalPhone(registeredPhone)
		keyIntl := redisKeyAuthPending + toIntlPhone(registeredPhone)
		for _, key := range []string{keyLocal, keyIntl} {
			if v, e := redisShared.Get(ctx, key).Result(); e == nil && v != "" {
				slog.Info("handleWAOTPRequest: found via wa_jid/lid lookup",
					"senderJID", senderJID, "registeredPhone", registeredPhone)
				return registeredPhone, true
			}
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

	if regPhone, found := checkRegisteredPhoneByJID(ctx, senderJID); found {
		slog.Info("handleWAOTPRequest: found via registered phone by JID", "phone", regPhone)
		generateAndSendOTP(ctx, tenantID, senderJID, regPhone)
		return
	}

	slog.Warn("OTP request without pending login", "tenant_id", tenantID, "sender_phone", senderPhone, "local_phone", localPhone)

	sendWAMessage(tenantID, senderJID, "❌ Tidak ada permintaan login aktif. Silakan coba login di website terlebih dahulu.\n\nTips: Jika nomor WhatsApp berbeda dari yang diinput di website, Anda bisa ketik: *OTP [nomor_HP]* (contoh: OTP 081234567890)")
}

func handleWAOTPRequestWithPhone(tenantID, senderJID, senderPhone, phoneArg string) {
	ctx := context.Background()
	cleanPhone := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phoneArg)

	if cleanPhone == "" || len(cleanPhone) < 9 {
		sendWAMessage(tenantID, senderJID, "❌ Format nomor HP tidak valid. Contoh: OTP 081234567890")
		return
	}

	localPhone := toLocalPhone(cleanPhone)
	intlPhone := toIntlPhone(cleanPhone)

	slog.Info("handleWAOTPRequestWithPhone: checking OTP request with explicit phone",
		"tenant_id", tenantID,
		"sender_jid", senderJID,
		"phone_arg", phoneArg,
		"local_phone", localPhone,
		"intl_phone", intlPhone)

	found := false
	for _, key := range []string{
		redisKeyAuthPending + cleanPhone,
		redisKeyAuthPending + localPhone,
		redisKeyAuthPending + intlPhone,
	} {
		v, e := redisShared.Get(ctx, key).Result()
		if e == nil && v != "" {
			found = true
			break
		}
	}

	if !found {
		sendWAMessage(tenantID, senderJID, "❌ Tidak ada permintaan login untuk nomor "+cleanPhone+". Silakan klik 'Kirim OTP' di website terlebih dahulu.")
		return
	}

	// Save mapping to users table & whatsmeow_lid_map if possible
	mapUserJIDIfNeeded(senderJID, cleanPhone)
	if db != nil && strings.Contains(senderJID, "@lid") {
		lid := strings.Split(strings.TrimSuffix(senderJID, "@lid"), ":")[0]
		_, _ = db.Exec("INSERT INTO whatsmeow_lid_map (lid, pn) VALUES ($1, $2) ON CONFLICT (lid) DO UPDATE SET pn = EXCLUDED.pn", lid, intlPhone)
	}

	slog.Info("handleWAOTPRequestWithPhone: found and mapped", "phone", cleanPhone, "sender_jid", senderJID)
	generateAndSendOTP(ctx, tenantID, senderJID, cleanPhone)
}

func generateAndSendOTP(ctx context.Context, tenantID, senderJID, phone string) {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	otp := fmt.Sprintf("%06d", n.Int64())
	localPhone := toLocalPhone(phone)
	intlPhone := toIntlPhone(phone)

	redisShared.Set(ctx, "phone-login-otp:"+localPhone, otp+"|"+localPhone, 1*time.Hour)
	if intlPhone != localPhone {
		redisShared.Set(ctx, "phone-login-otp:"+intlPhone, otp+"|"+intlPhone, 1*time.Hour)
	}

	sendWAMessage(tenantID, senderJID, "📩 Kode OTP Anda: *"+otp+"*\n\nBalas pesan ini dengan 6 digit kode OTP tersebut.\n\nContoh: 123456")
	slog.Info("OTP generated & sent via WA Center", "phone", localPhone, "otp", otp)
}

func handleWALoginOTPReply(tenantID, senderJID, senderPhone, code string) {
	authSvcURL := getAuthServiceURL()
	phoneToVerify := senderPhone
	// If senderPhone looks like a LID, resolve real phone from DB
	if !strings.HasPrefix(phoneToVerify, "08") && !strings.HasPrefix(phoneToVerify, "628") {
		if db != nil {
			var pn string
			if strings.Contains(senderJID, "@lid") {
				lid := strings.Split(strings.TrimSuffix(senderJID, "@lid"), ":")[0]
				_ = db.QueryRow("SELECT pn FROM whatsmeow_lid_map WHERE lid = $1", lid).Scan(&pn)
			}
			if pn == "" {
				_ = db.QueryRow("SELECT phone_number FROM users WHERE wa_jid = $1", senderJID).Scan(&pn)
			}
			if pn != "" {
				phoneToVerify = pn
			}
		}
	}

	payload := map[string]interface{}{
		"phoneNumber": phoneToVerify,
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
