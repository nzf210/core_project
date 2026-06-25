package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

type GenerateVouchersReq struct {
	PlanID        string `json:"plan_id"`
	ValidityDays  int    `json:"validity_days"`
	Quantity      int    `json:"quantity"`
	ProgramName   string `json:"program_name"`
	MaxUses       int    `json:"max_uses"`
	VoucherType   string `json:"voucher_type"`
	DiscountValue int    `json:"discount_value"`
}

func handleAdminGenerateVouchers(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req GenerateVouchersReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}
	if req.PlanID == "" || req.ValidityDays <= 0 || req.Quantity <= 0 || req.Quantity > 1000 {
		response.Error(w, http.StatusBadRequest, "plan_id, validity_days (>0), and quantity (1-1000) required", nil)
		return
	}

	ctx := r.Context()
	creatorID, _ := r.Context().Value(auth.UserIDKey).(string)

	var programID string
	programName := req.ProgramName
	if programName == "" {
		programName = "Ad-hoc Voucher - " + req.PlanID
	}

	vType := req.VoucherType
	if vType == "" {
		vType = "bonus_months"
	}

	err := DB.QueryRow(ctx, `
		INSERT INTO voucher_programs (name, voucher_type, discount_value, target_plan_id, duration_months, max_uses, is_active)
		VALUES ($1, $2, $3, $4, 0, $5, true)
		ON CONFLICT DO NOTHING
		RETURNING id
	`, programName, vType, req.DiscountValue, req.PlanID, req.MaxUses).Scan(&programID)
	if err != nil {
		DB.QueryRow(ctx, `SELECT id FROM voucher_programs WHERE name = $1`, programName).Scan(&programID)
	}

	type codeOut struct {
		Code string `json:"code"`
		Days int    `json:"validity_days"`
	}
	codes := make([]codeOut, 0, req.Quantity)

	for i := 0; i < req.Quantity; i++ {
		code := generateVoucherCode(req.PlanID, uuid.NewString()[:8])
		_, err := DB.Exec(ctx, `
			INSERT INTO voucher_codes (program_id, code, is_redeemed, validity_days, created_at)
			VALUES ($1, $2, false, $3, NOW())
		`, programID, code, req.ValidityDays)
		if err != nil {
			slog.Warn("Failed to create voucher code", "error", err)
			continue
		}
		codes = append(codes, codeOut{Code: code, Days: req.ValidityDays})
	}

	if programID != "" {
		_, _ = DB.Exec(ctx, `
			INSERT INTO voucher_generation_logs (program_id, generated_by, count, prefix)
			VALUES ($1, $2, $3, $4)
		`, programID, creatorID, len(codes), "")
	}

	slog.Info("Batch voucher codes generated", "plan", req.PlanID, "count", len(codes), "by", creatorID)

	response.JSON(w, http.StatusOK, "Voucher codes generated", map[string]interface{}{
		"plan_id":       req.PlanID,
		"validity_days": req.ValidityDays,
		"count":         len(codes),
		"codes":         codes,
	})
}

func handleAdminVouchers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleAdminListVouchers(w, r)
	case http.MethodDelete:
		handleAdminDeleteVoucher(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func handleAdminListVouchers(w http.ResponseWriter, r *http.Request) {
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
	planID := r.URL.Query().Get("plan_id")
	usedStr := r.URL.Query().Get("used")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
		limit = l
	}

	query := `
		SELECT vc.id, vc.code, vc.program_id, COALESCE(vp.name, ''),
		       vc.is_redeemed, COALESCE(vc.used_by::text, ''), vc.used_at, vc.created_at,
		       COALESCE(vp.target_plan_id, ''), COALESCE(vp.is_active, false)
		FROM voucher_codes vc
		LEFT JOIN voucher_programs vp ON vp.id = vc.program_id
		WHERE ($1 = '' OR vp.target_plan_id = $1)
		  AND ($2 = '' OR ($2 = 'true' AND vc.is_redeemed = true) OR ($2 = 'false' AND vc.is_redeemed = false))
		ORDER BY vc.created_at DESC
		LIMIT $3
	`

	rows, err := DB.Query(ctx, query, planID, usedStr, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list vouchers", err)
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var id, code, progID, progName, usedBy string
		var isRedeemed bool
		var usedAt *time.Time
		var createdAt time.Time
		var targetPlan string
		var progActive bool
		if rows.Scan(&id, &code, &progID, &progName, &isRedeemed, &usedBy, &usedAt, &createdAt, &targetPlan, &progActive) == nil {
			entry := map[string]interface{}{
				"id":             id,
				"code":           code,
				"program_id":     progID,
				"program_name":   progName,
				"is_redeemed":    isRedeemed,
				"used_by":        usedBy,
				"used_at":        formatTime(usedAt),
				"created_at":     createdAt.Format(time.RFC3339),
				"target_plan":    targetPlan,
				"program_active": progActive,
			}
			out = append(out, entry)
		}
	}

	var total, used, unused int
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_codes`).Scan(&total)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_codes WHERE is_redeemed = true`).Scan(&used)
	DB.QueryRow(ctx, `SELECT COUNT(*) FROM voucher_codes WHERE is_redeemed = false`).Scan(&unused)

	response.JSON(w, http.StatusOK, "Vouchers retrieved", map[string]interface{}{
		"total":  total,
		"used":   used,
		"unused": unused,
		"codes":  out,
	})
}

func handleAdminDeleteVoucher(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	voucherID := r.URL.Query().Get("id")
	if voucherID == "" {
		response.Error(w, http.StatusBadRequest, "Missing voucher id", nil)
		return
	}

	ctx := r.Context()

	var isRedeemed bool
	err := DB.QueryRow(ctx, `SELECT is_redeemed FROM voucher_codes WHERE id = $1`, voucherID).Scan(&isRedeemed)
	if err != nil {
		response.Error(w, http.StatusNotFound, "Voucher not found", err)
		return
	}
	if isRedeemed {
		response.Error(w, http.StatusBadRequest, "Cannot delete a voucher that has already been used", nil)
		return
	}

	_, err = DB.Exec(ctx, `DELETE FROM voucher_codes WHERE id = $1 AND is_redeemed = false`, voucherID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete voucher", err)
		return
	}

	response.JSON(w, http.StatusOK, "Voucher deleted successfully", nil)
}
