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
		otpKey := "otp:" + senderPhone
		if val, _ := redisShared.Get(ctx, otpKey).Result(); val != "" {
			sendWAMessage(tenantID, senderJID, "Nomor ini sedang dalam proses pendaftaran di website. Selesaikan verifikasi di website terlebih dahulu, atau tunggu 1 jam.")
			return
		}
	}

	waRegistrationSessions[senderJID] = &waRegistrationSession{
		SenderJID:   senderJID,
		PhoneNumber: senderPhone,
		Step:        1,
		CreatedAt:   time.Now(),
	}

	sendWAMessage(tenantID, senderJID, "Halo! Selamat datang di WCH Platform.\n\nUntuk mendaftar, silakan jawab pertanyaan berikut:\n\n1️⃣ Nama bisnis Anda叫什么? (ketik nama toko/usaha Anda)")
}

func handleBusinessNameStep(tenantID string, session *waRegistrationSession, rawText string) bool {
	session.BusinessName = strings.TrimSpace(rawText)
	session.Step = 2
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
	sendWAMessage(tenantID, session.SenderJID, "✅ Tipe bisnis: "+btDesc[upperText]+"\n\n3️⃣ Buat username untuk login (huruf, angka, underscore, min 3 karakter):")
	return true
}

func handleUsernameStep(tenantID string, session *waRegistrationSession, rawText string) bool {
	if len(rawText) < 3 || len(strings.Trim(rawText, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")) != 0 {
		sendWAMessage(tenantID, session.SenderJID, "❌ Username minimal 3 karakter, hanya huruf, angka, dan underscore.")
		return true
	}
	if db != nil {
		var exists bool
		if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", rawText).Scan(&exists); err != nil {
			slog.Error("Failed to check username", "username", rawText, "error", err)
		}
		if exists {
			sendWAMessage(tenantID, session.SenderJID, "❌ Username sudah digunakan. Coba yang lain:")
			return true
		}
	}
	session.Username = rawText
	session.Step = 4
	sendWAMessage(tenantID, session.SenderJID, "✅ Username: "+session.Username+"\n\n4️⃣ Buat password (minimal 6 karakter):")
	return true
}

func handlePasswordStep(tenantID string, session *waRegistrationSession, rawText string) bool {
	if len(rawText) < 6 {
		sendWAMessage(tenantID, session.SenderJID, "❌ Password minimal 6 karakter.")
		return true
	}
	session.Password = rawText
	session.Step = 5
	sendWAMessage(tenantID, session.SenderJID, "✅ Password tersimpan.\n\n5️⃣ Konfirmasi nomor HP: "+session.PhoneNumber+"\n\nKetik YA jika benar, atau ketik ulang nomor Anda:")
	return true
}

func handlePhoneConfirmationStep(tenantID string, session *waRegistrationSession, rawText, upperText string) bool {
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
		return handlePhoneConfirmationStep(tenantID, session, rawText, upperText)
	}
	return false
}

func submitWARegistration(tenantID string, session *waRegistrationSession) {
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
