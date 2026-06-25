package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
)

func handleAdminPlanFeaturesCollection(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listPlanFeatures(w, r)
	case http.MethodPost:
		createPlanFeature(w, r)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func handleAdminPlanFeaturesItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/admin/plan-features/")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Feature ID required", nil)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		updatePlanFeature(w, r, id)
	case http.MethodDelete:
		deletePlanFeature(w, r, id)
	default:
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
	}
}

func handleAdminPlanFeaturesMatrix(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	planID := strings.TrimPrefix(r.URL.Path, "/admin/plan-features-matrix/")
	if planID == "" {
		response.Error(w, http.StatusBadRequest, "Plan ID required", nil)
		return
	}

	// GET — return current numeric limits for this plan
	if r.Method == http.MethodGet {
		row := DB.QueryRow(r.Context(), `
			SELECT COALESCE(MAX(max_users), 0), COALESCE(MAX(max_transactions), 0), COALESCE(MAX(max_ai_text), 0), COALESCE(MAX(max_ai_vision), 0),
				   COALESCE(MAX(max_ai_audio_minutes), 0), COALESCE(MAX(max_image_gen), 0), COALESCE(MAX(max_products), 0), COALESCE(MAX(max_customers), 0),
				   COALESCE(MAX(max_storage_mb), 0), COALESCE(MAX(api_rate_limit_per_min), 0), COALESCE(MAX(data_retention_months), 0)
			FROM plan_features WHERE plan_id = $1
		`, planID)
		var m struct {
			MaxUsers            int `json:"max_users"`
			MaxTransactions     int `json:"max_transactions"`
			MaxAIText           int `json:"max_ai_text"`
			MaxAIVision         int `json:"max_ai_vision"`
			MaxAIAudioMinutes   int `json:"max_ai_audio_minutes"`
			MaxImageGen         int `json:"max_image_gen"`
			MaxProducts         int `json:"max_products"`
			MaxCustomers        int `json:"max_customers"`
			MaxStorageMB        int `json:"max_storage_mb"`
			APIRateLimitPerMin  int `json:"api_rate_limit_per_min"`
			DataRetentionMonths int `json:"data_retention_months"`
		}
		if err := row.Scan(&m.MaxUsers, &m.MaxTransactions, &m.MaxAIText, &m.MaxAIVision,
			&m.MaxAIAudioMinutes, &m.MaxImageGen, &m.MaxProducts, &m.MaxCustomers,
			&m.MaxStorageMB, &m.APIRateLimitPerMin, &m.DataRetentionMonths); err != nil {
			response.Error(w, http.StatusNotFound, "Plan not found", err)
			return
		}
		response.JSON(w, http.StatusOK, "ok", m)
		return
	}

	// PATCH — update numeric limits (columns are in saas_plans table)
	if r.Method != http.MethodPatch {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}

	var req map[string]int
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	updates := []string{}
	args := []any{}
	idx := 1

	// We map exact columns allowed to be updated to prevent SQL injection
	allowedColumns := map[string]bool{
		"max_users": true, "max_transactions": true, "max_ai_text": true,
		"max_ai_vision": true, "max_ai_audio_minutes": true, "max_image_gen": true,
		"max_products": true, "max_customers": true, "max_storage_mb": true,
		"api_rate_limit_per_min": true, "data_retention_months": true,
	}

	for key, val := range req {
		if allowedColumns[key] {
			updates = append(updates, fmt.Sprintf("%s = $%d", key, idx))
			args = append(args, val)
			idx++
		}
	}

	if len(updates) == 0 {
		response.JSON(w, http.StatusOK, "No updates applied", nil)
		return
	}

	args = append(args, planID)
	query := fmt.Sprintf("UPDATE plan_features SET %s WHERE plan_id = $%d", strings.Join(updates, ", "), idx)
	if _, err := DB.Exec(r.Context(), query, args...); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update matrix", err)
		return
	}
	// Invalidate the cache across all services so the new limits take effect immediately
	if cache.Client != nil {
		cache.Client.Del(r.Context(), "plan_features:"+planID)
	}
	response.JSON(w, http.StatusOK, "Matrix updated", nil)
}

func listPlanFeatures(w http.ResponseWriter, r *http.Request) {
	planID := r.URL.Query().Get("plan_id")
	var (
		rows interface {
			Next() bool
			Close()
			Scan(...any) error
		}
		err error
	)

	if planID != "" {
		rows, err = DB.Query(r.Context(), `
			SELECT id, plan_id, feature_key, feature_name, feature_value, is_enabled, created_at
			FROM plan_features WHERE plan_id = $1 ORDER BY feature_key ASC
		`, planID)
	} else {
		rows, err = DB.Query(r.Context(), `
			SELECT id, plan_id, feature_key, feature_name, feature_value, is_enabled, created_at
			FROM plan_features ORDER BY plan_id ASC, feature_key ASC
		`)
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to list features", err)
		return
	}
	defer rows.Close()

	features := []map[string]interface{}{}
	for rows.Next() {
		var id, planID, key, name, value string
		var enabled bool
		var createdAt time.Time
		if rows.Scan(&id, &planID, &key, &name, &value, &enabled, &createdAt) == nil {
			features = append(features, map[string]interface{}{
				"id":            id,
				"plan_id":       planID,
				"feature_key":   key,
				"feature_name":  name,
				"feature_value": value,
				"is_enabled":    enabled,
				"created_at":    createdAt.Format(time.RFC3339),
			})
		}
	}

	response.JSON(w, http.StatusOK, "Features retrieved", features)
}

func createPlanFeature(w http.ResponseWriter, r *http.Request) {
	var req PlanFeatureReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}
	if req.PlanID == "" || req.FeatureKey == "" || req.FeatureName == "" {
		response.Error(w, http.StatusBadRequest, "plan_id, feature_key, and feature_name are required", nil)
		return
	}

	var planExists bool
	if err := DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM saas_plans WHERE id=$1)", req.PlanID).Scan(&planExists); err != nil || !planExists {
		response.Error(w, http.StatusBadRequest, "Invalid plan_id", nil)
		return
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}

	var newID string
	err := DB.QueryRow(r.Context(), `
		INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (plan_id, feature_key) DO UPDATE SET
			feature_name = EXCLUDED.feature_name,
			feature_value = EXCLUDED.feature_value,
			is_enabled = EXCLUDED.is_enabled
		RETURNING id
	`, req.PlanID, req.FeatureKey, req.FeatureName, req.FeatureValue, enabled).Scan(&newID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create/update feature", err)
		return
	}

	slog.Info("Plan feature upserted", "plan_id", req.PlanID, "feature_key", req.FeatureKey, "value", req.FeatureValue)

	response.JSON(w, http.StatusOK, "Feature saved", map[string]interface{}{
		"id":            newID,
		"plan_id":       req.PlanID,
		"feature_key":   req.FeatureKey,
		"feature_name":  req.FeatureName,
		"feature_value": req.FeatureValue,
		"is_enabled":    enabled,
	})
}

func updatePlanFeature(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		FeatureName  *string `json:"feature_name,omitempty"`
		FeatureValue *string `json:"feature_value,omitempty"`
		IsEnabled    *bool   `json:"is_enabled,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
		return
	}

	updates := []string{}
	args := []any{}
	idx := 1
	if req.FeatureName != nil {
		updates = append(updates, fmt.Sprintf("feature_name = $%d", idx))
		args = append(args, *req.FeatureName)
		idx++
	}
	if req.FeatureValue != nil {
		updates = append(updates, fmt.Sprintf("feature_value = $%d", idx))
		args = append(args, *req.FeatureValue)
		idx++
	}
	if req.IsEnabled != nil {
		updates = append(updates, fmt.Sprintf("is_enabled = $%d", idx))
		args = append(args, *req.IsEnabled)
		idx++
	}

	if len(updates) == 0 {
		response.Error(w, http.StatusBadRequest, "No fields to update", nil)
		return
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE plan_features SET %s WHERE id = $%d", strings.Join(updates, ", "), idx)
	res, err := DB.Exec(r.Context(), query, args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update feature", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Feature not found", nil)
		return
	}

	slog.Info("Plan feature updated", "id", id)
	response.JSON(w, http.StatusOK, "Feature updated", nil)
}

func deletePlanFeature(w http.ResponseWriter, r *http.Request, id string) {
	res, err := DB.Exec(r.Context(), "DELETE FROM plan_features WHERE id = $1", id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete feature", err)
		return
	}
	if res.RowsAffected() == 0 {
		response.Error(w, http.StatusNotFound, "Feature not found", nil)
		return
	}
	slog.Info("Plan feature deleted", "id", id)
	response.JSON(w, http.StatusOK, "Feature deleted", nil)
}

