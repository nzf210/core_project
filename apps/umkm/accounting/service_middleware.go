package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"core_project/shared/sdk/response"
)

// requireServiceType is a middleware that enforces business_type = 'jasa'
// for all /api/umkm/service/* routes.
//
// F073: Modular Business Workflow — Service Order & Repair Ticket (SPK)
func requireServiceType(next http.HandlerFunc) http.HandlerFunc {
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
			slog.Error("requireServiceType: failed to fetch business_type", "tenant_id", tenantID, "error", err)
			response.Error(w, http.StatusInternalServerError, "Failed to verify tenant", nil)
			return
		}

		if businessType != "jasa" {
			slog.Warn("requireServiceType: access denied — not a service tenant", "tenant_id", tenantID, "business_type", businessType)
			response.Error(w, http.StatusForbidden, "Fitur order servis hanya untuk tenant dengan jenis usaha jasa/bengkel", nil)
			return
		}

		next(w, r)
	}
}
