package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"core_project/shared/sdk/response"

	xendit "github.com/xendit/xendit-go/v6"
)

func isSuperadmin(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get(response.XUserRole) != "superadmin" {
		response.Error(w, http.StatusForbidden, response.SuperadminOnly, nil)
		return false
	}
	return true
}

type cachedTenant struct {
	name      string
	email     string
	waNumber  string
	telegram  string
	expiresAt time.Time
}

type xenditClientCacheEntry struct {
	client    *xendit.APIClient
	createdAt time.Time
}

var (
	xenditClientMu    sync.RWMutex
	xenditClientCache = make(map[string]*xenditClientCacheEntry)
)

func getTenantXenditClient(ctx context.Context, tenantID string) (*xendit.APIClient, error) {
	xenditClientMu.RLock()
	entry, ok := xenditClientCache[tenantID]
	xenditClientMu.RUnlock()

	if ok && time.Since(entry.createdAt) < 5*time.Minute {
		return entry.client, nil
	}

	var apiKey string
	err := DB.QueryRow(ctx, "SELECT xendit_api_key FROM tenants WHERE id = $1", tenantID).Scan(&apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get xendit_api_key for tenant %s: %w", tenantID, err)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("tenant %s has no xendit_api_key configured", tenantID)
	}

	client := xendit.NewClient(apiKey)

	xenditClientMu.Lock()
	// Re-check under write lock: another goroutine may have populated the cache
	// between our RUnlock above and this Lock.
	if existing, exists := xenditClientCache[tenantID]; exists && time.Since(existing.createdAt) < 5*time.Minute {
		xenditClientMu.Unlock()
		return existing.client, nil
	}
	xenditClientCache[tenantID] = &xenditClientCacheEntry{
		client:    client,
		createdAt: time.Now(),
	}
	xenditClientMu.Unlock()

	return client, nil
}

func getTenantXenditWebhookToken(ctx context.Context, tenantID string) (string, error) {
	var token string
	err := DB.QueryRow(ctx, "SELECT xendit_webhook_token FROM tenants WHERE id = $1", tenantID).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("failed to get xendit_webhook_token for tenant %s: %w", tenantID, err)
	}
	return token, nil
}

type SubscribeReq struct {
	PlanID       string `json:"plan_id"`
	VoucherCode  string `json:"voucher_code,omitempty"`
	PayViaWallet bool   `json:"pay_via_wallet"`
}

type VoucherRedeemReq struct {
	Code string `json:"code"`
}

type TicketPayload struct {
	TicketNumber  string `json:"ticket_number"`
	TenantName    string `json:"tenant_name"`
	PlanName      string `json:"plan_name"`
	PlanID        string `json:"plan_id"`
	ActivatedAt   string `json:"activated_at"`
	ExpiresAt     string `json:"expires_at"`
	AmountPaid    int64  `json:"amount_paid"`
	PaymentMethod string `json:"payment_method"`
	VoucherCode   string `json:"voucher_code,omitempty"`
}

type planRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceMonthly int64  `json:"price_monthly"`
	PriceYearly  int64  `json:"price_yearly"`
	IsActive     bool   `json:"is_active"`
	SortOrder    int    `json:"sort_order"`
}

type featureRow struct {
	FeatureKey   string `json:"feature_key"`
	FeatureName  string `json:"feature_name"`
	FeatureValue string `json:"feature_value"`
	IsEnabled    bool   `json:"is_enabled"`
}

type planWithFeatures struct {
	planRow
	Features []featureRow `json:"features"`
}

type N8NStatus struct {
	Status          string `json:"status"`
	Version         string `json:"version"`
	ActiveWorkflows int    `json:"active_workflows"`
	QueueMode       bool   `json:"queue_mode"`
	LastHealthCheck string `json:"last_health_check"`
}
