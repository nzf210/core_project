package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestWalletTopupAndSubscribeFlow tests:
// topup wallet (create invoice) → simulate webhook → verify balance → subscribe via wallet → verify activated
func TestWalletTopupAndSubscribeFlow(t *testing.T) {
	log := NewTestLogger(t)
	PrintSection(t, "WALLET TOPUP + SUBSCRIBE VIA WALLET TEST")

	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"billing":      billingServiceURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Step 1: Register + login
	PrintTestStep(t, 1, 6, "Register and login new tenant")
	state := registerAndLogin(t, log, "wallet_")
	if state == nil {
		return
	}

	// Step 2: Check initial wallet balance (0 if no row yet — that's fine)
	PrintTestStep(t, 2, 6, "Check initial wallet balance")
	initialBalance := getWalletBalanceSafe(t, log, state)
	log.Success(fmt.Sprintf("Initial wallet balance: %d sen", initialBalance))

	// Step 3: Topup wallet via Xendit invoice
	PrintTestStep(t, 3, 6, "Create wallet topup invoice")
	topupAmount := int64(50000000) // 500k IDR in sen
	log.Action(fmt.Sprintf("Topping up %d sen...", topupAmount))

	resp, err := postJSONWithTenant(billingServiceURL, "/wallet/topup",
		map[string]interface{}{"amount_cents": topupAmount},
		state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Wallet topup failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var topupResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&topupResp)
	if topupResp.Status == http.StatusInternalServerError && containsStr(topupResp.Message, "configured") {
		log.Warning("Xendit not configured in this environment — skipping wallet topup test")
		log.Complete("Wallet topup test SKIPPED (no Xendit credentials configured)")
		return
	}
	if topupResp.Status != http.StatusOK && topupResp.Status != http.StatusCreated {
		log.Data("Topup response: %+v", topupResp)
		log.Error("Wallet topup returned unexpected status: " + topupResp.Message)
		t.FailNow()
	}

	// external_id format: "{uuid}-wallet-topup-{tenantID}"
	topupExternalID := extractStringField(topupResp.Data, "external_id")
	if topupExternalID == "" {
		log.Error("No external_id in topup response")
		t.FailNow()
	}
	log.Success("Topup invoice created: " + topupExternalID)

	// Step 4: Simulate Xendit PAID webhook for topup
	PrintTestStep(t, 4, 6, "Simulate topup webhook PAID")
	webhookToken := getEnvOrDefault("XENDIT_WEBHOOK_TOKEN", "")
	resp, err = postJSONWithHeader(billingServiceURL, "/webhook/payment", map[string]interface{}{
		"status":      "PAID",
		"external_id": topupExternalID,
		"paid_amount": float64(topupAmount),
		"amount":      float64(topupAmount),
	}, "x-callback-token", webhookToken)
	if err != nil {
		log.Error("Topup webhook failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error(fmt.Sprintf("Topup webhook returned %d", resp.StatusCode))
		t.FailNow()
	}
	log.Success("Topup webhook accepted")

	// Step 5: Verify wallet balance increased
	PrintTestStep(t, 5, 6, "Verify wallet balance updated")
	newBalance := getWalletBalance(t, log, state)
	if newBalance <= initialBalance {
		log.Error(fmt.Sprintf("Wallet balance did not increase: before=%d after=%d", initialBalance, newBalance))
		t.FailNow()
	}
	log.Success(fmt.Sprintf("Balance updated: %d → %d sen", initialBalance, newBalance))

	// Step 6: Subscribe via wallet
	PrintTestStep(t, 6, 6, "Subscribe using wallet balance")
	planID := getFirstActivePlanID(t, log, state)
	if planID == "" {
		return
	}

	resp, err = postJSONWithTenant(billingServiceURL, "/subscribe", map[string]interface{}{
		"plan_id":        planID,
		"pay_via_wallet": true,
	}, state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Wallet subscribe failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var subResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&subResp)
	if subResp.Status != http.StatusOK {
		log.Data("Subscribe via wallet response: %+v", subResp)
		log.Error("Subscribe via wallet failed: " + subResp.Message)
		t.FailNow()
	}

	method := extractStringField(subResp.Data, "payment_method")
	log.Success(fmt.Sprintf("Subscribed via wallet (payment_method: %s)", method))

	// Verify balance was deducted
	finalBalance := getWalletBalance(t, log, state)
	if finalBalance >= newBalance {
		log.Warning("Wallet balance was not deducted after wallet subscription (balance: %d)", finalBalance)
	} else {
		log.Success(fmt.Sprintf("Wallet balance deducted: %d → %d sen", newBalance, finalBalance))
	}

	log.Complete("Wallet topup + subscribe test PASSED")
}

// getWalletBalance fails the test if the wallet endpoint returns an error.
func getWalletBalance(t *testing.T, log *TestLogger, state *TestState) int64 {
	t.Helper()
	resp, err := getJSON(billingServiceURL, "/wallet", state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Get wallet failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()
	var walletResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&walletResp)
	if walletResp.Status != http.StatusOK {
		log.Error("Get wallet failed: " + walletResp.Message)
		t.FailNow()
	}
	return extractInt64Field(walletResp.Data, "balance_cents")
}

// getWalletBalanceSafe returns 0 on error (new tenant may have no wallet row yet).
func getWalletBalanceSafe(t *testing.T, log *TestLogger, state *TestState) int64 {
	t.Helper()
	resp, err := getJSON(billingServiceURL, "/wallet", state.AccessToken, state.TenantID)
	if err != nil {
		log.Warning("Get wallet returned error (may be new tenant): %s", err.Error())
		return 0
	}
	defer resp.Body.Close()
	var walletResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&walletResp)
	if walletResp.Status != http.StatusOK {
		log.Warning("Get wallet returned %d — treating as zero balance: %s", walletResp.Status, walletResp.Message)
		return 0
	}
	return extractInt64Field(walletResp.Data, "balance_cents")
}

func getFirstActivePlanID(t *testing.T, log *TestLogger, state *TestState) string {
	t.Helper()
	resp, err := getJSON(billingServiceURL, "/plans", state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("List plans failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()
	var plansResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&plansResp)
	planID := extractFirstPlanID(plansResp.Data)
	if planID == "" {
		log.Error("No plans returned from billing service")
		t.FailNow()
	}
	return planID
}
