package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid payload"})
		return
	}

	ctx := context.Background()
	var userID string
	err := DB.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	if err == pgx.ErrNoRows {
		// Don't leak whether email exists
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "If the email is registered, a reset token will be sent."})
		return
	} else if err != nil {
		slog.Error("DB error in forgot password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal error"})
		return
	}

	// Generate a mock random token
	tokenStr := fmt.Sprintf("%x", time.Now().UnixNano())

	expiresAt := time.Now().Add(1 * time.Hour)
	_, err = DB.Exec(ctx, "INSERT INTO password_resets (email, token, expires_at) VALUES ($1, $2, $3)", req.Email, tokenStr, expiresAt)
	if err != nil {
		slog.Error("Failed to insert reset token", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal error"})
		return
	}

	// In a real application, send via SMTP here. For now, print to log.
	slog.Info("🔑 PASSWORD RESET TOKEN GENERATED (Simulating Email)", "email", req.Email, "token", tokenStr)

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "If the email is registered, a reset token will be sent."})
}

func handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid payload"})
		return
	}

	if req.NewPassword == "" || req.Token == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Token and new password are required"})
		return
	}

	ctx := context.Background()
	var email string
	err := DB.QueryRow(ctx, "SELECT email FROM password_resets WHERE token = $1 AND expires_at > NOW()", req.Token).Scan(&email)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid or expired token"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal error"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to hash password"})
		return
	}

	_, err = DB.Exec(ctx, "UPDATE users SET password_hash = $1 WHERE email = $2", string(hashedPassword), email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to reset password"})
		return
	}

	// Consume token
	DB.Exec(ctx, "DELETE FROM password_resets WHERE email = $1", email)

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Password has been successfully reset."})
}

// handleResetPasswordDefault - reset password ke default berdasarkan username + phone
func handleResetPasswordDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req ResetPasswordDefaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.Username == "" || req.PhoneNumber == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Username dan nomor HP wajib diisi"})
		return
	}

	ctx := context.Background()

	// Cari user berdasarkan username, pastikan phone number cocok
	var userID string
	var storedPhone sql.NullString
	err := DB.QueryRow(ctx, "SELECT id, phone_number FROM users WHERE username = $1", req.Username).Scan(&userID, &storedPhone)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Jika username terdaftar, password akan direset."})
		return
	} else if err != nil {
		slog.Error("DB error looking up user for password reset", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	// Validasi nomor HP cocok (jika user punya phone number)
	if storedPhone.Valid && storedPhone.String != req.PhoneNumber {
		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Jika username terdaftar, password akan direset."})
		return
	}

	// Default password hardcoded sesuai spesifikasi
	defaultPw := "x210wchsaasumkm"

	// Hash dengan bcrypt cost=12
	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPw), 12)
	if err != nil {
		slog.Error("Failed to hash default password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	// Update password di DB + set flag must_change_password
	_, err = DB.Exec(ctx, "UPDATE users SET password_hash = $1, must_change_password = true, updated_at = NOW() WHERE id = $2", string(hashed), userID)
	if err != nil {
		slog.Error("Failed to update password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	slog.Info("Password reset to default (force change required)", "username", req.Username, "phone", req.PhoneNumber)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Password berhasil direset ke default. Silakan login dan ubah password Anda.",
	})
}

// handleForceChangePassword - wajib dipanggil setelah reset password default
// User harus mengirim old_password (password default) + new_password
func handleForceChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	// Auth middleware sudah validate token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Missing authorization"})
		return
	}

	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid token"})
		return
	}

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "oldPassword dan newPassword wajib diisi"})
		return
	}

	if len(req.NewPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Password baru minimal 8 karakter"})
		return
	}

	ctx := context.Background()

	// Cek apakah user memang wajib ganti password
	var mustChange bool
	var currentHash string
	err = DB.QueryRow(ctx, "SELECT password_hash, must_change_password FROM users WHERE id = $1", claims.UserID).Scan(&currentHash, &mustChange)
	if err != nil {
		slog.Error("DB error checking force change password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	if !mustChange {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Password tidak perlu diganti"})
		return
	}

	// Verifikasi old password (password default)
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword)); err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Password lama tidak sesuai"})
		return
	}

	// Update password baru + reset flag
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 12)
	if err != nil {
		slog.Error("Failed to hash new password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	_, err = DB.Exec(ctx, "UPDATE users SET password_hash = $1, must_change_password = false, updated_at = NOW() WHERE id = $2", string(newHash), claims.UserID)
	if err != nil {
		slog.Error("Failed to update password", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	slog.Info("Password changed successfully after forced reset", "user_id", claims.UserID)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Password berhasil diubah. Silakan login kembali.",
	})
}
