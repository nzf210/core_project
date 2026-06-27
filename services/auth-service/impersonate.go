package main

import (
	"core_project/shared/sdk/config"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// handleImpersonate — POST /superadmin/tenants/{tenant_id}/impersonate
// Generate JWT token untuk login sebagai owner dari tenant tertentu
func handleImpersonate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Message: "Method not allowed"})
		return
	}

	// Extract superadmin user dari JWT (sudah di-validate di middleware)
	superadminID := r.Header.Get("X-User-ID")
	superadminRole := r.Header.Get("X-User-Role")
	if superadminRole != "superadmin" {
		writeJSON(w, http.StatusForbidden, Response{Message: "Superadmin access required"})
		return
	}

	// Parse tenant_id dari URL path
	tenantID := r.URL.Path[len("/superadmin/tenants/"):]
	if idx := len(tenantID) - len("/impersonate"); idx > 0 && tenantID[idx:] == "/impersonate" {
		tenantID = tenantID[:idx]
	}
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Message: "Missing tenant_id"})
		return
	}

	// Query tenant owner
	var ownerID, ownerName, businessName, plan string
	err := DB.QueryRow(r.Context(), `
		SELECT u.id, u.username, t.business_name, t.plan
		FROM tenants t
		JOIN users u ON u.tenant_id = t.id AND u.role = 'owner'
		WHERE t.id = $1
		LIMIT 1
	`, tenantID).Scan(&ownerID, &ownerName, &businessName, &plan)
	if err != nil {
		slog.Error("impersonate query failed", "tenant_id", tenantID, "error", err)
		writeJSON(w, http.StatusNotFound, Response{Message: "Tenant not found or has no owner"})
		return
	}

	// Generate JWT token — role owner, bukan superadmin
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":         ownerID,
		"tenant_id":       tenantID,
		"role":            "owner",
		"impersonated_by": superadminID, // audit trail
		"exp":             time.Now().Add(12 * time.Hour).Unix(),
	})
	// Access global config from main.go
	cfgGlobal := config.LoadConfig(".env")
	accessToken, err := token.SignedString([]byte(cfgGlobal.JWTSecret))
	if err != nil {
		slog.Error("impersonate JWT sign failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Message: "Token generation failed"})
		return
	}

	slog.Info("impersonate token issued", "superadmin", superadminID, "tenant", tenantID, "owner", ownerID)
	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Impersonate token generated",
		Data: map[string]any{
			"access_token": accessToken,
			"tenant": map[string]string{
				"id":            tenantID,
				"business_name": businessName,
				"owner_name":    ownerName,
				"plan":          plan,
			},
		},
	})
}
