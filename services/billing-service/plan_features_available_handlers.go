package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

func handleAdminAvailableFeaturesCollection(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method == http.MethodGet {
		rows, err := DB.Query(r.Context(), `
			SELECT feature_key, feature_name, description, category,
			       is_addon, default_enabled, addon_price_rupiah, addon_unit
			FROM available_features ORDER BY is_addon, feature_key
		`)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to list features", err)
			return
		}
		defer rows.Close()
		var items []map[string]interface{}
		for rows.Next() {
			var key, name, desc, cat, unit string
			var isAddon bool
			var defaultEnabled []string
			var price int64
			if rows.Scan(&key, &name, &desc, &cat, &isAddon, &defaultEnabled, &price, &unit) == nil {
				items = append(items, map[string]interface{}{
					"feature_key":         key,
					"feature_name":        name,
					"description":         desc,
					"category":            cat,
					"is_addon":            isAddon,
					"default_enabled":     defaultEnabled,
					"addon_price_rupiah":  price,
					"addon_unit":          unit,
				})
			}
		}
		response.JSON(w, http.StatusOK, "ok", items)
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			FeatureKey      string   `json:"feature_key"`
			FeatureName     string   `json:"feature_name"`
			Description     string   `json:"description"`
			Category        string   `json:"category"`
			IsAddon         bool     `json:"is_addon"`
			DefaultEnabled  []string `json:"default_enabled"`
			AddonPriceCents int64    `json:"addon_price_rupiah"`
			AddonUnit       string   `json:"addon_unit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}
		if req.FeatureKey == "" || req.FeatureName == "" || req.Category == "" {
			response.Error(w, http.StatusBadRequest, "feature_key, feature_name, category required", nil)
			return
		}
		_, err := DB.Exec(r.Context(), `
			INSERT INTO available_features (feature_key, feature_name, description, category, is_addon, default_enabled, addon_price_rupiah, addon_unit)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (feature_key) DO UPDATE SET
				feature_name=EXCLUDED.feature_name, description=EXCLUDED.description,
				category=EXCLUDED.category, is_addon=EXCLUDED.is_addon,
				default_enabled=EXCLUDED.default_enabled,
				addon_price_rupiah=EXCLUDED.addon_price_rupiah, addon_unit=EXCLUDED.addon_unit
		`, req.FeatureKey, req.FeatureName, req.Description, req.Category,
			req.IsAddon, req.DefaultEnabled, req.AddonPriceCents, req.AddonUnit)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to save feature", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), req.FeatureKey)
		response.JSON(w, http.StatusOK, "Feature saved", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}

func handleAdminAvailableFeaturesItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/admin/available-features/")
	if key == "" {
		response.Error(w, http.StatusBadRequest, "Feature key required", nil)
		return
	}
	if r.Method == http.MethodPatch {
		var req struct {
			FeatureName     *string  `json:"feature_name,omitempty"`
			Description     *string  `json:"description,omitempty"`
			Category        *string  `json:"category,omitempty"`
			IsAddon         *bool    `json:"is_addon,omitempty"`
			DefaultEnabled  []string `json:"default_enabled"`
			AddonPriceCents *int64   `json:"addon_price_rupiah,omitempty"`
			AddonUnit       *string  `json:"addon_unit,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, response.InvalidRequest, err)
			return
		}
		updates, args, idx := []string{}, []any{}, 1
		if req.FeatureName != nil {
			updates = append(updates, fmt.Sprintf("feature_name=$%d", idx))
			args = append(args, *req.FeatureName)
			idx++
		}
		if req.Description != nil {
			updates = append(updates, fmt.Sprintf("description=$%d", idx))
			args = append(args, *req.Description)
			idx++
		}
		if req.Category != nil {
			updates = append(updates, fmt.Sprintf("category=$%d", idx))
			args = append(args, *req.Category)
			idx++
		}
		if req.IsAddon != nil {
			updates = append(updates, fmt.Sprintf("is_addon=$%d", idx))
			args = append(args, *req.IsAddon)
			idx++
		}
		if req.DefaultEnabled != nil {
			updates = append(updates, fmt.Sprintf("default_enabled=$%d", idx))
			args = append(args, req.DefaultEnabled)
			idx++
		}
		if req.AddonPriceCents != nil {
			updates = append(updates, fmt.Sprintf("addon_price_rupiah=$%d", idx))
			args = append(args, *req.AddonPriceCents)
			idx++
		}
		if req.AddonUnit != nil {
			updates = append(updates, fmt.Sprintf("addon_unit=$%d", idx))
			args = append(args, *req.AddonUnit)
			idx++
		}
		if len(updates) == 0 {
			response.JSON(w, http.StatusOK, "No updates applied", nil)
			return
		}
		args = append(args, key)
		query := fmt.Sprintf("UPDATE available_features SET %s WHERE feature_key=$%d", strings.Join(updates, ", "), idx)
		_, err := DB.Exec(r.Context(), query, args...)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to update", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), key)
		response.JSON(w, http.StatusOK, "Feature updated", nil)
		return
	}
	if r.Method == http.MethodDelete {
		_, err := DB.Exec(r.Context(), "DELETE FROM available_features WHERE feature_key=$1", key)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "Failed to delete", err)
			return
		}
		auth.InvalidateFeatureDefCache(r.Context(), key)
		response.JSON(w, http.StatusOK, "Feature deleted", nil)
		return
	}
	response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
}
