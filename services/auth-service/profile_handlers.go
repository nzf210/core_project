package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func handleUploadProfileLogo(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAuth(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authentication required"})
		return
	}

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	file, ext, err := parseProfileLogoUpload(w, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: err.Error()})
		return
	}
	defer file.Close()

	logoURL, err := saveProfileLogo(claims.TenantID, file, ext)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}

	if err := updateProfileLogoURL(claims.TenantID, logoURL); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to update logo URL"})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Logo uploaded successfully", Data: map[string]any{"logo_url": logoURL}})
}

func parseProfileLogoUpload(w http.ResponseWriter, r *http.Request) (io.ReadCloser, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return nil, "", fmt.Errorf("File terlalu besar (max 2MB)")
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		return nil, "", fmt.Errorf("Logo file is required")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isValidLogoExtension(ext) {
		file.Close()
		return nil, "", fmt.Errorf("Format file tidak didukung (PNG, JPG, WebP)")
	}

	return file, ext, nil
}

func saveProfileLogo(tenantID string, file io.Reader, ext string) (string, error) {
	uploadDir := getUploadDir()
	os.MkdirAll(filepath.Join(uploadDir, "logos"), 0755)

	outExt := normalizeExtension(ext)
	cleanupProfileLogos(uploadDir, tenantID, outExt)

	filename := tenantID + outExt
	outPath := filepath.Join(uploadDir, "logos", filename)

	dst, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		return "", err
	}

	return "/uploads/logos/" + filename, nil
}

func cleanupProfileLogos(uploadDir, tenantID, keepExt string) {
	// Validate tenant ID to prevent path traversal
	if !uuidRE.MatchString(tenantID) {
		return
	}
	oldExts := []string{".png", ".jpg", ".jpeg", ".webp"}
	for _, e := range oldExts {
		if e != keepExt {
			os.Remove(filepath.Join(uploadDir, "logos", tenantID+e))
		}
	}
}

func updateProfileLogoURL(tenantID, logoURL string) error {
	ctx := context.Background()
	_, err := DB.Exec(ctx, `UPDATE tenants SET logo_url = $1, updated_at = NOW() WHERE id = $2`, logoURL, tenantID)
	return err
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireAuth(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authentication required"})
		return
	}

	ctx := context.Background()
	switch r.Method {
	case http.MethodGet:
		getProfileData(ctx, w, claims)
	case http.MethodPut:
		updateProfileData(ctx, w, r, claims)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
	}
}

func getProfileData(ctx context.Context, w http.ResponseWriter, claims *Claims) {
	var username, role string
	var phoneNumber, email, businessName, waNumber, logoURL, businessAddress, businessType, plan, tenantName *string
	var isFrozen, onboardingCompleted, mustChangePw bool
	err := DB.QueryRow(ctx, `
		SELECT u.username, u.email, u.phone_number, u.role,
		       COALESCE(t.business_name, t.name), t.wa_number, t.logo_url, t.business_address, t.business_type, t.plan, t.name, COALESCE(t.is_frozen, false), COALESCE(t.onboarding_completed, false), u.must_change_password
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.id = $1 AND u.tenant_id = $2
	`, claims.UserID, claims.TenantID).Scan(
		&username, &email, &phoneNumber, &role,
		&businessName, &waNumber, &logoURL, &businessAddress, &businessType, &plan, &tenantName, &isFrozen, &onboardingCompleted, &mustChangePw,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "User not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"username":             username,
			"email":                derefStr(email),
			"phone_number":         derefStr(phoneNumber),
			"role":                 role,
			"business_name":        derefStr(businessName),
			"wa_number":            derefStr(waNumber),
			"logo_url":             derefStr(logoURL),
			"business_address":     derefStr(businessAddress),
			"business_type":        derefStr(businessType),
			"plan":                 derefStr(plan),
			"tenant_id":            claims.TenantID,
			"is_frozen":            isFrozen,
			"onboarding_completed": onboardingCompleted,
			"must_change_password": mustChangePw,
		},
	})
}

func updateProfileData(ctx context.Context, w http.ResponseWriter, r *http.Request, claims *Claims) {
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	if req.NewPassword != "" && !updatePassword(ctx, w, req, claims.UserID) {
		return
	}

	if req.Username != "" && !updateUsername(ctx, w, req.Username, claims) {
		return
	}

	if req.PhoneNumber != "" {
		DB.Exec(ctx, "UPDATE users SET phone_number = $1 WHERE id = $2 AND tenant_id = $3", req.PhoneNumber, claims.UserID, claims.TenantID)
	}

	updateTenantFields(ctx, req, claims.TenantID)
	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Profile updated successfully"})
}

func updatePassword(ctx context.Context, w http.ResponseWriter, req UpdateProfileRequest, userID string) bool {
	if req.OldPassword == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "old_password is required to change password"})
		return false
	}
	var currentHash string
	err := DB.QueryRow(ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&currentHash)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: msgInternalServerError})
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword)); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Old password is incorrect"})
		return false
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	DB.Exec(ctx, "UPDATE users SET password_hash = $1, must_change_password = false WHERE id = $2", string(newHash), userID)
	return true
}

func updateUsername(ctx context.Context, w http.ResponseWriter, username string, claims *Claims) bool {
	var exists bool
	err := DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND id != $2)", username, claims.UserID).Scan(&exists)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Terjadi kesalahan saat memeriksa username"})
		return false
	}
	if exists {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Username sudah digunakan"})
		return false
	}
	_, err = DB.Exec(ctx, "UPDATE users SET username = $1 WHERE id = $2 AND tenant_id = $3", username, claims.UserID, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal menyimpan username"})
		return false
	}
	return true
}

func updateTenantFields(ctx context.Context, req UpdateProfileRequest, tenantID string) {
	tenantUpdates := []string{}
	tenantArgs := []any{}
	argIdx := 1
	if req.BusinessName != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("business_name = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.BusinessName)
		argIdx++
	}
	if req.BusinessAddress != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("business_address = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.BusinessAddress)
		argIdx++
	}
	if req.BusinessType != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("business_type = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.BusinessType)
		argIdx++
	}
	if req.WaNumber != "" {
		tenantUpdates = append(tenantUpdates, fmt.Sprintf("wa_number = $%d", argIdx))
		tenantArgs = append(tenantArgs, req.WaNumber)
		argIdx++
	}
	if len(tenantUpdates) > 0 {
		tenantArgs = append(tenantArgs, tenantID)
		// Safe: tenantUpdates contains only "$N" placeholders built via fmt.Sprintf above, not user input
		query := "UPDATE tenants SET " + strings.Join(tenantUpdates, ", ") + fmt.Sprintf(", updated_at = NOW() WHERE id = $%d", argIdx)
		DB.Exec(ctx, query, tenantArgs...)
	}
}
