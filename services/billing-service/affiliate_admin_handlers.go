package main

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"core_project/shared/sdk/response"
)

func handleAdminReferralConfig(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getReferralConfig(w, r)
	case http.MethodPut, http.MethodPost:
		updateReferralConfig(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func getReferralConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var discountPct, commissionPct, minPurchase, maxCommission float64
	var isActive bool
	var linkBase string

	err := DB.QueryRow(ctx, `
		SELECT COALESCE(discount_percent,10), COALESCE(commission_percent,10),
		       COALESCE(min_purchase_rupiah,0), COALESCE(max_commission_rupiah,0),
		       COALESCE(is_active,true), COALESCE(referral_link_base,'wch.id/r')
		FROM referral_config WHERE id = 1
	`).Scan(&discountPct, &commissionPct, &minPurchase, &maxCommission, &isActive, &linkBase)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to load config", err)
		return
	}

	response.JSON(w, http.StatusOK, "Referral config loaded", map[string]interface{}{
		"discount_percent":       discountPct,
		"commission_percent":     commissionPct,
		"min_purchase_rupiah":    minPurchase,
		"max_commission_rupiah":  maxCommission,
		"is_active":              isActive,
		"referral_link_base":     linkBase,
	})
}

func updateReferralConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		DiscountPercent     float64 `json:"discount_percent"`
		CommissionPercent   float64 `json:"commission_percent"`
		MinPurchaseRupiah   int64   `json:"min_purchase_rupiah"`
		MaxCommissionRupiah int64   `json:"max_commission_rupiah"`
		IsActive            bool    `json:"is_active"`
		ReferralLinkBase    string  `json:"referral_link_base"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid body", err)
		return
	}

	if !validateReferralPercentages(req.DiscountPercent, req.CommissionPercent) {
		response.Error(w, http.StatusBadRequest, "Percentage must be 0-100", nil)
		return
	}

	if err := saveReferralConfig(ctx, &req); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update config", err)
		return
	}

	slog.Info("Referral config updated", "discount", req.DiscountPercent, "commission", req.CommissionPercent)
	response.JSON(w, http.StatusOK, "Referral config updated", map[string]interface{}{
		"discount_percent":      req.DiscountPercent,
		"commission_percent":    req.CommissionPercent,
		"min_purchase_rupiah":   req.MinPurchaseRupiah,
		"max_commission_rupiah": req.MaxCommissionRupiah,
		"is_active":             req.IsActive,
		"referral_link_base":    req.ReferralLinkBase,
	})
}
