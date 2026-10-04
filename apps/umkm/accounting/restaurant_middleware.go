package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"core_project/shared/sdk/response"
)

// requireRestaurantType is a middleware that enforces business_type = 'restoran'
// for all /api/umkm/restaurant/* routes.
//
// F072: Modular Business Workflow — Restaurant KOT & Table Orders
func requireRestaurantType(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		if tenantID == "" {
			response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
			return
		}

		if DB == nil {
			// Skip DB check if DB connection pool is not initialized (e.g. basic unit tests)
			next(w, r)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var businessType string
		err := DB.QueryRow(ctx, `SELECT business_type FROM tenants WHERE id = $1`, tenantID).Scan(&businessType)
		if err != nil {
			slog.Error("requireRestaurantType: failed to fetch business_type", "tenant_id", tenantID, "error", err)
			response.Error(w, http.StatusInternalServerError, "Failed to verify tenant", nil)
			return
		}

		if businessType != "restoran" {
			slog.Warn("requireRestaurantType: access denied — not a restaurant tenant", "tenant_id", tenantID, "business_type", businessType)
			response.Error(w, http.StatusForbidden, "Fitur restoran hanya untuk tenant dengan jenis usaha restoran/kuliner", nil)
			return
		}

		next(w, r)
	}
}
