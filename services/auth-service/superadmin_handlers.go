package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func handleSuperAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
		return
	}

	var req SuperAdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
		return
	}

	ctx := context.Background()
	var userID, tenantID, passwordHash, role string
	var isDataVerified bool
	err := DB.QueryRow(ctx,
		"SELECT id, tenant_id, role, password_hash, is_phone_verified FROM users WHERE username = $1 AND role = 'superadmin'",
		req.Username,
	).Scan(&userID, &tenantID, &role, &passwordHash, &isDataVerified)

	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials or not a super admin"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{Success: false, Message: "Invalid credentials"})
		return
	}

	tokens, err := generateTokens(userID, tenantID, role, isDataVerified)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to generate tokens"})
		return
	}

	tokenHash := hashToken(tokens.RefreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	DB.Exec(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)", userID, tokenHash, expiresAt)
	Redis.Set(ctx, "refresh_token:"+tokenHash, userID, 7*24*time.Hour)

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Message: "Superadmin login successful",
		Data: map[string]interface{}{
			"accessToken":  tokens.AccessToken,
			"refreshToken": tokens.RefreshToken,
			"tenantId":     tenantID,
			"role":         role,
		},
	})
}

func requireSuperAdmin(r *http.Request) (*Claims, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || claims.Role != "superadmin" {
		return nil, false
	}
	return claims, true
}

func handleVerifierStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = "http://wa-gateway:8202"
	}
	verifierTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if verifierTenant == "" {
		verifierTenant = "verifier"
	}
	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		resp, err = client.Get(waURL + "/api/wa/status?tenant_id=" + verifierTenant)
		if err != nil {
			slog.Warn("WA Gateway not reachable", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// If it's a 200, we need to read body to check if it's delegated
		if resp.StatusCode == http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var tempStatus map[string]interface{}
			json.Unmarshal(bodyBytes, &tempStatus)
			if tempStatus["status"] == "delegated" {
				if i == 2 {
					tempStatus["status"] = "connected"
					newBody, _ := json.Marshal(tempStatus)
					resp.Body = io.NopCloser(bytes.NewBuffer(newBody))
					break
				}
				slog.Warn("WA Gateway status delegated, retrying...", "attempt", i+1)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			// Reconstruct body for next steps
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			break
		}
		break
	}

	if err != nil {
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status":  "unavailable",
			"message": "WA Gateway tidak berjalan. Verifier WhatsApp tidak tersedia.",
		}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("WA Gateway returned non-200", "status", resp.StatusCode)
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status":  "unavailable",
			"message": "WA Gateway tidak merespon dengan benar.",
		}})
		return
	}

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		slog.Warn("Failed to decode WA Gateway response", "error", err)
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status":  "unavailable",
			"message": "Gagal membaca status WA Gateway.",
		}})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: status})
}

func handleVerifierQR(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = "http://wa-gateway:8202"
	}
	verifierTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if verifierTenant == "" {
		verifierTenant = "verifier"
	}
	resp, err := client.Get(waURL + "/api/wa/qr?tenant_id=" + verifierTenant)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to get QR code"})
		return
	}
	defer resp.Body.Close()

	var qrData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&qrData)

	writeJSON(w, http.StatusOK, Response{Success: true, Data: qrData})
}

func handleVerifierDisconnect(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	// msisdn parameter is no longer required as we logout using tenant_id=verifier

	// Use WA Gateway's logout endpoint
	client := &http.Client{Timeout: 10 * time.Second}
	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = "http://wa-gateway:8202"
	}
	verifierTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if verifierTenant == "" {
		verifierTenant = "verifier"
	}
	req, _ := http.NewRequest(http.MethodPost, waURL+"/api/wa/logout?tenant_id="+verifierTenant, nil)
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to disconnect verifier"})
		return
	}
	defer resp.Body.Close()

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Verifier disconnected. Scan QR to reconnect."})
}

func handleSuperadminTenants(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireSuperAdmin(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		rows, err := DB.Query(ctx, `
			SELECT t.id, t.name, t.plan, t.created_at,
				COALESCE(u.username, '') as owner_username,
				COALESCE(u.phone_number, '') as owner_phone,
				(SELECT COUNT(*) FROM users WHERE tenant_id = t.id) as user_count,
				t.xendit_merchant_id
			FROM tenants t
			LEFT JOIN users u ON u.tenant_id = t.id AND u.role = 'owner'
			ORDER BY t.created_at DESC
		`)
		if err != nil {
			slog.Error("Failed to fetch tenants", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to fetch tenants"})
			return
		}
		defer rows.Close()

		var tenants []map[string]interface{}
		for rows.Next() {
			var id, name, plan, ownerUsername, ownerPhone string
			var userCount int
			var createdAt time.Time
			var xenditMerchantID *string
			if err := rows.Scan(&id, &name, &plan, &createdAt, &ownerUsername, &ownerPhone, &userCount, &xenditMerchantID); err != nil {
				continue
			}
			merchant := ""
			if xenditMerchantID != nil {
				merchant = *xenditMerchantID
			}
			tenants = append(tenants, map[string]interface{}{
				"id":                 id,
				"name":               name,
				"plan":               plan,
				"owner_username":     ownerUsername,
				"owner_phone":        ownerPhone,
				"user_count":         userCount,
				"created_at":         createdAt,
				"xendit_merchant_id": merchant,
			})
		}

		if tenants == nil {
			tenants = []map[string]interface{}{}
		}

		writeJSON(w, http.StatusOK, Response{Success: true, Data: tenants})

	case http.MethodDelete:
		tenantID := r.URL.Query().Get("id")
		if tenantID == "" {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
			return
		}

		// Prevent superadmin from deleting their own tenant
		if tenantID == claims.TenantID {
			writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin tidak diperbolehkan menghapus tenant miliknya sendiri"})
			return
		}

		tx, err := DB.Begin(ctx)
		if err != nil {
			slog.Error("Failed to begin transaction", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, "DELETE FROM journal_lines WHERE entry_id IN (SELECT id FROM journal_entries WHERE tenant_id = $1)", tenantID)
		if err != nil {
			slog.Error("Failed to delete journal lines", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		_, err = tx.Exec(ctx, "DELETE FROM journal_entries WHERE tenant_id = $1", tenantID)
		if err != nil {
			slog.Error("Failed to delete journal entries", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		_, err = tx.Exec(ctx, "DELETE FROM chart_of_accounts WHERE tenant_id = $1", tenantID)
		if err != nil {
			slog.Error("Failed to delete chart of accounts", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		tag, err := tx.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
		if err != nil {
			slog.Error("Failed to delete tenant", "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "Tenant tidak ditemukan"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("Failed to commit transaction", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to delete tenant"})
			return
		}

		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Tenant berhasil dihapus"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
	}
}

func handleSuperadminTenantProfile(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin access required"})
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		tenantID := r.URL.Query().Get("id")
		if tenantID == "" {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
			return
		}

		var id, name, plan, ownerUsername, ownerID string
		var businessName, waNumber, logoURL, businessAddress, businessType, ownerPhone, customDomain, subdomain, xenditMerchantID *string
		if err := DB.QueryRow(ctx, `
			SELECT t.id, t.name, t.plan, t.business_name, t.wa_number, t.logo_url, t.business_address, t.business_type,
			       t.custom_domain, t.subdomain, t.xendit_merchant_id,
			       COALESCE(u.username, '') as owner_username, COALESCE(u.id::text, '') as owner_id, u.phone_number as owner_phone
			FROM tenants t
			LEFT JOIN users u ON u.tenant_id = t.id AND u.role = 'owner'
			WHERE t.id = $1
		`, tenantID).Scan(&id, &name, &plan, &businessName, &waNumber, &logoURL, &businessAddress, &businessType, &customDomain, &subdomain, &xenditMerchantID, &ownerUsername, &ownerID, &ownerPhone); err != nil {
			if err == pgx.ErrNoRows {
				writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "Tenant tidak ditemukan"})
				return
			}
			slog.Error("Failed to fetch tenant profile", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Internal server error"})
			return
		}

		data := map[string]interface{}{
			"id":                 id,
			"name":               name,
			"plan":               plan,
			"business_name":      derefStr(businessName),
			"wa_number":          derefStr(waNumber),
			"owner_phone":        derefStr(ownerPhone),
			"logo_url":           derefStr(logoURL),
			"business_address":   derefStr(businessAddress),
			"business_type":      derefStr(businessType),
			"custom_domain":      derefStr(customDomain),
			"subdomain":          derefStr(subdomain),
			"xendit_merchant_id": derefStr(xenditMerchantID),
			"owner_username":     ownerUsername,
			"owner_id":           ownerID,
		}
		writeJSON(w, http.StatusOK, Response{Success: true, Data: data})

	case http.MethodPut:
		var req TenantProfileUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Invalid request payload"})
			return
		}
		slog.Info("Received update tenant profile request", "payload", req)
		if req.TenantID == "" {
			writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "tenant_id is required"})
			return
		}

		tag, err := DB.Exec(ctx, `
			UPDATE tenants SET name=$1, business_name=$2, wa_number=$3, business_address=$4, business_type=$5, plan=$6, custom_domain=NULLIF($7, ''), subdomain=NULLIF($8, ''), xendit_merchant_id=NULLIF($9, ''), updated_at=NOW()
			WHERE id=$10
		`, req.Name, req.BusinessName, req.WaNumber, req.BusinessAddress, req.BusinessType, req.Plan, req.CustomDomain, req.Subdomain, req.XenditMerchantID, req.TenantID)
		if err != nil {
			slog.Error("Failed to update tenant profile", "error", err)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to update profile"})
			return
		}
		if tag.RowsAffected() == 0 {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: "Tenant tidak ditemukan"})
			return
		}

		if req.NewPassword != "" {
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
			DB.Exec(ctx, `UPDATE users SET password_hash=$1 WHERE tenant_id=$2 AND role='owner'`, string(hash), req.TenantID)
		}

		if req.OwnerPhone != "" {
			_, err := DB.Exec(ctx, `UPDATE users SET phone_number=$1 WHERE tenant_id=$2 AND role='owner'`, req.OwnerPhone, req.TenantID)
			if err != nil {
				slog.Error("Failed to update owner phone", "error", err, "phone", req.OwnerPhone, "tenant", req.TenantID)
				writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Gagal memperbarui nomor login owner. Pastikan nomor belum digunakan."})
				return
			}
		}

		writeJSON(w, http.StatusOK, Response{Success: true, Message: "Profil tenant berhasil diperbarui"})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "Method not allowed"})
	}
}
