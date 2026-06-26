package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"core_project/apps/campaign/api/repository"

	"github.com/google/uuid"
	xendit "github.com/xendit/xendit-go/v6"
	invoice "github.com/xendit/xendit-go/v6/invoice"
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
	env := os.Getenv("APP_ENV")
	if env == "production" {
		return "https://wch.id"
	}
	return "http://localhost:3201"
}

// HandleBillingCheckout creates a real Xendit invoice for campaign checkout.
func HandleBillingCheckout(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Missing tenant ID"})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	type Request struct {
		CampaignID string `json:"campaign_id"`
		OrderType  string `json:"order_type"`
		Quantity   int    `json:"quantity"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid JSON"})
		return
	}

	var priceRupiah int64
	switch req.OrderType {
	case "wargame_token":
		priceRupiah = 100_000 * int64(req.Quantity)
	case "intelligence_pack":
		priceRupiah = 5_000_000 * int64(req.Quantity)
	default:
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid order type"})
		return
	}

	ctx := context.Background()

	// Validate campaign belongs to tenant
	var campaignExists bool
	err := repository.DB.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1 AND tenant_id = $2)
	`, req.CampaignID, tenantID).Scan(&campaignExists)
	if err != nil || !campaignExists {
		WriteJSON(w, http.StatusForbidden, APIResponse{Message: "Campaign not found or not owned by tenant"})
		return
	}

	externalID := "camp_" + uuid.NewString()[:20]

	// Check for referral
	var referredByAffiliateID *int
	var referralDiscountPct float64
	err = repository.DB.QueryRow(ctx,
		"SELECT referred_by_affiliate_id FROM tenants WHERE id = $1", tenantID,
	).Scan(&referredByAffiliateID)
	if err == nil && referredByAffiliateID != nil {
		_ = repository.DB.QueryRow(ctx,
			"SELECT COALESCE(discount_percent, 0) FROM referral_config WHERE id = 1",
		).Scan(&referralDiscountPct)
	}

	referralDiscountAmount := int64(float64(priceRupiah) * referralDiscountPct / 100)
	finalPrice := priceRupiah - referralDiscountAmount

	// Get Xendit client
	xClient, errXc := getTenantXenditClient(ctx, tenantID)

	var paymentURL, invoiceID string

	if errXc != nil {
		slog.Error("Failed to get xendit client for campaign", "tenant_id", tenantID, "error", errXc)
		if os.Getenv("APP_ENV") == "development" || os.Getenv("ENV") == "development" || os.Getenv("ENV") == "" {
			slog.Warn("DEV mode: Mocking Xendit invoice for campaign checkout")
			invoiceID = externalID
			paymentURL = fmt.Sprintf("https://checkout.xendit.co/web/%s", externalID)
			_, _ = repository.DB.Exec(ctx, `
				INSERT INTO campaign_billing_orders
					(tenant_id, campaign_id, order_type, amount_rupiah, quantity, invoice_url, xendit_invoice_id, status, referral_discount_rupiah, final_amount_rupiah)
				VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9)
			`, tenantID, req.CampaignID, req.OrderType, priceRupiah, req.Quantity,
				paymentURL, invoiceID, referralDiscountAmount, finalPrice)
		} else {
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Payment provider not configured"})
			return
		}
	} else {
		// Real Xendit invoice creation
		frontendURL := getFrontendURL()
		successURL := frontendURL + "/campaign/success?order=" + externalID
		failureURL := frontendURL + "/campaign/failed?order=" + externalID

		desc := fmt.Sprintf("Campaign %s - %s x%d", req.CampaignID, req.OrderType, req.Quantity)
		currency := string(invoice.INVOICECURRENCY_IDR)
		createReq := invoice.CreateInvoiceRequest{
			ExternalId:         externalID,
			Amount:             float64(finalPrice),
			Description:        &desc,
			Currency:           &currency,
			SuccessRedirectUrl: &successURL,
			FailureRedirectUrl: &failureURL,
		}
		resp, _, xenditErr := xClient.InvoiceApi.CreateInvoice(ctx).CreateInvoiceRequest(createReq).Execute()
		if xenditErr != nil {
			slog.Error("Failed to create Xendit campaign invoice", "error", xenditErr)
			WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create invoice"})
			return
		}
		invoiceID = *resp.Id
		paymentURL = resp.GetInvoiceUrl()

		_, dbErr := repository.DB.Exec(ctx, `
			INSERT INTO campaign_billing_orders
				(tenant_id, campaign_id, order_type, amount_rupiah, quantity, invoice_url, xendit_invoice_id, status, referral_discount_rupiah, final_amount_rupiah)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', $8, $9)
		`, tenantID, req.CampaignID, req.OrderType, priceRupiah, req.Quantity,
			paymentURL, invoiceID, referralDiscountAmount, finalPrice)
		if dbErr != nil {
			slog.Warn("Failed to save campaign order to DB", "error", dbErr)
		}
	}

	// Record referral discount
	if referredByAffiliateID != nil && referralDiscountAmount > 0 {
		_, _ = repository.DB.Exec(ctx, `
			INSERT INTO invoice_referrals (invoice_id, affiliate_id, discount_amount)
			VALUES ($1, $2, $3) ON CONFLICT (invoice_id) DO NOTHING
		`, invoiceID, *referredByAffiliateID, referralDiscountAmount)
	}

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"invoice_url": paymentURL,
			"invoice_id":  invoiceID,
		},
	})
}

// HandleBillingWebhook processes Xendit callback — with X-Callback-Token validation.
func HandleBillingWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	// Validate X-Callback-Token (F054 AC-12)
	callbackToken := r.Header.Get("X-Callback-Token")
	if callbackToken != "" {
		ctx := context.Background()
		var rawPayload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&rawPayload); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid JSON"})
			return
		}
		externalID, _ := rawPayload["external_id"].(string)

		var tenantID string
		if externalID != "" {
			parts := splitExternalID(externalID)
			if len(parts) >= 2 {
				tenantID = parts[1]
			}
		}

		if tenantID != "" {
			var dbToken string
			err := repository.DB.QueryRow(ctx,
				"SELECT xendit_webhook_token FROM tenants WHERE id = $1", tenantID,
			).Scan(&dbToken)
			if err == nil && dbToken != "" {
				if callbackToken != dbToken {
					slog.Warn("Unauthorized campaign webhook: token mismatch",
						"tenant_id", tenantID, "expected", dbToken, "got", callbackToken)
					WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Unauthorized"})
					return
				}
			} else if envToken := os.Getenv("XENDIT_WEBHOOK_TOKEN"); envToken != "" {
				if callbackToken != envToken {
					slog.Warn("Unauthorized campaign webhook: env token mismatch")
					WriteJSON(w, http.StatusUnauthorized, APIResponse{Message: "Unauthorized"})
					return
				}
			}
		}
	}

	type XenditPayload struct {
		ID         string  `json:"id"`
		ExternalID string  `json:"external_id"`
		Status     string  `json:"status"`
		PaidAmount float64 `json:"paid_amount"`
	}

	var payload XenditPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid JSON"})
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

	switch payload.Status {
	case "PAID", "SETTLED":
		tx.Exec(ctx, "UPDATE campaign_billing_orders SET status = 'PAID', paid_at = NOW() WHERE id = $1", orderID)

		switch orderType {
		case "wargame_token":
			tokensToAdd := 10 * quantity
			tx.Exec(ctx, "UPDATE campaigns SET wargame_tokens = wargame_tokens + $1 WHERE id = $2", tokensToAdd, campaignID)
		case "intelligence_pack":
			tx.Exec(ctx, `
				UPDATE campaigns
				SET active_addons = (
					SELECT jsonb_agg(DISTINCT e) FROM (
						SELECT jsonb_array_elements_text(COALESCE(active_addons, '[]'::jsonb)) as e
						UNION SELECT 'fraud_map' UNION SELECT 'anomaly_detect'
					) sub
				)
				WHERE id = $1
			`, campaignID)
		}

		// F054: Affiliate commission from real paid_amount
		var referredByID *int
		tx.QueryRow(ctx, "SELECT referred_by_affiliate_id FROM tenants WHERE id = $1", tenantID).Scan(&referredByID)
		if referredByID != nil {
			var commissionPct float64
			_ = tx.QueryRow(ctx, `
				SELECT COALESCE(commission_percent, 10) FROM referral_config WHERE id = 1
			`).Scan(&commissionPct)

			commission := int64(payload.PaidAmount * float64(commissionPct) / 100)
			if commission > 0 {
				tx.Exec(ctx, `
					UPDATE affiliates
					SET cash_balance_rupiah = cash_balance_rupiah + $1,
						total_earnings_rupiah = total_earnings_rupiah + $1,
						updated_at = NOW()
					WHERE id = $2
				`, commission, *referredByID)
				tx.Exec(ctx, `
					INSERT INTO affiliate_earnings (affiliate_id, tenant_id, invoice_id, amount_rupiah, commission_rate_percent)
					VALUES ($1, $2, $3, $4, $5)
				`, *referredByID, tenantID, payload.ID, commission, int(commissionPct))
			}
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("Failed to commit campaign billing tx", "error", err)
		}
	case "EXPIRED":
		tx.Exec(ctx, "UPDATE campaign_billing_orders SET status = 'EXPIRED' WHERE id = $1", orderID)
		if err := tx.Commit(ctx); err != nil {
			slog.Error("Failed to commit campaign billing tx", "error", err)
		}
	default:
		if err := tx.Commit(ctx); err != nil {
			slog.Error("Failed to commit campaign billing tx", "error", err)
		}
	}

	WriteJSON(w, http.StatusOK, APIResponse{Message: "OK"})
}
