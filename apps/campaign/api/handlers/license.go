package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"core_project/apps/campaign/api/repository"
)

// HandleSuperadminGenerateLicense generates a manual B2B license key.
func HandleSuperadminGenerateLicense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	type Request struct {
		LicenseKey   string   `json:"license_key"`
		ElectionType string   `json:"election_type"`
		BaseQuota    int      `json:"base_quota"`
		Addons       []string `json:"addons"`
		Tokens       int      `json:"tokens"`
		PriceCents   int64    `json:"price_cents"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid JSON"})
		return
	}

	addonsJSON, _ := json.Marshal(req.Addons)

	ctx := context.Background()
	query := `
		INSERT INTO campaign_licenses (license_key, election_type, base_quota, addons, wargame_tokens, price_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := repository.DB.Exec(ctx, query,
		strings.ToUpper(req.LicenseKey), req.ElectionType, req.BaseQuota, string(addonsJSON), req.Tokens, req.PriceCents)

	if err != nil {
		slog.Error("Failed to create license", "key", req.LicenseKey, "error", err)
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create license: " + err.Error()})
		return
	}

	slog.Info("License generated", "key", req.LicenseKey, "price_cents", req.PriceCents, "addons", req.Addons)
	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "License generated successfully",
		Data:    req,
	})
}

// HandleListLicenses — superadmin dashboard: list all generated licenses with usage status.
// GET /superadmin/licenses?used=false&limit=50
func HandleListLicenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	usedFilter := r.URL.Query().Get("used")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		var n int
		if _, err := fmt.Sscanf(limitStr, "%d", &n); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	ctx := context.Background()

	query := `
		SELECT license_key, election_type, base_quota, addons, wargame_tokens,
		       price_cents, is_used, used_by_tenant, used_at, created_at
		FROM campaign_licenses
		WHERE ($1 = '' OR is_used = $1::boolean)
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := repository.DB.Query(ctx, query, usedFilter, limit)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to list licenses"})
		return
	}
	defer rows.Close()

	type License struct {
		LicenseKey   string  `json:"license_key"`
		ElectionType string  `json:"election_type"`
		BaseQuota    int     `json:"base_quota"`
		Addons       string  `json:"addons"`
		WargameTokens int    `json:"wargame_tokens"`
		PriceCents   int64   `json:"price_cents"`
		IsUsed       bool    `json:"is_used"`
		UsedByTenant *string `json:"used_by_tenant,omitempty"`
		UsedAt       *string `json:"used_at,omitempty"`
		CreatedAt    string  `json:"created_at"`
	}

	var licenses []License
	for rows.Next() {
		var lc License
		var usedByTenant *string
		var usedAt *string
		if err := rows.Scan(&lc.LicenseKey, &lc.ElectionType, &lc.BaseQuota, &lc.Addons,
			&lc.WargameTokens, &lc.PriceCents, &lc.IsUsed, &usedByTenant, &usedAt, &lc.CreatedAt); err == nil {
			lc.UsedByTenant = usedByTenant
			lc.UsedAt = usedAt
			licenses = append(licenses, lc)
		}
	}
	if licenses == nil {
		licenses = []License{}
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]interface{}{"count": len(licenses), "licenses": licenses},
	})
}

// HandleTenantActiveAddons — Tenant-side: returns current active addons + election_type + remaining tokens per campaign.
// GET /licenses/active?campaign_id=...
func HandleTenantActiveAddons(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant ID"})
		return
	}

	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	campaignID := r.URL.Query().Get("campaign_id")
	if campaignID == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "campaign_id required"})
		return
	}

	ctx := context.Background()

	var electionType string
	var maxVoters, wargameTokens int
	var activeAddonsBytes []byte
	err := repository.DB.QueryRow(ctx, `
		SELECT election_type, max_voters, wargame_tokens, active_addons
		FROM campaigns
		WHERE id = $1 AND tenant_id = $2
	`, campaignID, tenantID).Scan(&electionType, &maxVoters, &wargameTokens, &activeAddonsBytes)
	if err != nil {
		WriteJSON(w, http.StatusNotFound, APIResponse{Message: "Campaign not found"})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"campaign_id":     campaignID,
			"election_type":  electionType,
			"max_voters":      maxVoters,
			"wargame_tokens":  wargameTokens,
			"active_addons":   string(activeAddonsBytes),
		},
	})
}

// HandleRedeemLicense called by tenant to unlock features
func HandleRedeemLicense(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant ID"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	type Request struct {
		LicenseKey string `json:"license_key"`
		CampaignID string `json:"campaign_id"`
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid JSON"})
		return
	}

	ctx := context.Background()
	var electionType, addons string
	var baseQuota, tokens int
	var isUsed bool

	err := repository.DB.QueryRow(ctx, `
		SELECT election_type, base_quota, addons, wargame_tokens, is_used 
		FROM campaign_licenses WHERE license_key = $1
	`, strings.ToUpper(req.LicenseKey)).Scan(&electionType, &baseQuota, &addons, &tokens, &isUsed)

	if err != nil {
		slog.Warn("Invalid license key attempt", "tenant_id", tenantID, "key", req.LicenseKey)
		WriteJSON(w, http.StatusNotFound, APIResponse{Message: "Invalid license key"})
		return
	}

	if isUsed {
		slog.Warn("Expired/Used license key attempt", "tenant_id", tenantID, "key", req.LicenseKey)
		WriteJSON(w, http.StatusGone, APIResponse{Message: "License already used"})
		return
	}

	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction failed"})
		return
	}
	defer tx.Rollback(ctx)

	// 1. Update Campaign Features
	_, err = tx.Exec(ctx, `
		UPDATE campaigns SET 
			election_type = $1,
			max_voters = GREATEST(max_voters, $2),
			active_addons = active_addons || $3::jsonb,
			wargame_tokens = wargame_tokens + $4,
			updated_at = NOW()
		WHERE id = $5 AND tenant_id = $6
	`, electionType, baseQuota, addons, tokens, req.CampaignID, tenantID)

	if err != nil {
		slog.Error("Failed to update campaign via license", "tenant_id", tenantID, "error", err)
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to unlock features"})
		return
	}

	// 2. Mark License as Used
	_, err = tx.Exec(ctx, `
		UPDATE campaign_licenses SET is_used = true, used_by_tenant = $1, used_at = NOW()
		WHERE license_key = $2
	`, tenantID, strings.ToUpper(req.LicenseKey))

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to burn license"})
		return
	}

	tx.Commit(ctx)
	slog.Info("License redeemed successfully", "tenant_id", tenantID, "key", req.LicenseKey, "campaign_id", req.CampaignID)

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "License redeemed, features unlocked!",
	})
}
