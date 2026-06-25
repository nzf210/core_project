package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"core_project/shared/sdk/response"
)

func handleAdminLicenses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	usedFilter := r.URL.Query().Get("used")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	query := `
		SELECT id::text, license_key, COALESCE(program_name, ''), COALESCE(election_type, ''),
		       COALESCE(max_voters, base_quota, 5000), COALESCE(wargame_tokens, 0),
		       COALESCE(validity_days, 365), is_used, used_by_tenant::text, created_at, used_at
		FROM campaign_licenses
		WHERE 1=1
	`

	if usedFilter == "true" {
		query += " AND is_used = true"
	} else if usedFilter == "false" {
		query += " AND is_used = false"
	}

	query += " ORDER BY created_at DESC LIMIT $1"

	rows, err := DB.Query(ctx, query, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch licenses", err)
		return
	}
	defer rows.Close()

	var licenses []map[string]interface{}
	for rows.Next() {
		var id, key, programName, electionType string
		var maxVoters, wargameTokens, validityDays int
		var isUsed bool
		var usedByTenantID *string
		var createdAt time.Time
		var usedAt *time.Time

		if err := rows.Scan(&id, &key, &programName, &electionType, &maxVoters, &wargameTokens, &validityDays, &isUsed, &usedByTenantID, &createdAt, &usedAt); err != nil {
			continue
		}

		licenses = append(licenses, map[string]interface{}{
			"id":                id,
			"license_key":       key,
			"program_name":      programName,
			"election_type":     electionType,
			"max_voters":        maxVoters,
			"wargame_tokens":    wargameTokens,
			"validity_days":     validityDays,
			"is_used":           isUsed,
			"used_by_tenant_id": usedByTenantID,
			"created_at":        createdAt.Format(time.RFC3339),
			"used_at":           formatTimePtr(usedAt),
		})
	}

	if licenses == nil {
		licenses = []map[string]interface{}{}
	}

	response.JSON(w, http.StatusOK, "Licenses retrieved", licenses)
}

func handleAdminGenerateLicenses(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req struct {
		ElectionType  string `json:"election_type"`
		MaxVoters     int    `json:"max_voters"`
		WargameTokens int    `json:"wargame_tokens"`
		ValidityDays  int    `json:"validity_days"`
		Quantity      int    `json:"quantity"`
		ProgramName   string `json:"program_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	if req.ElectionType == "" {
		req.ElectionType = "pilkada"
	}
	if req.MaxVoters == 0 {
		req.MaxVoters = 10000
	}
	if req.ValidityDays == 0 {
		req.ValidityDays = 365
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	if req.Quantity > 50 {
		req.Quantity = 50
	}

	ctx := r.Context()
	var keys []map[string]interface{}

	for i := 0; i < req.Quantity; i++ {
		licenseKey := "WCH-" + strings.ToUpper(uuid.NewString()[:8]) + "-" + strings.ToUpper(req.ElectionType[:4])

		var id string
		err := DB.QueryRow(ctx, `
			INSERT INTO campaign_licenses (license_key, election_type, max_voters, wargame_tokens, validity_days, program_name)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (license_key) DO UPDATE SET license_key = EXCLUDED.license_key
			RETURNING id::text
		`, licenseKey, req.ElectionType, req.MaxVoters, req.WargameTokens, req.ValidityDays, req.ProgramName).Scan(&id)

		if err == nil {
			keys = append(keys, map[string]interface{}{
				"id":          id,
				"license_key": licenseKey,
				"days":        req.ValidityDays,
			})
		}
	}

	if len(keys) == 0 {
		response.Error(w, http.StatusInternalServerError, "Failed to generate licenses", nil)
		return
	}

	slog.Info("Campaign licenses generated", "count", len(keys), "election_type", req.ElectionType)
	response.JSON(w, http.StatusOK, "Licenses generated", map[string]interface{}{
		"keys": keys,
	})
}

func generateTicketNumber() string {
	now := time.Now()
	return fmt.Sprintf("TKT-%d-%s-%04d",
		now.Year(),
		now.Format("0102"),
		now.UnixNano()%10000)
}

func formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
