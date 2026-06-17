package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"core_project/shared/sdk/response"
)

// requireClinicType is a middleware that enforces business_type = 'clinic'
// for all /clinic/* routes. Tenants with other business types will receive 403.
//
// F047: Business Type-Based Module System
func requireClinicType(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := getTenantID(r)
		if tenantID == "" {
			response.Error(w, http.StatusUnauthorized, "Missing tenant ID", nil)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var businessType string
		err := DB.QueryRow(ctx, `SELECT business_type FROM tenants WHERE id = $1`, tenantID).Scan(&businessType)
		if err != nil {
			slog.Error("requireClinicType: failed to fetch business_type", "tenant_id", tenantID, "error", err)
			response.Error(w, http.StatusInternalServerError, "Failed to verify tenant", nil)
			return
		}

		if businessType != "clinic" {
			slog.Warn("requireClinicType: access denied — not a clinic tenant", "tenant_id", tenantID, "business_type", businessType)
			response.Error(w, http.StatusForbidden, "Fitur klinik hanya untuk tenant dengan jenis usaha klinik", nil)
			return
		}

		next(w, r)
	}
}
