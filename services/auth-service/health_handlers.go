package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Auth service is healthy"})
}

// handleMe returns a lightweight, GET-only summary of the current user + tenant.
// Designed for the frontend router guard to re-sync state (onboarding_completed,
// plan, role, is_frozen) from the backend on every page reload — fixes the
// onboarding redirect loop when localStorage flags are missing (e.g. new device).
func handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	claims, ok := requireAuth(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Authentication required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var role string
	var plan, businessType *string
	var isFrozen, onboardingCompleted bool
	var userID, tenantID, email, username, phoneNumber string
	var telegramChatID *string

	err := DB.QueryRow(ctx, `
		SELECT u.id, u.username, COALESCE(u.email, ''), u.phone_number, u.role, u.telegram_chat_id,
		       t.id, t.plan, t.business_type, COALESCE(t.is_frozen, false), COALESCE(t.onboarding_completed, false)
		FROM users u
		JOIN tenants t ON t.id = u.tenant_id
		WHERE u.id = $1 AND u.tenant_id = $2
	`, claims.UserID, claims.TenantID).Scan(
		&userID, &username, &email, &phoneNumber, &role, &telegramChatID,
		&tenantID, &plan, &businessType, &isFrozen, &onboardingCompleted,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "User not found"})
			return
		}
		slog.Error("handleMe query failed", "error", err, "user_id", claims.UserID)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	// Check tenant addons (wallet_credits balance > 0 for wa_session_meta)
	var addons []string
	rows, err := DB.Query(ctx, `
		SELECT DISTINCT reference FROM wallet_transactions
		WHERE tenant_id = $1 AND reference LIKE 'wa_session_meta%' AND amount_rupiah < 0
		LIMIT 1
	`, tenantID)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			addons = append(addons, "wa_session_meta")
		}
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]any{
			"user_id":              userID,
			"username":             username,
			"email":                email,
			"phone_number":         phoneNumber,
			"role":                 role,
			"telegram_chat_id":     derefStr(telegramChatID),
			"tenant_id":            tenantID,
			"plan":                 derefStr(plan),
			"business_type":        derefStr(businessType),
			"is_frozen":            isFrozen,
			"onboarding_completed": onboardingCompleted,
			"addons":               addons,
		},
	})
}

func handleTenantResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: msgMethodNotAllowed})
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Domain is required"})
		return
	}

	ctx := context.Background()
	var tenantID, businessName string
	var logoURL *string

	err := DB.QueryRow(ctx,
		"SELECT id, business_name, logo_url FROM tenants WHERE custom_domain = $1 OR subdomain = $1",
		domain,
	).Scan(&tenantID, &businessName, &logoURL)

	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusOK, Response{Success: false, Message: "Tenant not found"})
		return
	} else if err != nil {
		slog.Error("Failed to resolve tenant", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	logo := ""
	if logoURL != nil {
		logo = *logoURL
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"tenant_id":     tenantID,
			"business_name": businessName,
			"logo_url":      logo,
		},
	})
}
