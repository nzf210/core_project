package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"core_project/shared/sdk/response"
)

func handleAdminListPlans(w http.ResponseWriter, r *http.Request) {
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
	rows, err := DB.Query(ctx, `
		SELECT id, name, description, price_monthly, price_yearly, is_active, sort_order
		FROM saas_plans ORDER BY sort_order ASC
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch plans", err)
		return
	}
	defer rows.Close()

	var plans []planRow
	for rows.Next() {
		var p planRow
		if rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceMonthly, &p.PriceYearly, &p.IsActive, &p.SortOrder) == nil {
			plans = append(plans, p)
		}
	}

	response.JSON(w, http.StatusOK, "Plans retrieved", plans)
}

// ─────────────────────────────────────────────
// Superadmin: Update Plan Price
// ─────────────────────────────────────────────

type UpdatePlanReq struct {
	PriceMonthly int64 `json:"price_monthly"`
	PriceYearly  int64 `json:"price_yearly"`
	IsActive     *bool `json:"is_active"`
	SortOrder    *int  `json:"sort_order"`
}

func handleAdminUpdatePlan(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	planID := strings.TrimPrefix(r.URL.Path, "/admin/plans/")
	if planID == "" {
		response.Error(w, http.StatusBadRequest, "Missing plan ID", nil)
		return
	}

	var req UpdatePlanReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, nil)
		return
	}

	// Validate: price must be non-negative (0 = free plan)
	if req.PriceMonthly < 0 || req.PriceYearly < 0 {
		response.Error(w, http.StatusBadRequest, "Price cannot be negative", nil)
		return
	}

	ctx := r.Context()

	// Verify plan exists
	var existingID string
	if err := DB.QueryRow(ctx, "SELECT id FROM saas_plans WHERE id = $1", planID).Scan(&existingID); err != nil {
		response.Error(w, http.StatusNotFound, "Plan not found", nil)
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []any{}
	argIdx := 1

	updates = append(updates, fmt.Sprintf("price_monthly = $%d", argIdx))
	args = append(args, req.PriceMonthly)
	argIdx++

	updates = append(updates, fmt.Sprintf("price_yearly = $%d", argIdx))
	args = append(args, req.PriceYearly)
	argIdx++

	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}

	if req.SortOrder != nil {
		updates = append(updates, fmt.Sprintf("sort_order = $%d", argIdx))
		args = append(args, *req.SortOrder)
		argIdx++
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, planID)

	query := fmt.Sprintf("UPDATE saas_plans SET %s WHERE id = $%d", strings.Join(updates, ", "), argIdx)

	_, err := DB.Exec(ctx, query, args...)
	if err != nil {
		slog.Error("Failed to update plan", "plan_id", planID, "error", err)
		response.Error(w, http.StatusInternalServerError, "Failed to update plan", err)
		return
	}

	slog.Info("Plan updated by superadmin", "plan_id", planID, "price_monthly", req.PriceMonthly, "price_yearly", req.PriceYearly)

	// Fetch updated plan
	var updated planRow
	DB.QueryRow(ctx, `
		SELECT id, name, description, price_monthly, price_yearly, is_active, sort_order
		FROM saas_plans WHERE id = $1
	`, planID).Scan(&updated.ID, &updated.Name, &updated.Description, &updated.PriceMonthly, &updated.PriceYearly, &updated.IsActive, &updated.SortOrder)

	response.JSON(w, http.StatusOK, "Plan updated", updated)
}

// ─────────────────────────────────────────────
// Superadmin: Plan Features CRUD (Dynamic per tier)
// Superadmin bisa add/edit/toggle feature per plan kapan saja
// tanpa harus tulis migration baru.
// ─────────────────────────────────────────────

type PlanFeatureReq struct {
	PlanID       string `json:"plan_id"`
	FeatureKey   string `json:"feature_key"`
	FeatureName  string `json:"feature_name"`
	FeatureValue string `json:"feature_value"`
	IsEnabled    *bool  `json:"is_enabled"`
}
