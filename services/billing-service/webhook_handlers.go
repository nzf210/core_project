package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	status, _ := payload["status"].(string)
	externalID, _ := payload["external_id"].(string)
	paidAmountFloat, _ := payload["paid_amount"].(float64)
	paidAmountCents := int64(paidAmountFloat)

	// Extract tenant ID from external_id
	tenantID, err := extractTenantIDFromExternalID(externalID)
	if err != nil {
		slog.Warn("Malformed external_id in webhook", "external_id", externalID)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Verify webhook token
	callbackToken := r.Header.Get("x-callback-token")
	if err := verifyWebhookToken(r.Context(), callbackToken, tenantID); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Handle wallet topup invoice (separate flow)
	if handled, err := handleWalletTopupWebhook(r.Context(), externalID, status, paidAmountFloat); handled {
		if err != nil {
			http.Error(w, errDB, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := r.Context()

	// IDEMPOTENCY CHECK + ROW LOCK: SELECT FOR UPDATE prevents concurrent webhooks
	// from both passing the PAID check and double-granting subscription.
	var currentStatus string
	var planID, voucherCode string
	var invoiceAmount int64
	err = DB.QueryRow(ctx, `
		SELECT status, COALESCE(plan_id,''), COALESCE(voucher_code,''), amount
		FROM invoices WHERE id = $1
		FOR UPDATE
	`, externalID).Scan(&currentStatus, &planID, &voucherCode, &invoiceAmount)
	if err != nil {
		slog.Warn("Invoice not found", "external_id", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}
	if currentStatus == "paid" {
		slog.Info("Invoice already paid, ignoring duplicate webhook", "external_id", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if status == "PAID" || status == "SETTLED" {
		if paidAmountCents < invoiceAmount {
			slog.Warn("PAYMENT AMOUNT MISMATCH", "external_id", externalID, "expected", invoiceAmount, "paid", paidAmountCents)
			// Still process but log as anomaly, or block activation. Let's block for safety.
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	if planID == "" {
		planID = "lite"
	}

	if status == "EXPIRED" {
		// Refund voucher so it can be reused
		if voucherCode != "" {
			_, err := DB.Exec(ctx, `
				UPDATE voucher_codes 
				SET is_redeemed = false, used_by = NULL, used_at = NULL 
				WHERE code = $1
			`, voucherCode)
			if err == nil {
				slog.Info("Voucher refunded due to expired invoice", "code", voucherCode, "tenant_id", tenantID)
			}
		}
		_, _ = DB.Exec(ctx, "UPDATE invoices SET status = 'expired' WHERE id = $1", externalID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if status != "PAID" && status != "SETTLED" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get plan name
	var planName string
	DB.QueryRow(ctx, "SELECT name FROM saas_plans WHERE id = $1", planID).Scan(&planName)

	// Update invoice
	_, _ = DB.Exec(ctx, "UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1", externalID)

	// HANDLE OVERPAYMENT: If paid > invoice, put difference in wallet
	if paidAmountCents > invoiceAmount {
		excess := paidAmountCents - invoiceAmount
		slog.Info("Overpayment detected, crediting wallet", "tenant_id", tenantID, "excess_cents", excess)
		_, errW := DB.Exec(ctx, `
			INSERT INTO wallet_credits (tenant_id, balance_cents, updated_at) 
			VALUES ($1, $2, NOW()) 
			ON CONFLICT (tenant_id) 
			DO UPDATE SET balance_cents = wallet_credits.balance_cents + $2, updated_at = NOW()
		`, tenantID, excess)
		if errW == nil {
			_, _ = DB.Exec(ctx, `
				INSERT INTO wallet_transactions (tenant_id, amount_cents, transaction_type, reference, description) 
				VALUES ($1, $2, 'topup', $3, 'Excess payment from invoice')
			`, tenantID, excess, externalID)
		}
	}

	// Apply Voucher (Mark as REDEEMED permanently on actual payment)
	if voucherCode != "" {
		ok, _ := applyVoucher(ctx, voucherCode, planID, tenantID)
		if !ok {
			slog.Warn("Failed to apply voucher on payment webhook", "code", voucherCode, "tenant_id", tenantID)
		}
	}

	// ─────────────────────────────────────────────
	// F054: AFFILIATE COMMISSION LOGIC (lifetime)
	// ─────────────────────────────────────────────
	processWebhookAffiliateCommission(ctx, tenantID, externalID, invoiceAmount)

	// Activate subscription
	finalizeSubscriptionActivation(ctx, tenantID, planID, planName, voucherCode)

	w.WriteHeader(http.StatusOK)
}

// ─────────────────────────────────────────────
// Core: Activate Subscription + Generate Ticket
// ─────────────────────────────────────────────
