package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"core_project/apps/campaign/api/repository"
)

// HandleCampaignAffiliateRedeemReferral links a campaign tenant to an affiliate agent.
func HandleCampaignAffiliateRedeemReferral(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant ID"})
		return
	}

	type Request struct {
		ReferralCode string `json:"referral_code"`
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ReferralCode == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Referral code required"})
		return
	}

	ctx := context.Background()

	// Find the affiliate
	var affID int
	err := repository.DB.QueryRow(ctx,
		"SELECT id FROM affiliates WHERE referral_code = $1",
		strings.ToUpper(req.ReferralCode),
	).Scan(&affID)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid referral code"})
		return
	}

	// Link tenant to affiliate
	_, err = repository.DB.Exec(ctx,
		"UPDATE tenants SET referred_by_affiliate_id = $1 WHERE id = $2 AND referred_by_affiliate_id IS NULL",
		affID, tenantID,
	)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to apply referral: " + err.Error()})
		return
	}

	slog.Info("Campaign: Referral applied", "tenant_id", tenantID, "affiliate_id", affID)
	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Referral applied successfully"})
}

// HandleCampaignAffiliateLeaderboard returns public leaderboard (same as billing service).
func HandleCampaignAffiliateLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	rows, err := repository.DB.Query(context.Background(), `
		SELECT 
			u.full_name, 
			COUNT(DISTINCT ae.tenant_id) as total_closing, 
			SUM(ae.amount_cents) as total_revenue
		FROM affiliate_earnings ae
		JOIN affiliates a ON ae.affiliate_id = a.id
		JOIN users u ON a.user_id = u.id
		GROUP BY a.id, u.full_name
		ORDER BY total_revenue DESC
		LIMIT 10
	`)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to query leaderboard: " + err.Error()})
		return
	}
	defer rows.Close()

	type Leader struct {
		Name         string `json:"name"`
		TotalClosing int    `json:"total_closing"`
		TotalRevenue int64  `json:"total_revenue_cents"`
	}
	var leaders []Leader
	for rows.Next() {
		var l Leader
		var rawName string
		if err := rows.Scan(&rawName, &l.TotalClosing, &l.TotalRevenue); err == nil {
			parts := strings.Split(rawName, " ")
			if len(parts) > 1 {
				l.Name = parts[0] + " " + string(parts[1][0]) + "."
			} else {
				l.Name = parts[0]
			}
			leaders = append(leaders, l)
		}
	}

	WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: leaders})
}
