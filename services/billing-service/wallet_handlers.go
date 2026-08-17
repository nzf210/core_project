package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"core_project/shared/sdk/response"

	invoice "github.com/xendit/xendit-go/v6/invoice"
)

func handleAdminAddonPrices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	ctx := r.Context()
	rows, err := DB.Query(ctx, `
		SELECT feature_key, feature_name, description, category,
		       is_addon, addon_price_cents, addon_unit, default_enabled
		FROM available_features
		WHERE is_addon = true
		ORDER BY feature_key
	`)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query addon prices", err)
		return
	}
	defer rows.Close()
	type ap struct {
		Key         string   `json:"addon_key"`
		Name        string   `json:"feature_name"`
		Price       int64    `json:"price_cents"`
		Unit        string   `json:"unit"`
		Description string   `json:"description"`
		IsActive    bool     `json:"is_active"`
		DefaultEna  []string `json:"default_enabled"`
	}
	var list []ap
	for rows.Next() {
		var a ap
		var price int64
		var unit, desc string
		var defaultEna []string
		var category string
		var isAddon bool
		if rows.Scan(&a.Key, &a.Name, &desc, &category, &isAddon, &price, &unit, &defaultEna) == nil {
			a.Price = price
			a.Unit = unit
			a.Description = desc
			a.DefaultEna = defaultEna
			list = append(list, a)
		}
	}
	if err := rows.Err(); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to iterate addon prices", err)
		return
	}
	response.JSON(w, http.StatusOK, "Addon prices retrieved", list)
}

// PATCH /admin/addon-prices/{key} — update one addon price (superadmin)
func handleAdminAddonPricesItem(w http.ResponseWriter, r *http.Request) {
	role := r.Header.Get(response.XUserRole)
	if role != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return
	}
	if r.Method != http.MethodPatch {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/admin/addon-prices/")
	if key == "" || strings.Contains(key, "/") {
		response.Error(w, http.StatusBadRequest, "Missing addon_key", nil)
		return
	}
	var req struct {
		Price       *int64  `json:"price_cents"`
		Unit        *string `json:"unit"`
		IsActive    *bool   `json:"is_active"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON", nil)
		return
	}
	ctx := r.Context()
	setParts := "updated_at = NOW()"
	args := []any{}
	argIdx := 1
	if req.Price != nil {
		setParts += fmt.Sprintf(", price_cents = $%d", argIdx)
		args = append(args, *req.Price)
		argIdx++
	}
	if req.Unit != nil {
		setParts += fmt.Sprintf(", unit = $%d", argIdx)
		args = append(args, *req.Unit)
		argIdx++
	}
	if req.IsActive != nil {
		setParts += fmt.Sprintf(", is_active = $%d", argIdx)
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.Description != nil {
		setParts += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, *req.Description)
		argIdx++
	}
	args = append(args, key)
	// Safe: setParts contains only "$N" placeholders built via fmt.Sprintf above, not user input
	query := "UPDATE available_features SET " + setParts + fmt.Sprintf(" WHERE feature_key = $%d", argIdx)
	_, err := DB.Exec(ctx, query, args...)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Update failed", err)
		return
	}
	response.JSON(w, http.StatusOK, "Addon price updated", map[string]interface{}{"addon_key": key})
}

// GET /wallet — get current tenant wallet balance + transactions
func handleWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant", nil)
		return
	}
	ctx := r.Context()
	var balance int64
	err := DB.QueryRow(ctx, `SELECT balance_cents FROM wallet_credits WHERE tenant_id = $1`, tenantID).Scan(&balance)
	if err != nil {
		balance = 0
	}
	rows, err := DB.Query(ctx, `SELECT id, amount_cents, transaction_type, reference, description, created_at FROM wallet_transactions WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 20`, tenantID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to query transactions", err)
		return
	}
	defer rows.Close()
	type tx struct {
		ID     int    `json:"id"`
		Amount int64  `json:"amount_cents"`
		Type   string `json:"transaction_type"`
		Ref    string `json:"reference"`
		Desc   string `json:"description,omitempty"`
		Time   string `json:"created_at"`
	}
	var txs []tx
	for rows.Next() {
		var t tx
		var t2 time.Time
		if rows.Scan(&t.ID, &t.Amount, &t.Type, &t.Ref, &t.Desc, &t2) == nil {
			t.Time = t2.Format(time.RFC3339)
			txs = append(txs, t)
		}
	}
	if err := rows.Err(); err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to iterate transactions", err)
		return
	}
	response.JSON(w, http.StatusOK, "Wallet retrieved", map[string]interface{}{
		"balance_cents": balance,
		"transactions":  txs,
	})
}

// POST /wallet/topup — create Xendit invoice for wallet topup
func handleWalletTopup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, response.MethodNotAllowed, nil)
		return
	}
	// FIX #3: Use X-Tenant-ID (consistent with all other handlers), not X-User-Id
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		response.Error(w, http.StatusUnauthorized, "Missing tenant", nil)
		return
	}
	var req struct {
		AmountCents int64 `json:"amount_cents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AmountCents < 10000 {
		response.Error(w, http.StatusBadRequest, "Invalid amount (min Rp 10.000)", nil)
		return
	}
	desc := "Top-up Wallet Credit"
	curr := "IDR"
	ctx := r.Context()
	// FIX #4: Use UUID for external_id (unpredictable, not UnixNano)
	invoiceReq := invoice.CreateInvoiceRequest{
		ExternalId:  uuid.NewString() + keyWalletTopup + tenantID,
		Amount:      float64(req.AmountCents),
		Description: &desc,
		Currency:    &curr,
	}
	xClient, errXc := getTenantXenditClient(ctx, tenantID)
	if errXc != nil {
		slog.Error("Failed to get xendit client for tenant", "tenant_id", tenantID, "error", errXc)
		response.Error(w, http.StatusInternalServerError, "Payment provider not configured", nil)
		return
	}
	invoiceResp, _, err := xClient.InvoiceApi.CreateInvoice(context.Background()).CreateInvoiceRequest(invoiceReq).Execute()
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create invoice", err)
		return
	}
	response.JSON(w, http.StatusOK, "Topup invoice created", map[string]interface{}{
		"invoice_url": invoiceResp.InvoiceUrl,
		"external_id": invoiceResp.ExternalId,
	})
}

func cleanupPendingTenants(ctx context.Context) {
	rows, err := DB.Query(ctx, `
		SELECT t.id FROM tenants t
		JOIN tenant_subscriptions ts ON ts.tenant_id = t.id
		WHERE ts.status = 'pending' AND ts.pending_expires_at < NOW()
	`)
	if err != nil {
		slog.Error("Pending cleanup: failed to query", "error", err)
		return
	}
	defer rows.Close()

	deleted := 0
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			_, err := DB.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
			if err == nil {
				deleted++
			}
		}
	}
	if err := rows.Err(); err != nil {
		slog.Error("Pending cleanup: failed to iterate rows", "error", err)
	}
	if deleted > 0 {
		slog.Info("Pending cleanup worker ran", "deleted", deleted)
	}
}

// ─────────────────────────────────────────────
// Superadmin: Per-Tenant Quota Dashboard (F025, Task 2.8)
// GET /admin/quota/{tenant_id}
// Returns current period quota usage + plan limits for a tenant.
// Superadmin-only (caller responsible — API Gateway enforces).
// ─────────────────────────────────────────────
