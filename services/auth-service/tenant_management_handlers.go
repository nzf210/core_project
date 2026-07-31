package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	errTenantDeleteFailed = "Failed to delete tenant"
	errTenantNotFound     = "Tenant tidak ditemukan"
)

func handleSuperadminTenants(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireSuperAdmin(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: errSuperadminRequired})
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		superadminListTenants(w, ctx)
	case http.MethodDelete:
		superadminDeleteTenant(w, r, ctx, claims)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: errMethodNotAllowed})
	}
}

func superadminListTenants(w http.ResponseWriter, ctx context.Context) {
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
}

func superadminDeleteTenant(w http.ResponseWriter, r *http.Request, ctx context.Context, claims *Claims) {
	tenantID := r.URL.Query().Get("id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
		return
	}

	if tenantID == claims.TenantID {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: "Superadmin tidak diperbolehkan menghapus tenant miliknya sendiri"})
		return
	}

	tx, err := DB.Begin(ctx)
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: errTenantDeleteFailed})
		return
	}
	defer tx.Rollback(ctx)

	tables := []string{
		"journal_lines WHERE entry_id IN (SELECT id FROM journal_entries WHERE tenant_id = $1)",
		"journal_entries WHERE tenant_id = $1",
		"chart_of_accounts WHERE tenant_id = $1",
	}
	for _, t := range tables {
		if _, err := tx.Exec(ctx, "DELETE FROM "+t, tenantID); err != nil {
			slog.Error("Failed to delete from "+t, "error", err, "tenant_id", tenantID)
			writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: errTenantDeleteFailed})
			return
		}
	}

	tag, err := tx.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenantID)
	if err != nil {
		slog.Error(errTenantDeleteFailed, "error", err, "tenant_id", tenantID)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: errTenantDeleteFailed})
		return
	}
	if tag.RowsAffected() == 0 {
		writeJSON(w, http.StatusNotFound, Response{Success: false, Message: errTenantNotFound})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: errTenantDeleteFailed})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Tenant berhasil dihapus"})
}

func handleSuperadminTenantProfile(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: errSuperadminRequired})
		return
	}

	ctx := context.Background()

	switch r.Method {
	case http.MethodGet:
		superadminGetTenantProfile(w, r, ctx)
	case http.MethodPut:
		superadminUpdateTenantProfile(w, r, ctx)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: errMethodNotAllowed})
	}
}

func superadminGetTenantProfile(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	tenantID := r.URL.Query().Get("id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "Parameter id tenant diperlukan"})
		return
	}

	var id, name, plan, ownerUsername, ownerID, ownerEmail string
	var businessName, waNumber, logoURL, businessAddress, businessType, ownerPhone, customDomain, subdomain, xenditMerchantID *string
	if err := DB.QueryRow(ctx, `
		SELECT t.id, t.name, t.plan, t.business_name, t.wa_number, t.logo_url, t.business_address, t.business_type,
		       t.custom_domain, t.subdomain, t.xendit_merchant_id,
		       COALESCE(u.username, '') as owner_username, COALESCE(u.id::text, '') as owner_id,
		       u.phone_number as owner_phone, COALESCE(u.email, '') as owner_email
		FROM tenants t
		LEFT JOIN users u ON u.tenant_id = t.id AND u.role = 'owner'
		WHERE t.id = $1
	`, tenantID).Scan(&id, &name, &plan, &businessName, &waNumber, &logoURL, &businessAddress, &businessType, &customDomain, &subdomain, &xenditMerchantID, &ownerUsername, &ownerID, &ownerPhone, &ownerEmail); err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, Response{Success: false, Message: errTenantNotFound})
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
		"owner_email":        ownerEmail,
	}
	writeJSON(w, http.StatusOK, Response{Success: true, Data: data})
}

func superadminUpdateTenantProfile(w http.ResponseWriter, r *http.Request, ctx context.Context) {
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
		writeJSON(w, http.StatusNotFound, Response{Success: false, Message: errTenantNotFound})
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
}
