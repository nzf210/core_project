package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func handleUploadTenantLogo(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	tenantID := r.URL.Query().Get("id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "File terlalu besar (max 2MB)"})
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Logo file is required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Format file tidak didukung (PNG, JPG, WebP)"})
		return
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	os.MkdirAll(filepath.Join(uploadDir, "logos"), 0755)

	outExt := ".png"
	switch ext {
	case ".jpg", ".jpeg":
		outExt = ".jpg"
	case ".webp":
		outExt = ".webp"
	}

	// Delete existing old files with different extensions before writing new one
	oldExts := []string{".png", ".jpg", ".jpeg", ".webp"}
	for _, e := range oldExts {
		if e != outExt {
			os.Remove(filepath.Join(uploadDir, "logos", tenantID+e))
		}
	}

	filename := tenantID + outExt
	outPath := filepath.Join(uploadDir, "logos", filename)
	dst, err := os.Create(outPath)
	if err != nil {
		slog.Error("Failed to create logo file", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		slog.Error("Failed to write logo file", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}

	logoURL := "/uploads/logos/" + filename
	ctx := context.Background()
	_, err = DB.Exec(ctx, `UPDATE tenants SET logo_url=$1, updated_at=NOW() WHERE id=$2`, logoURL, tenantID)
	if err != nil {
		slog.Error("Failed to update logo_url", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to update logo URL"})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Logo berhasil diupload", Data: map[string]interface{}{"logo_url": logoURL}})
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ─────────────────────────────────────────────
// Telegram Auth Handlers
// ─────────────────────────────────────────────

// sendTelegramMessage sends a text message to a Telegram chat ID using the Telegram Bot API.
func sendTelegramMessage(chatID, message string) error {
	if telegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not configured")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// sendTelegramOTP sends an OTP message to a Telegram chat ID using the Telegram Bot API.
func sendTelegramOTP(chatID, message string) error {
	if telegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not configured")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramBotToken)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	body, _ := json.Marshal(payload)

	slog.Info("[TELEGRAM:OTP] Calling Telegram API", "chatID", chatID, "url", apiURL, "payload", string(body))

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Error("[TELEGRAM:OTP] HTTP request failed", "chatID", chatID, "error", err)
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	slog.Info("[TELEGRAM:OTP] API response", "chatID", chatID, "status", resp.StatusCode, "body", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// handleTelegramRegister starts registration via Telegram Bot.
// Reuses the same Redis OTP key ("otp:{phone}") so /verify-otp works for both WA and Telegram.
