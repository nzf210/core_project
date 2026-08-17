package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

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

const waRegSessionTTL = 30 * time.Minute

func regSessionKey(senderJID string) string {
	return "wa:reg-session:" + senderJID
}

func saveRegSession(session *waRegistrationSession) {
	if redisShared == nil {
		waRegistrationSessions[session.SenderJID] = session
		return
	}
	data, err := json.Marshal(session)
	if err != nil {
		waRegistrationSessions[session.SenderJID] = session
		return
	}
	ctx := context.Background()
	if err := redisShared.Set(ctx, regSessionKey(session.SenderJID), data, waRegSessionTTL).Err(); err != nil {
		// fallback to in-memory on Redis error
		waRegistrationSessions[session.SenderJID] = session
	}
}

func loadRegSession(senderJID string) (*waRegistrationSession, bool) {
	if redisShared != nil {
		ctx := context.Background()
		data, err := redisShared.Get(ctx, regSessionKey(senderJID)).Bytes()
		if err == nil {
			var session waRegistrationSession
			if json.Unmarshal(data, &session) == nil {
				return &session, true
			}
		}
	}
	// fallback to in-memory
	s, ok := waRegistrationSessions[senderJID]
	return s, ok
}

func deleteRegSession(senderJID string) {
	if redisShared != nil {
		ctx := context.Background()
		redisShared.Del(ctx, regSessionKey(senderJID))
	}
	delete(waRegistrationSessions, senderJID)
}

// in-memory fallback map (used when Redis is unavailable)
var waRegistrationSessions = make(map[string]*waRegistrationSession)

func startWARegistration(tenantID, senderJID, senderPhone string) {
	if db != nil {
		var exists bool
		if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE phone_number = $1)", senderPhone).Scan(&exists); err != nil {
			slog.Error("Failed to check existing user", "phone", senderPhone, "error", err)
		}
		if exists {
			sendWAMessage(tenantID, senderJID, "Nomor ini sudah terdaftar. Gunakan menu LOGIN atau hubungi admin.")
			return
		}
	}

	if redisShared != nil {
		ctx := context.Background()
		// Check both formats (62xxx and 0xxx) since website registration may store either
		localPhone := senderPhone
		if strings.HasPrefix(localPhone, "62") {
			localPhone = "0" + localPhone[2:]
		}
		intlPhone := senderPhone
		if strings.HasPrefix(intlPhone, "0") {
			intlPhone = "62" + intlPhone[1:]
		}
		for _, key := range []string{"otp:" + senderPhone, "otp:" + localPhone, "otp:" + intlPhone} {
			if val, _ := redisShared.Get(ctx, key).Result(); val != "" {
				sendWAMessage(tenantID, senderJID, "Nomor ini sedang dalam proses pendaftaran di website. Selesaikan verifikasi di website terlebih dahulu, atau tunggu 1 jam.")
				return
			}
		}
	}

	session := &waRegistrationSession{
		SenderJID:   senderJID,
		PhoneNumber: senderPhone,
		Step:        1,
		CreatedAt:   time.Now(),
	}
	saveRegSession(session)

	sendWAMessage(tenantID, senderJID, "Halo! Selamat datang di WCH Platform.\n\nUntuk mendaftar, silakan jawab pertanyaan berikut:\n\n1️⃣ Nama bisnis Anda? (ketik nama toko/usaha Anda)")
}

func handleBusinessNameStep(tenantID string, session *waRegistrationSession, rawText string) bool {
	session.BusinessName = strings.TrimSpace(rawText)
	session.Step = 2
	saveRegSession(session)
	sendWAMessage(tenantID, session.SenderJID, "✅ Nama bisnis: "+session.BusinessName+"\n\n2️⃣ Tipe bisnis Anda?\n1. Umum\n2. Warung/Kedai\n3. Klinik\n\nKetik nomor (1-3):")
	return true
}

func handleBusinessTypeStep(tenantID string, session *waRegistrationSession, upperText string) bool {
	bt := map[string]string{"1": "umum", "2": "warung", "3": "clinic"}
	btDesc := map[string]string{"1": "Umum", "2": "Warung/Kedai", "3": "Klinik"}
	btVal, ok := bt[upperText]
	if !ok {
		sendWAMessage(tenantID, session.SenderJID, "❌ Pilih angka 1, 2, atau 3 saja.")
		return true
	}
	session.BusinessType = btVal
	session.Step = 3
	saveRegSession(session)
	sendWAMessage(tenantID, session.SenderJID, "✅ Tipe bisnis: "+btDesc[upperText]+"\n\n3️⃣ Buat username untuk login (huruf, angka, underscore, min 3 karakter):")
	return true
}

func handleUsernameStep(tenantID string, session *waRegistrationSession, rawText string) bool {
	username := strings.TrimSpace(rawText)
	if len(username) < 3 {
		sendWAMessage(tenantID, session.SenderJID, "❌ Username minimal 3 karakter. Coba lagi:")
		return true
	}
	session.Username = username
	session.Step = 4
	saveRegSession(session)
	sendWAMessage(tenantID, session.SenderJID, "✅ Username: "+session.Username+"\n\n4️⃣ Buat password (minimal 6 karakter):")
	return true
}

func handlePasswordStep(tenantID string, session *waRegistrationSession, rawText string) bool {
	if len(rawText) < 6 {
		sendWAMessage(tenantID, session.SenderJID, "❌ Password minimal 6 karakter. Coba lagi:")
		return true
	}
	session.Password = rawText
	session.Step = 5
	saveRegSession(session)
	sendWAMessage(tenantID, session.SenderJID, "✅ Password tersimpan.\n\n5️⃣ Konfirmasi nomor HP Anda (format: 08xx atau 628xx):")
	return true
}

func handlePhoneConfirmStep(tenantID string, session *waRegistrationSession, rawText string) bool {
	phone := strings.TrimSpace(rawText)
	normalizedInput := phone
	if strings.HasPrefix(normalizedInput, "0") {
		normalizedInput = "62" + normalizedInput[1:]
	}
	normalizedSession := session.PhoneNumber
	if strings.HasPrefix(normalizedSession, "0") {
		normalizedSession = "62" + normalizedSession[1:]
	}
	if normalizedInput != normalizedSession {
		sendWAMessage(tenantID, session.SenderJID, "❌ Nomor HP tidak cocok. Masukkan nomor HP yang sama dengan nomor WA ini:")
		return true
	}
	submitWARegistration(tenantID, session)
	return true
}

func handleWARegistrationStep(tenantID string, session *waRegistrationSession, rawText, upperText string) bool {
	switch session.Step {
	case 1:
		return handleBusinessNameStep(tenantID, session, rawText)
	case 2:
		return handleBusinessTypeStep(tenantID, session, upperText)
	case 3:
		return handleUsernameStep(tenantID, session, rawText)
	case 4:
		return handlePasswordStep(tenantID, session, rawText)
	case 5:
		return handlePhoneConfirmStep(tenantID, session, rawText)
	}
	return false
}

func submitWARegistration(tenantID string, session *waRegistrationSession) {
	authSvcURL := getAuthServiceURL()
	payload := map[string]any{
		"phoneNumber":  session.PhoneNumber,
		"username":     session.Username,
		"password":     session.Password,
		"businessName": session.BusinessName,
		"businessType": session.BusinessType,
		"source":       "wa",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(authSvcURL+"/register-wa", contentTypeJSON, bytes.NewReader(body))
	if err != nil || resp == nil {
		sendWAMessage(tenantID, session.SenderJID, "❌ Gagal mendaftar. Silakan coba lagi nanti.")
		return
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	deleteRegSession(session.SenderJID)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		sendWAMessage(tenantID, session.SenderJID, "🎉 Pendaftaran berhasil!\n\nUsername: "+session.Username+"\n\nSilakan login di website dengan username dan password yang sudah dibuat.")
	} else {
		msg := "Pendaftaran gagal."
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		sendWAMessage(tenantID, session.SenderJID, "❌ Gagal: "+msg+"\n\nKetik REG untuk mulai ulang.")
	}
}

func handleWAVerifyOTP(tenantID, senderJID, code string) {
	if len(code) != 6 {
		sendWAMessage(tenantID, senderJID, "❌ Format salah. Ketik: VERIF <6-digit-kode>\n\nContoh: VERIF 123456")
		return
	}

	authSvcURL := getAuthServiceURL()
	payload := map[string]any{
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

	var result map[string]any
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
