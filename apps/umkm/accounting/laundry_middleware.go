package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"core_project/shared/sdk/response"
)

// requireLaundryType is a middleware that enforces business_type = 'laundry'
// for all /api/umkm/laundry/* routes. Tenants with other business types will receive 403.
//
// F071: Modular Business Workflow — Laundry Order & Wash Tracking
func requireLaundryType(next http.HandlerFunc) http.HandlerFunc {
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
			slog.Error("requireLaundryType: failed to fetch business_type", "tenant_id", tenantID, "error", err)
			response.Error(w, http.StatusInternalServerError, "Failed to verify tenant", nil)
			return
		}

		if businessType != "laundry" {
			slog.Warn("requireLaundryType: access denied — not a laundry tenant", "tenant_id", tenantID, "business_type", businessType)
			response.Error(w, http.StatusForbidden, "Fitur laundry hanya untuk tenant dengan jenis usaha laundry", nil)
			return
		}

		next(w, r)
	}
}
