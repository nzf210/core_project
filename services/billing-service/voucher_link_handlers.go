package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

type GenerateVoucherLinksReq struct {
	ProgramID string `json:"program_id"`
	Count     int    `json:"count"`
	ValidDays int    `json:"valid_days"`
	BaseURL   string `json:"base_url"`
}

func handleAdminGenerateVoucherLinks(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req GenerateVoucherLinksReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}
	if req.ProgramID == "" || req.Count <= 0 || req.Count > 1000 {
		response.Error(w, http.StatusBadRequest, "program_id and count (1-1000) required", nil)
		return
	}

	var (
		planID             string
		durationMonths    int
		programExpiresAt   *time.Time
	)
	err := DB.QueryRow(r.Context(), `
		SELECT target_plan_id, COALESCE(duration_months, 1), expires_at
		FROM voucher_programs WHERE id = $1 AND is_active = true
	`, req.ProgramID).Scan(&planID, &durationMonths, &programExpiresAt)
	if err != nil || planID == nilStr() {
		response.Error(w, http.StatusBadRequest, "Program not found or inactive", nil)
		return
	}

	var linkExpiresAt time.Time
	if req.ValidDays > 0 {
		linkExpiresAt = time.Now().AddDate(0, 0, req.ValidDays)
	} else if programExpiresAt != nil {
		linkExpiresAt = *programExpiresAt
	} else {
		linkExpiresAt = time.Now().AddDate(1, 0, 0)
	}

	creatorID, _ := r.Context().Value(auth.UserIDKey).(string)

	baseURL := req.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("PUBLIC_BASE_URL")
		if baseURL == "" {
			baseURL = "https://app.wch.id"
		}
	}

	type linkOut struct {
		Token string `json:"token"`
		URL   string `json:"url"`
	}
	links := make([]linkOut, 0, req.Count)

	for i := 0; i < req.Count; i++ {
		tok, err := signVoucherToken(req.ProgramID, planID, durationMonths, linkExpiresAt)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to sign token", err)
			return
		}
		hash := hashToken(tok)
		prefix := tok[:8]
		_, err = DB.Exec(r.Context(), `
			INSERT INTO voucher_links (program_id, token_hash, token_prefix, created_by, expires_at, is_active)
			VALUES ($1, $2, $3, $4, $5, true)
		`, req.ProgramID, hash, prefix, creatorID, linkExpiresAt)
		if err != nil {
			slog.Warn("Failed to persist link", "error", err)
			continue
		}
		links = append(links, linkOut{
			Token: tok,
			URL:   fmt.Sprintf("%s/redeem?token=%s", baseURL, tok),
		})
	}

	_, _ = DB.Exec(r.Context(), `
		INSERT INTO voucher_generation_logs (program_id, generated_by, count, prefix)
		VALUES ($1, $2, $3, $4)
	`, req.ProgramID, creatorID, len(links), "")

	slog.Info("Voucher links generated", "program_id", req.ProgramID, "count", len(links), "by", creatorID)

	response.JSON(w, http.StatusOK, "Links generated", map[string]interface{}{
		"program_id": req.ProgramID,
		"count":      len(links),
		"expires_at": linkExpiresAt.Format(time.RFC3339),
		"links":      links,
	})
}

func handleAdminListVoucherLinks(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	programID := r.URL.Query().Get("program_id")
	redeemedOnly := r.URL.Query().Get("redeemed") == "true"

	var (
		rows pgxRows
		err  error
	)
	if programID != "" {
		rows, err = DB.Query(r.Context(), `
			SELECT id, program_id, token_prefix, redeemed_by, redeemed_at, expires_at, is_active, created_at
			FROM voucher_links WHERE program_id = $1
			  AND ($2 = false OR redeemed_by IS NOT NULL)
			ORDER BY created_at DESC LIMIT 200
		`, programID, redeemedOnly)
	} else {
		rows, err = DB.Query(r.Context(), `
			SELECT id, program_id, token_prefix, redeemed_by, redeemed_at, expires_at, is_active, created_at
			FROM voucher_links
			WHERE ($1 = false OR redeemed_by IS NOT NULL)
			ORDER BY created_at DESC LIMIT 200
		`, redeemedOnly)
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list links", err)
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, progID, prefix               string
			redeemedBy                       *string
			redeemedAt, expiresAt, createdAt time.Time
			isActive                         bool
		)
		if rows.Scan(&id, &progID, &prefix, &redeemedBy, &redeemedAt, &expiresAt, &isActive, &createdAt) == nil {
			entry := map[string]interface{}{
				"id":          id,
				"program_id":  progID,
				"prefix":      prefix,
				"redeemed_by": redeemedBy,
				"redeemed_at": redeemedAt.Format(time.RFC3339),
				"expires_at":  expiresAt.Format(time.RFC3339),
				"is_active":   isActive,
				"created_at":  createdAt.Format(time.RFC3339),
			}
			out = append(out, entry)
		}
	}
	response.JSON(w, http.StatusOK, "Links retrieved", out)
}

type pgxRows interface {
	Next() bool
	Close()
	Scan(...any) error
}
