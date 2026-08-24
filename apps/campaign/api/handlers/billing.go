package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"core_project/apps/campaign/api/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
	"core_project/shared/sdk/response"
)

const (
	errInvalidJSON              = "Invalid JSON"
	errCommitCampaignBillingTx  = "Failed to commit campaign billing tx"
)

// xenditClientCache — per-tenant Xendit client, 5-min TTL
var (
	xenditMu      sync.RWMutex
	xenditClients = make(map[string]*xenditClientEntry)
)

type xenditClientEntry struct {
	client    *xendit.APIClient
	createdAt time.Time
}

func getTenantXenditClient(ctx context.Context, tenantID string) (*xendit.APIClient, error) {
	xenditMu.RLock()
	entry, ok := xenditClients[tenantID]
	xenditMu.RUnlock()
	if ok && time.Since(entry.createdAt) < 5*time.Minute {
		return entry.client, nil
	}

	var apiKey string
	err := repository.DB.QueryRow(ctx,
		"SELECT xendit_api_key FROM tenants WHERE id = $1", tenantID,
	).Scan(&apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get xendit_api_key for tenant %s: %w", tenantID, err)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("tenant %s has no xendit_api_key configured", tenantID)
	}

	client := xendit.NewClient(apiKey)
	xenditMu.Lock()
	xenditClients[tenantID] = &xenditClientEntry{client: client, createdAt: time.Now()}
	xenditMu.Unlock()
	return client, nil
}

// splitExternalID parses "prefix|tenantID" format from external_id
func splitExternalID(externalID string) []string {
	return strings.Split(externalID, "|")
}

func getFrontendURL() string {
	if AppConfig.Env == "production" {
		return "https://wch.id"
	}
	return "http://localhost:3201"
}

type checkoutRequest struct {
	CampaignID string `json:"campaign_id"`
	OrderType  string `json:"order_type"`
	Quantity   int    `json:"quantity"`
}

type checkoutResult struct {
	paymentURL string
	invoiceID  string
}

func HandleBillingCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant ID"})
		return
	}

	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: errInvalidJSON})
		return
	}

	priceRupiah, ok := calculatePrice(req.OrderType, req.Quantity)
	if !ok {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid order type"})
		return
	}

	ctx := context.Background()
	if !validateCampaignAccess(ctx, req.CampaignID, tenantID) {
		WriteJSON(w, http.StatusForbidden, APIResponse{Message: "Campaign not found or not owned by tenant"})
		return
	}

	externalID := "camp_" + uuid.NewString()[:20]
	referralDiscount := getReferralDiscount(ctx, tenantID, priceRupiah)
	finalPrice := priceRupiah - referralDiscount.amount

	result, err := createXenditInvoice(ctx, invoiceParams{
		tenantID:         tenantID,
		externalID:       externalID,
		campaignID:       req.CampaignID,
		orderType:        req.OrderType,
		quantity:         req.Quantity,
		priceRupiah:      priceRupiah,
		referralDiscount: referralDiscount.amount,
		finalPrice:       finalPrice,
	})
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: err.Error()})
		return
	}

	if referralDiscount.affiliateID != nil && referralDiscount.amount > 0 {
		_ = saveReferralDiscount(ctx, result.invoiceID, *referralDiscount.affiliateID, referralDiscount.amount)
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"invoice_url": result.paymentURL, "invoice_id": result.invoiceID},
	})
}

func calculatePrice(orderType string, quantity int) (int64, bool) {
	switch orderType {
	case "wargame_token":
		return 100_000 * int64(quantity), true
	case "intelligence_pack":
		return 5_000_000 * int64(quantity), true
	default:
		return 0, false
	}
}

type referralInfo struct {
	affiliateID *int
	amount      int64
}

func getReferralDiscount(ctx context.Context, tenantID string, priceRupiah int64) referralInfo {
	var referredByID *int
	if err := repository.DB.QueryRow(ctx,
		"SELECT referred_by_affiliate_id FROM tenants WHERE id = $1", tenantID,
	).Scan(&referredByID); err != nil || referredByID == nil {
		return referralInfo{}
	}

	var discountPct float64
	repository.DB.QueryRow(ctx, "SELECT COALESCE(discount_percent, 0) FROM referral_config WHERE id = 1").Scan(&discountPct)

	return referralInfo{
		affiliateID: referredByID,
		amount:      int64(float64(priceRupiah) * discountPct / 100),
	}
}

func validateCampaignAccess(ctx context.Context, campaignID, tenantID string) bool {
	var exists bool
	err := repository.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1 AND tenant_id = $2)
	`, campaignID, tenantID).Scan(&exists)
	return err == nil && exists
}

type invoiceParams struct {
	tenantID          string
	externalID        string
	campaignID        string
	orderType         string
	quantity          int
	priceRupiah       int64
	referralDiscount  int64
	finalPrice        int64
}

func createXenditInvoice(ctx context.Context, params invoiceParams) (checkoutResult, error) {
	xClient, err := getTenantXenditClient(ctx, params.tenantID)
	if err != nil {
		return createDevModeInvoice(ctx, params, err)
	}
	return createRealXenditInvoice(ctx, params, xClient)
}

func createDevModeInvoice(ctx context.Context, params invoiceParams, xerr error) (checkoutResult, error) {
	slog.Error("Failed to get xendit client for campaign", "tenant_id", params.tenantID, "error", xerr)
	if AppConfig.Env == "development" {
		slog.Warn("DEV mode: Mocking Xendit invoice for campaign checkout")
		invoiceID := params.externalID
		paymentURL := fmt.Sprintf("https://checkout.xendit.co/web/%s", params.externalID)
		_, _ = repository.DB.Exec(ctx, `
			INSERT INTO campaign_billing_orders
				(tenant_id, campaign_id, order_type, amount_rupiah, quantity, invoice_url, xendit_invoice_id, status, referral_discount_rupiah, final_amount_rupiah)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9)
		`, params.tenantID, params.campaignID, params.orderType, params.priceRupiah, params.quantity, paymentURL, invoiceID, params.referralDiscount, params.finalPrice)
		return checkoutResult{paymentURL: paymentURL, invoiceID: invoiceID}, nil
	}
	return checkoutResult{}, fmt.Errorf("payment provider not configured")
}

func createRealXenditInvoice(ctx context.Context, params invoiceParams, xClient *xendit.APIClient) (checkoutResult, error) {
	frontendURL := getFrontendURL()
	desc := fmt.Sprintf("Campaign %s - %s x%d", params.campaignID, params.orderType, params.quantity)
	currency := string(invoice.INVOICECURRENCY_IDR)
	createReq := invoice.CreateInvoiceRequest{
		ExternalId:         params.externalID,
		Amount:             float64(params.finalPrice),
		Description:        &desc,
		Currency:           &currency,
		SuccessRedirectUrl: strPtr(frontendURL + "/campaign/success?order=" + params.externalID),
		FailureRedirectUrl: strPtr(frontendURL + "/campaign/failed?order=" + params.externalID),
	}
	resp, _, xenditErr := xClient.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(createReq).Execute()
	if xenditErr != nil {
		slog.Error("Failed to create Xendit campaign invoice", "error", xenditErr)
		return checkoutResult{}, fmt.Errorf("failed to create invoice")
	}

	invoiceID := *resp.Id
	paymentURL := resp.GetInvoiceUrl()
	_, dbErr := repository.DB.Exec(ctx, `
		INSERT INTO campaign_billing_orders
			(tenant_id, campaign_id, order_type, amount_rupiah, quantity, invoice_url, xendit_invoice_id, status, referral_discount_rupiah, final_amount_rupiah)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9)
	`, params.tenantID, params.campaignID, params.orderType, params.priceRupiah, params.quantity, paymentURL, invoiceID, params.referralDiscount, params.finalPrice)
	if dbErr != nil {
		slog.Warn("Failed to save campaign order to DB", "error", dbErr)
	}
	return checkoutResult{paymentURL: paymentURL, invoiceID: invoiceID}, nil
}

func strPtr(s string) *string { return &s }

func saveReferralDiscount(ctx context.Context, invoiceID string, affiliateID int, amount int64) error {
	_, err := repository.DB.Exec(ctx, `
		INSERT INTO invoice_referrals (invoice_id, affiliate_id, discount_amount)
		VALUES ($1, $2, $3) ON CONFLICT (invoice_id) DO NOTHING
	`, invoiceID, affiliateID, amount)
	return err
}

// HandleBillingWebhook processes Xendit callback — with X-Callback-Token validation.
func HandleBillingWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}

	// Read body once so both token validation and payload decode can use it.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: errInvalidJSON})
		return
	}

	type XenditPayload struct {
		ID         string  `json:"id"`
		ExternalID string  `json:"external_id"`
		Status     string  `json:"status"`
		PaidAmount float64 `json:"paid_amount"`
	}

	var payload XenditPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: errInvalidJSON})
		return
	}

	// Validate token using already-parsed external_id to avoid double body read.
	cbToken := r.Header.Get("X-Callback-Token")
	if !checkWebhookToken(r.Context(), w, cbToken, payload.ExternalID) {
		return
	}

	ctx := context.Background()
	tx, err := repository.DB.Begin(ctx)
	if err != nil {
		WriteJSON(w, http.StatusOK, APIResponse{Message: "Acknowledged"})
		return
	}
	defer tx.Rollback(ctx)

	var orderID, tenantID, campaignID, orderType string
	var quantity int
	err = tx.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, campaign_id::text, order_type, quantity
		FROM campaign_billing_orders
		WHERE xendit_invoice_id = $1 AND status = 'PENDING'
		FOR UPDATE
	`, payload.ID).Scan(&orderID, &tenantID, &campaignID, &orderType, &quantity)

	if err != nil {
		if err == sql.ErrNoRows {
			WriteJSON(w, http.StatusOK, APIResponse{Message: "Already processed or not found"})
		} else {
			WriteJSON(w, http.StatusOK, APIResponse{Message: "Acknowledged"})
		}
		return
	}

	processBillingTx(ctx, tx, payload.Status, billingOrder{
		orderID:    orderID,
		tenantID:   tenantID,
		campaignID: campaignID,
		orderType:  orderType,
		quantity:   quantity,
		paidAmount: payload.PaidAmount,
		invoiceID:  payload.ID,
	})
	WriteJSON(w, http.StatusOK, APIResponse{Message: "OK"})
}

// checkWebhookToken validates the callback token using an already-parsed externalID.
// Avoids double-reading r.Body — caller must parse the body first and pass externalID.
func checkWebhookToken(ctx context.Context, w http.ResponseWriter, callbackToken, externalID string) bool {
	var tenantID string
	if externalID != "" {
		parts := splitExternalID(externalID)
		if len(parts) >= 2 {
			tenantID = parts[1]
		}
	}

	if tenantID != "" {
		var dbToken string
		err := repository.DB.QueryRow(ctx, "SELECT xendit_webhook_token FROM tenants WHERE id = $1", tenantID).Scan(&dbToken)
		if err == nil && dbToken != "" && callbackToken != dbToken {
			slog.Warn("Unauthorized campaign webhook: token mismatch", "tenant_id", tenantID)
			WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Unauthorized"})
			return false
		}
	} else {
		envToken := os.Getenv("XENDIT_WEBHOOK_TOKEN")
		if envToken != "" && callbackToken != envToken {
			slog.Warn("Unauthorized campaign webhook: env token mismatch")
			WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Unauthorized"})
			return false
		}
	}
	return true
}

type billingOrder struct {
	orderID     string
	tenantID    string
	campaignID  string
	orderType   string
	quantity    int
	paidAmount  float64
	invoiceID   string
}

func processBillingTx(ctx context.Context, tx pgx.Tx, status string, order billingOrder) {
	switch status {
	case "PAID", "SETTLED":
		tx.Exec(ctx, "UPDATE campaign_billing_orders SET status = 'PAID', paid_at = NOW() WHERE id = $1", order.orderID)
		applyOrder(ctx, tx, order.orderType, order.quantity, order.campaignID)
		applyCommission(ctx, tx, order.tenantID, order.paidAmount, order.invoiceID)
		if err := tx.Commit(ctx); err != nil {
			slog.Error(errCommitCampaignBillingTx, "error", err)
		}
	case "EXPIRED":
		tx.Exec(ctx, "UPDATE campaign_billing_orders SET status = 'EXPIRED' WHERE id = $1", order.orderID)
		if err := tx.Commit(ctx); err != nil {
			slog.Error(errCommitCampaignBillingTx, "error", err)
		}
	default:
		if err := tx.Commit(ctx); err != nil {
			slog.Error(errCommitCampaignBillingTx, "error", err)
		}
	}
}

func applyOrder(ctx context.Context, tx pgx.Tx, orderType string, quantity int, campaignID string) {
	if orderType == "wargame_token" {
		tx.Exec(ctx, "UPDATE campaigns SET wargame_tokens = wargame_tokens + $1 WHERE id = $2", 10*quantity, campaignID)
	}
}

func applyCommission(ctx context.Context, tx pgx.Tx, tenantID string, paidAmount float64, invoiceID string) {
	var referredByID *int
	tx.QueryRow(ctx, "SELECT referred_by_affiliate_id FROM tenants WHERE id = $1", tenantID).Scan(&referredByID)
	if referredByID == nil {
		return
	}

	var commissionPct float64
	tx.QueryRow(ctx, `SELECT COALESCE(commission_percent, 10) FROM referral_config WHERE id = 1`).Scan(&commissionPct)
	commission := int64(paidAmount * commissionPct / 100)
	if commission <= 0 {
		return
	}

	tx.Exec(ctx, `UPDATE affiliates SET cash_balance_rupiah = cash_balance_rupiah + $1, total_earnings_rupiah = total_earnings_rupiah + $1, updated_at = NOW() WHERE id = $2`, commission, *referredByID)
	tx.Exec(ctx, `INSERT INTO affiliate_earnings (affiliate_id, tenant_id, invoice_id, amount_rupiah, commission_rate_percent) VALUES ($1, $2, $3, $4, $5)`, *referredByID, tenantID, invoiceID, commission, int(commissionPct))
}
