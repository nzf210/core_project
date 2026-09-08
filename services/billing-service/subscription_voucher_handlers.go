package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

func extractTokenFromVoucherInput(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	if strings.Contains(input, "token=") {
		parts := strings.Split(input, "token=")
		if len(parts) > 1 {
			tokenPart := parts[1]
			if idx := strings.IndexAny(tokenPart, "&# "); idx != -1 {
				tokenPart = tokenPart[:idx]
			}
			return strings.TrimSpace(tokenPart)
		}
	}
	// Direct JWT token check: starts with "ey" and has at least two dots
	if strings.HasPrefix(input, "ey") && strings.Count(input, ".") >= 2 {
		return input
	}
	return ""
}

func handleRedeemVoucher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	tenantID, ok := r.Context().Value(auth.TenantIDKey).(string)
	if !ok || tenantID == "" {
		response.Error(w, http.StatusUnauthorized, response.MissingTenantID, nil)
		return
	}

	var req VoucherRedeemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, nil)
		return
	}

	cleanInput := strings.TrimSpace(req.Code)
	if cleanInput == "" {
		response.Error(w, http.StatusBadRequest, "Kode voucher wajib diisi", nil)
		return
	}

	ctx := r.Context()
	if linkToken := extractTokenFromVoucherInput(cleanInput); linkToken != "" {
		handleRedeemLinkVoucher(w, r, ctx, linkToken, tenantID)
		return
	}

	handleRedeemCodeVoucher(w, ctx, cleanInput, tenantID)
}

func handleRedeemLinkVoucher(w http.ResponseWriter, r *http.Request, ctx context.Context, linkToken, tenantID string) {
	resp, err := redeemVoucherLinkCore(ctx, linkToken, tenantID, r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, "Voucher redeemed successfully", map[string]interface{}{
		"program_name":   resp.PlanName,
		"voucher_type":   "link",
		"discount_value": 0,
		"target_plan":    resp.PlanName,
		"plan_id":        resp.PlanID,
		"ticket_id":      resp.TicketNumber,
		"ticket_number":  resp.TicketNumber,
		"validity_days":  resp.DurationMonths * 30,
		"amount_charged": 0,
		"expires_at":     resp.ExpiresAt,
	})
}

func calculateVoucherCharge(priceMonthly int64, voucherType string, discountValue int) int64 {
	switch voucherType {
	case "free_months":
		return 0
	case "discount_percent":
		return priceMonthly * int64(100-discountValue) / 100
	case "discount_fixed":
		return maxInt64(0, priceMonthly-int64(discountValue))
	default:
		return priceMonthly
	}
}

func resolveVoucherValidityDays(codeValidityDays, programDurationMonths int) int {
	if codeValidityDays > 0 {
		return codeValidityDays
	}
	if programDurationMonths > 0 {
		return programDurationMonths * 30
	}
	return 30
}

func handleRedeemCodeVoucher(w http.ResponseWriter, ctx context.Context, cleanInput, tenantID string) {
	var programID, programName, voucherType string
	var discountValue, programDurationMonths int
	var targetPlanID *string
	var expiresAt *time.Time
	var maxUses, usesCount int
	var codeValidityDays int

	err := DB.QueryRow(ctx, `
		SELECT vp.id, vp.name, vp.voucher_type, vp.discount_value, vp.duration_months,
		       vp.target_plan_id, vp.expires_at, vp.max_uses, vp.uses_count,
		       vc.validity_days
		FROM voucher_programs vp
		JOIN voucher_codes vc ON vc.program_id = vp.id
		WHERE UPPER(TRIM(vc.code)) = UPPER(TRIM($1)) AND vc.is_redeemed = false
		  AND vp.is_active = true
		  AND (vp.expires_at IS NULL OR vp.expires_at > NOW())
		LIMIT 1
	`, cleanInput).Scan(&programID, &programName, &voucherType, &discountValue, &programDurationMonths,
		&targetPlanID, &expiresAt, &maxUses, &usesCount, &codeValidityDays)

	if err != nil {
		response.Error(w, http.StatusBadRequest, "Voucher invalid or already used", nil)
		return
	}

	if maxUses > 0 && usesCount >= maxUses {
		response.Error(w, http.StatusBadRequest, "Voucher quota exceeded", nil)
		return
	}

	planID := "lite"
	if targetPlanID != nil && *targetPlanID != "" {
		planID = *targetPlanID
	}

	var planName string
	var priceMonthly int64
	_ = DB.QueryRow(ctx, "SELECT name, price_monthly FROM saas_plans WHERE id = $1", planID).Scan(&planName, &priceMonthly)

	amountToCharge := calculateVoucherCharge(priceMonthly, voucherType, discountValue)
	finalValidityDays := resolveVoucherValidityDays(codeValidityDays, programDurationMonths)

	_, err = DB.Exec(ctx, `
		UPDATE voucher_codes SET is_redeemed = true, used_by = $1, used_at = NOW()
		WHERE UPPER(TRIM(code)) = UPPER(TRIM($2)) AND is_redeemed = false
	`, tenantID, cleanInput)
	if err != nil {
		slog.Warn("Failed to mark voucher redeemed", "error", err)
	}

	_, _ = DB.Exec(ctx, `UPDATE voucher_programs SET uses_count = uses_count + 1 WHERE id = $1`, programID)

	ticketID := activateSubscription(ctx, tenantID, planID, planName, finalValidityDays, "voucher", voucherActivationOpts{})

	response.JSON(w, http.StatusOK, "Voucher redeemed successfully", map[string]interface{}{
		"program_name":   programName,
		"voucher_type":   voucherType,
		"discount_value": discountValue,
		"target_plan":    planName,
		"plan_id":        planID,
		"validity_days":  finalValidityDays,
		"amount_charged": amountToCharge,
		"ticket_id":      ticketID,
	})
}
