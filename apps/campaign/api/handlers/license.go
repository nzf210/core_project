package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"core_project/apps/campaign/api/repository"
)

// HandleSuperadminGenerateLicense generates a manual B2B license key.
// Protected by Superadmin JWT claims middleware upstream.
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
		Tokens       int      `json:"wargame_tokens"`
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
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create license: " + err.Error()})
		return
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "License generated successfully",
		Data:    req,
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

	key := strings.ToUpper(req.LicenseKey)
	ctx := context.Background()

	// 1. Check if license is valid & unused
	var isUsed bool
	var baseQuota, tokens int
	var addons []byte
	
	err := repository.DB.QueryRow(ctx, `
		SELECT is_used, base_quota, wargame_tokens, addons 
		FROM campaign_licenses 
		WHERE license_key = $1
	`, key).Scan(&isUsed, &baseQuota, &tokens, &addons)

	if err != nil {
		WriteJSON(w, http.StatusNotFound, APIResponse{Message: "Invalid license key"})
		return
	}
	if isUsed {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "License key has already been used"})
		return
	}

	// 2. Transaction to redeem and inject to campaign
	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Transaction failed"})
		return
	}
	defer tx.Rollback(ctx)

	// Mark used
	_, err = tx.Exec(ctx, `
		UPDATE campaign_licenses 
		SET is_used = TRUE, used_by_tenant = $1, used_at = NOW() 
		WHERE license_key = $2
	`, tenantID, key)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update license"})
		return
	}

	// Inject to campaign
	// In PostgreSQL, to append JSONB array we can use the || operator.
	// For simplicity, here we just overwrite or merge. We'll overwrite max_voters if greater.
	_, err = tx.Exec(ctx, `
		UPDATE campaigns 
		SET wargame_tokens = wargame_tokens + $1,
		    max_voters = GREATEST(max_voters, $2),
		    active_addons = (
				SELECT jsonb_agg(DISTINCT e) 
				FROM (
					SELECT jsonb_array_elements_text(COALESCE(active_addons, '[]'::jsonb)) as e
					UNION
					SELECT jsonb_array_elements_text($3::jsonb) as e
				) sub
			)
		WHERE id = $4 AND tenant_id = $5
	`, tokens, baseQuota, string(addons), req.CampaignID, tenantID)

	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update campaign: " + err.Error()})
		return
	}

	tx.Commit(ctx)

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "License redeemed successfully! Features unlocked.",
	})
}
