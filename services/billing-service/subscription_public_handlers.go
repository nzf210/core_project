package main

import (
	"net/http"
	"time"

	"core_project/shared/sdk/response"
)

func handleListPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT p.id, p.name, p.description, p.price_monthly, p.price_yearly, p.is_active, p.sort_order
		FROM saas_plans p
		WHERE p.is_active = true
		ORDER BY p.sort_order ASC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch plans", err)
		return
	}
	defer rows.Close()

	var plans []planWithFeatures
	for rows.Next() {
		var p planRow
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceMonthly, &p.PriceYearly, &p.IsActive, &p.SortOrder); err != nil {
			continue
		}

		featRows, err := DB.Query(ctx, `
			SELECT feature_key, feature_name, feature_value, is_enabled
			FROM plan_features WHERE plan_id = $1 AND is_enabled = true
			ORDER BY feature_name ASC
		`, p.ID)
		if err == nil {
			var feats []featureRow
			for featRows.Next() {
				var f featureRow
				if featRows.Scan(&f.FeatureKey, &f.FeatureName, &f.FeatureValue, &f.IsEnabled) == nil {
					feats = append(feats, f)
				}
			}
			featRows.Close()
			plans = append(plans, planWithFeatures{planRow: p, Features: feats})
		} else {
			plans = append(plans, planWithFeatures{planRow: p})
		}
	}

	response.JSON(w, http.StatusOK, "Plans retrieved", plans)
}

// ─────────────────────────────────────────────
// Public: Validate Voucher
// ─────────────────────────────────────────────

func handleValidateVoucher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		response.Error(w, http.StatusBadRequest, "Missing voucher code", nil)
		return
	}

	ctx := r.Context()

	var programID, programName, voucherType string
	var discountValue, durationMonths int
	var targetPlanID *string
	var expiresAt *time.Time
	var maxUses, usesCount int
	var isActive bool

	err := DB.QueryRow(ctx, `
		SELECT vp.id, vp.name, vp.voucher_type, vp.discount_value, vp.duration_months,
		       vp.target_plan_id, vp.expires_at, vp.max_uses, vp.uses_count, vp.is_active
		FROM voucher_programs vp
		JOIN voucher_codes vc ON vc.program_id = vp.id
		WHERE vc.code = $1 AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.starts_at IS NULL OR vp.starts_at <= NOW())
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
		LIMIT 1
	`, code).Scan(&programID, &programName, &voucherType, &discountValue, &durationMonths,
		&targetPlanID, &expiresAt, &maxUses, &usesCount, &isActive)

	if err != nil {
		response.Error(w, http.StatusBadRequest, "Voucher invalid or expired", nil)
		return
	}

	if maxUses > 0 && usesCount >= maxUses {
		response.Error(w, http.StatusBadRequest, "Voucher quota exceeded", nil)
		return
	}

	// Check plan name
	planName := "All Plans"
	if targetPlanID != nil && *targetPlanID != "" {
		var pn string
		if err := DB.QueryRow(ctx, "SELECT name FROM saas_plans WHERE id = $1", *targetPlanID).Scan(&pn); err == nil {
			planName = pn
		}
	}

	var expStr *string
	if expiresAt != nil {
		s := expiresAt.Format(time.RFC3339)
		expStr = &s
	}

	response.JSON(w, http.StatusOK, "Voucher valid", map[string]interface{}{
		"program_id":      programID,
		"program_name":    programName,
		"voucher_type":    voucherType,
		"discount_value":  discountValue,
		"target_plan":     planName,
		"target_plan_id":  targetPlanID,
		"duration_months": durationMonths,
		"expires_at":      expStr,
	})
}

// ─────────────────────────────────────────────
// Protected: Subscribe
// ─────────────────────────────────────────────
