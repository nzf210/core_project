package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
			os.Remove(filepath.Join(uploadDir, "logos", claims.TenantID+e))
		}
	}

	filename := claims.TenantID + outExt
	outPath := filepath.Join(uploadDir, "logos", filename)
	dst, err := os.Create(outPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to save logo"})
		return
	}

	logoURL := "/uploads/logos/" + filename
	ctx := context.Background()
	_, err = DB.Exec(ctx, `UPDATE tenants SET logo_url = $1, updated_at = NOW() WHERE id = $2`, logoURL, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to update logo URL"})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Logo uploaded successfully", Data: map[string]interface{}{"logo_url": logoURL}})
}
