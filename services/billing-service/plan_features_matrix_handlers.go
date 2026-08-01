package main

import (
	"context"
	"encoding/json"
	"net/http"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/cache"
	"core_project/shared/sdk/response"
	"github.com/jackc/pgx/v5"
)

// buildFeatureMatrix constructs the feature matrix structure from DB rows.
func buildFeatureMatrix(ctx context.Context) (map[string]planRow, map[string]map[string]map[string]interface{}, []string, []string, error) {

	rows, err := DB.Query(ctx, `
		SELECT pf.plan_id, pf.feature_key, pf.feature_name, pf.is_enabled,
		       pf.feature_value, pf.min_tier,
		       sp.name as plan_name, sp.sort_order
		FROM plan_features pf
		JOIN saas_plans sp ON sp.id = pf.plan_id
		ORDER BY sp.sort_order, pf.feature_key
	`)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	defer rows.Close()

	planMap := map[string]planRow{}
	matrix := map[string]map[string]map[string]interface{}{}
	featureOrder := []string{}
	seenFeature := map[string]bool{}

	for rows.Next() {
		var planID, key, name, value, minTier, planName string
		var enabled bool
		var sortOrder int
		if rows.Scan(&planID, &key, &name, &enabled, &value, &minTier, &planName, &sortOrder) == nil {
			if _, ok := matrix[planID]; !ok {
				matrix[planID] = map[string]map[string]interface{}{}
				planMap[planID] = planRow{ID: planID, Name: planName}
			}
			matrix[planID][key] = map[string]interface{}{
				"feature_key":   key,
				"feature_name":  name,
				"is_enabled":    enabled,
				"feature_value": value,
				"min_tier":      minTier,
			}
			if !seenFeature[key] {
				seenFeature[key] = true
				featureOrder = append(featureOrder, key)
			}
		}
	}

	// Get plan IDs in order
	planIDs := []string{}
	rows2, _ := DB.Query(ctx, "SELECT id, name FROM saas_plans WHERE is_active=true ORDER BY sort_order")
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var id, nm string
			rows2.Scan(&id, &nm)
			planIDs = append(planIDs, id)
		}
	}

	return planMap, matrix, featureOrder, planIDs, nil
}

// buildAddonList constructs addon list from DB rows.
func buildAddonList(rows pgx.Rows) []map[string]interface{} {
	var items []map[string]interface{}
	for rows.Next() {
		var key, name, unit, minTier string
		var isAddon bool
		var defaultEnabled []string
		var price int64
		if rows.Scan(&key, &name, &isAddon, &defaultEnabled, &price, &unit, &minTier) == nil {
			items = append(items, map[string]interface{}{
				"feature_key":       key,
				"feature_name":      name,
				"is_addon":          isAddon,
				"default_enabled":   defaultEnabled,
				"addon_price_cents": price,
				"addon_unit":        unit,
				"min_tier":          minTier,
			})
		}
	}
	return items
}

func handleAdminFeatureMatrix(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	if r.Method == http.MethodGet {
		planMap, matrix, featureOrder, planIDs, err := buildFeatureMatrix(r.Context())
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to load matrix", err)
			return
		}

		response.JSON(w, http.StatusOK, "ok", map[string]interface{}{
			"plans":         planMap,
			"plan_ids":      planIDs,
			"feature_order": featureOrder,
			"matrix":        matrix,
		})
		return
	}

	if r.Method == http.MethodPatch {
		var req struct {
			PlanID     string `json:"plan_id"`
			FeatureKey string `json:"feature_key"`
			IsEnabled  bool   `json:"is_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}
		if req.PlanID == "" || req.FeatureKey == "" {
			response.Error(w, http.StatusBadRequest, "plan_id and feature_key required", nil)
			return
		}
		_, err := DB.Exec(r.Context(), `
			INSERT INTO plan_features (plan_id, feature_key, feature_name, feature_value, is_enabled)
			VALUES ($1, $2, $2, 'yes', $3)
			ON CONFLICT (plan_id, feature_key) DO UPDATE SET is_enabled=EXCLUDED.is_enabled
		`, req.PlanID, req.FeatureKey, req.IsEnabled)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to toggle feature", err)
			return
		}
		if cache.Client != nil {
			cache.Client.Del(r.Context(), "plan_features:"+req.PlanID)
		}
		auth.InvalidateFeatureDefCache(r.Context(), req.FeatureKey)
		response.JSON(w, http.StatusOK, "Feature toggled", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}

func handleAdminAddonGating(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}

	if r.Method == http.MethodGet {
		rows, err := DB.Query(r.Context(), `
			SELECT af.feature_key, af.feature_name, af.is_addon,
			       af.default_enabled, af.addon_price_cents, af.addon_unit,
			       pf.min_tier
			FROM available_features af
			LEFT JOIN plan_features pf ON pf.plan_id = 'lite' AND pf.feature_key = af.feature_key
			WHERE af.is_addon = true
			ORDER BY af.feature_key
		`)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to load addon gating", err)
			return
		}
		defer rows.Close()

		items := buildAddonList(rows)
		response.JSON(w, http.StatusOK, "ok", items)
		return
	}

	if r.Method == http.MethodPatch {
		var req struct {
			FeatureKey     string   `json:"feature_key"`
			MinTier        *string  `json:"min_tier"`
			DefaultEnabled []string `json:"default_enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}
		if req.FeatureKey == "" {
			response.Error(w, http.StatusBadRequest, "feature_key required", nil)
			return
		}
		_, err := DB.Exec(r.Context(), `
			UPDATE available_features SET default_enabled=$1 WHERE feature_key=$2
		`, req.DefaultEnabled, req.FeatureKey)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update default_enabled", err)
			return
		}
		if req.MinTier != nil && *req.MinTier != "" {
			_, err = DB.Exec(r.Context(), `
				UPDATE plan_features SET min_tier=$1 WHERE feature_key=$2
			`, *req.MinTier, req.FeatureKey)
		} else {
			_, err = DB.Exec(r.Context(), `
				UPDATE plan_features SET min_tier=NULL WHERE feature_key=$1
			`, req.FeatureKey)
		}
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update min_tier", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), req.FeatureKey)
		if cache.Client != nil {
			for _, p := range []string{"lite", "pro", "ultimate"} {
				cache.Client.Del(r.Context(), "plan_features:"+p)
			}
		}
		response.JSON(w, http.StatusOK, "Addon gating updated", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}
