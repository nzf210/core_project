package main

import (
	"context"
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

	ctx := r.Context()
	program, err := loadVoucherProgram(ctx, req.ProgramID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	linkExpiresAt := calculateLinkExpiry(req.ValidDays, program.ExpiresAt)
	creatorID, _ := ctx.Value(auth.UserIDKey).(string)
	baseURL := resolveBaseURL(req.BaseURL)

	links := generateVoucherLinks(ctx, req.ProgramID, req.Count, program, linkExpiresAt, creatorID, baseURL)

	logVoucherGeneration(ctx, req.ProgramID, creatorID, len(links))

	response.JSON(w, http.StatusOK, "Links generated", map[string]interface{}{
		"program_id": req.ProgramID,
		"count":      len(links),
		"expires_at": linkExpiresAt.Format(time.RFC3339),
		"links":      links,
	})
}

type voucherProgramData struct {
	PlanID         string
	DurationMonths int
	ExpiresAt      *time.Time
}

func loadVoucherProgram(ctx context.Context, programID string) (*voucherProgramData, error) {
	var prog voucherProgramData
	err := DB.QueryRow(ctx, `
		SELECT target_plan_id, COALESCE(duration_months, 1), expires_at
		FROM voucher_programs WHERE id = $1 AND is_active = true
	`, programID).Scan(&prog.PlanID, &prog.DurationMonths, &prog.ExpiresAt)

	if err != nil || prog.PlanID == nilStr() {
		return nil, fmt.Errorf("Program not found or inactive")
	}
	return &prog, nil
}

func calculateLinkExpiry(validDays int, programExpiresAt *time.Time) time.Time {
	if validDays > 0 {
		return time.Now().AddDate(0, 0, validDays)
	}
	if programExpiresAt != nil {
		return *programExpiresAt
	}
	return time.Now().AddDate(1, 0, 0)
}

func resolveBaseURL(requestedURL string) string {
	if requestedURL != "" {
		return requestedURL
	}
	if envURL := os.Getenv("PUBLIC_BASE_URL"); envURL != "" {
		return envURL
	}
	return "https://app.wch.id"
}

func generateVoucherLinks(ctx context.Context, programID string, count int, program *voucherProgramData, expiresAt time.Time, creatorID, baseURL string) []linkOut {
	links := make([]linkOut, 0, count)

	for i := 0; i < count; i++ {
		tok, err := signVoucherToken(programID, program.PlanID, program.DurationMonths, expiresAt)
		if err != nil {
			slog.Error("Failed to sign voucher token", "error", err)
			continue
		}

		if persistVoucherLink(ctx, programID, tok, creatorID, expiresAt) {
			links = append(links, linkOut{
				Token: tok,
				URL:   fmt.Sprintf("%s/redeem?token=%s", baseURL, tok),
			})
		}
	}

	return links
}

func persistVoucherLink(ctx context.Context, programID, token, creatorID string, expiresAt time.Time) bool {
	hash := hashToken(token)
	prefix := token[:8]

	_, err := DB.Exec(ctx, `
		INSERT INTO voucher_links (program_id, token_hash, token_prefix, created_by, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, true)
	`, programID, hash, prefix, creatorID, expiresAt)

	if err != nil {
		slog.Warn("Failed to persist link", "error", err)
		return false
	}
	return true
}

func logVoucherGeneration(ctx context.Context, programID, creatorID string, count int) {
	DB.Exec(ctx, `
		INSERT INTO voucher_generation_logs (program_id, generated_by, count, prefix)
		VALUES ($1, $2, $3, $4)
	`, programID, creatorID, count, "")

	slog.Info("Voucher links generated", "program_id", programID, "count", count, "by", creatorID)
}

type linkOut struct {
	Token string `json:"token"`
	URL   string `json:"url"`
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
			id, progID, prefix        string
			redeemedBy                *string
			redeemedAt, expiresAt     *time.Time
			createdAt                 time.Time
			isActive                  bool
		)
		if rows.Scan(&id, &progID, &prefix, &redeemedBy, &redeemedAt, &expiresAt, &isActive, &createdAt) == nil {
			var redeemedAtStr, expiresAtStr *string
			if redeemedAt != nil {
				s := redeemedAt.Format(time.RFC3339)
				redeemedAtStr = &s
			}
			if expiresAt != nil {
				s := expiresAt.Format(time.RFC3339)
				expiresAtStr = &s
			}
			entry := map[string]interface{}{
				"id":          id,
				"program_id":  progID,
				"prefix":      prefix,
				"redeemed_by": redeemedBy,
				"redeemed_at": redeemedAtStr,
				"expires_at":  expiresAtStr,
				"is_active":   isActive,
				"created_at":  createdAt.Format(time.RFC3339),
			}
			out = append(out, entry)
		}
	}
	if err := rows.Err(); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to iterate links", err)
		return
	}
	response.JSON(w, http.StatusOK, "Links retrieved", out)
}

type pgxRows interface {
	Next() bool
	Close()
	Scan(...any) error
	Err() error
}
