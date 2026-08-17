package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

func handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := parseWebhookPayload(r)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	tenantID, err := extractTenantIDFromExternalID(payload.ExternalID)
	if err != nil {
		slog.Warn("Malformed external_id in webhook", "external_id", payload.ExternalID)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := verifyWebhookToken(r.Context(), r.Header.Get("x-callback-token"), tenantID); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if handled, err := handleWalletTopupWebhook(r.Context(), payload.ExternalID, payload.Status, payload.PaidAmountFloat); handled {
		if err != nil {
			http.Error(w, errDB, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	processSubscriptionWebhook(r.Context(), w, tenantID, payload)
}

type webhookPayload struct {
	Status          string
	ExternalID      string
	PaidAmountFloat float64
	PaidAmountCents int64
}

func parseWebhookPayload(r *http.Request) (*webhookPayload, error) {
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	status, _ := raw["status"].(string)
	externalID, _ := raw["external_id"].(string)
	paidAmountFloat, _ := raw["paid_amount"].(float64)

	return &webhookPayload{
		Status:          status,
		ExternalID:      externalID,
		PaidAmountFloat: paidAmountFloat,
		PaidAmountCents: int64(paidAmountFloat),
	}, nil
}

func processSubscriptionWebhook(ctx context.Context, w http.ResponseWriter, tenantID string, payload *webhookPayload) {
	invoice, err := lockAndLoadInvoice(ctx, payload.ExternalID)
	if err != nil {
		slog.Warn("Invoice not found", "external_id", payload.ExternalID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if invoice.Status == "paid" {
		slog.Info("Invoice already paid, ignoring duplicate webhook", "external_id", payload.ExternalID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if payload.Status == "EXPIRED" {
		handleExpiredInvoice(ctx, invoice, tenantID)
		w.WriteHeader(http.StatusOK)
		return
	}

	if !isPaymentSuccess(payload.Status) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if payload.PaidAmountCents < invoice.Amount {
		slog.Warn("PAYMENT AMOUNT MISMATCH", "external_id", payload.ExternalID, "expected", invoice.Amount, "paid", payload.PaidAmountCents)
		w.WriteHeader(http.StatusOK)
		return
	}

	processSuccessfulPayment(ctx, tenantID, payload, invoice)
	w.WriteHeader(http.StatusOK)
}

type invoiceData struct {
	Status      string
	PlanID      string
	VoucherCode string
	Amount      int64
}

func lockAndLoadInvoice(ctx context.Context, externalID string) (*invoiceData, error) {
	var inv invoiceData
	err := DB.QueryRow(ctx, `
		SELECT status, COALESCE(plan_id,''), COALESCE(voucher_code,''), amount
		FROM invoices WHERE id = $1 FOR UPDATE
	`, externalID).Scan(&inv.Status, &inv.PlanID, &inv.VoucherCode, &inv.Amount)

	if err != nil {
		return nil, err
	}
	if inv.PlanID == "" {
		inv.PlanID = "lite"
	}
	return &inv, nil
}

func handleExpiredInvoice(ctx context.Context, invoice *invoiceData, tenantID string) {
	if invoice.VoucherCode != "" {
		_, err := DB.Exec(ctx, `
			UPDATE voucher_codes
			SET is_redeemed = false, used_by = NULL, used_at = NULL
			WHERE code = $1
		`, invoice.VoucherCode)
		if err == nil {
			slog.Info("Voucher refunded due to expired invoice", "code", invoice.VoucherCode, "tenant_id", tenantID)
		}
	}
	DB.Exec(ctx, "UPDATE invoices SET status = 'expired' WHERE id = $1", invoice.VoucherCode)
}

func isPaymentSuccess(status string) bool {
	return status == "PAID" || status == "SETTLED"
}

func processSuccessfulPayment(ctx context.Context, tenantID string, payload *webhookPayload, invoice *invoiceData) {
	var planName string
	DB.QueryRow(ctx, "SELECT name FROM saas_plans WHERE id = $1", invoice.PlanID).Scan(&planName)

	DB.Exec(ctx, "UPDATE invoices SET status = 'paid', paid_at = NOW() WHERE id = $1", payload.ExternalID)

	handleOverpayment(ctx, tenantID, payload.ExternalID, payload.PaidAmountCents, invoice.Amount)

	if invoice.VoucherCode != "" {
		if ok, _ := applyVoucher(ctx, invoice.VoucherCode, invoice.PlanID, tenantID); !ok {
			slog.Warn("Failed to apply voucher on payment webhook", "code", invoice.VoucherCode, "tenant_id", tenantID)
		}
	}

	processWebhookAffiliateCommission(ctx, tenantID, payload.ExternalID, invoice.Amount)
	finalizeSubscriptionActivation(ctx, tenantID, invoice.PlanID, planName, invoice.VoucherCode)
}

func handleOverpayment(ctx context.Context, tenantID, externalID string, paidAmount, invoiceAmount int64) {
	if paidAmount <= invoiceAmount {
		return
	}

	excess := paidAmount - invoiceAmount
	slog.Info("Overpayment detected, crediting wallet", "tenant_id", tenantID, "excess_cents", excess)

	_, errW := DB.Exec(ctx, `
		INSERT INTO wallet_credits (tenant_id, balance_cents, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET balance_cents = wallet_credits.balance_cents + $2, updated_at = NOW()
	`, tenantID, excess)

	if errW == nil {
		DB.Exec(ctx, `
			INSERT INTO wallet_transactions (tenant_id, amount_cents, transaction_type, reference, description)
			VALUES ($1, $2, 'topup', $3, 'Excess payment from invoice')
		`, tenantID, excess, externalID)
	}
}

// ─────────────────────────────────────────────
// Core: Activate Subscription + Generate Ticket
// ─────────────────────────────────────────────
