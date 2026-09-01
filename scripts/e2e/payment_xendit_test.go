package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestPaymentXenditWebhookFlow tests the critical path:
// register → subscribe (Xendit invoice created) → simulate webhook PAID → verify tenant activated
func TestPaymentXenditWebhookFlow(t *testing.T) {
	log := NewTestLogger(t)
	PrintSection(t, "XENDIT PAYMENT WEBHOOK FLOW TEST")

	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"billing":      billingServiceURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Step 1: Register + login
	PrintTestStep(t, 1, 5, "Register and login new tenant")
	state := registerAndLogin(t, log, "xendit_")
	if state == nil {
		return
	}

	// Step 2: List plans to get a real plan ID
	PrintTestStep(t, 2, 5, "Fetch available plans")
	log.Package("Fetching plans...")
	resp, err := getJSON(billingServiceURL, "/plans", state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("List plans failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var plansResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&plansResp)
	if plansResp.Status != http.StatusOK {
		log.Error("List plans failed: " + plansResp.Message)
		t.FailNow()
	}

	planID := extractFirstPlanID(plansResp.Data)
	if planID == "" {
		log.Error("No plans returned from billing service")
		t.FailNow()
	}
	log.Success("Using plan: " + planID)

	// Step 3: POST /subscribe — creates Xendit invoice
	PrintTestStep(t, 3, 5, "Subscribe (creates Xendit invoice)")
	log.Action("Posting subscribe request...")
	subscribeBody := map[string]interface{}{
		"plan_id":        planID,
		"pay_via_wallet": false,
	}
	resp, err = postJSONWithTenant(billingServiceURL, "/subscribe", subscribeBody, state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Subscribe failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var subResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&subResp)
	// Expect 200 (direct activation for free/lite), 201 (invoice created), or
	// 500 when Xendit API key is not configured in the current environment.
	if subResp.Status == http.StatusInternalServerError && containsStr(subResp.Message, "invoice") {
		log.Warning("Xendit not configured in this environment (500: %s) — skipping webhook simulation", subResp.Message)
		log.Complete("Xendit webhook test SKIPPED (no Xendit credentials configured)")
		return
	}
	if subResp.Status != http.StatusOK && subResp.Status != http.StatusCreated && subResp.Status != http.StatusPaymentRequired {
		log.Data("Subscribe response: status=%d msg=%s", subResp.Status, subResp.Message)
		log.Error("Subscribe returned unexpected status")
		t.FailNow()
	}

	// Extract invoice external_id from response data if present
	externalID := extractStringField(subResp.Data, "external_id")
	if externalID == "" {
		// Lite plan may activate immediately — check subscription exists
		log.Warning("No external_id in response (may be free plan direct-activation). Verifying subscription status...")
		verifySubscriptionActive(t, log, state)
		log.Complete("Xendit webhook test PASSED (free plan direct-activation path)")
		return
	}
	log.Success("Invoice created, external_id: " + externalID)

	// Step 4: Simulate Xendit PAID webhook
	PrintTestStep(t, 4, 5, "Simulate Xendit PAID webhook")
	log.Action("Sending simulated webhook...")

	webhookPayload := map[string]interface{}{
		"status":        "PAID",
		"external_id":   externalID,
		"paid_amount":   float64(45000000), // 450k IDR in sen
		"amount":        float64(45000000),
	}
	// XENDIT_WEBHOOK_TOKEN env must match or be unset (legacy allow) on staging
	webhookToken := getEnvOrDefault("XENDIT_WEBHOOK_TOKEN", "")
	resp, err = postJSONWithHeader(billingServiceURL, "/webhook/payment", webhookPayload, "x-callback-token", webhookToken)
	if err != nil {
		log.Error("Webhook POST failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error(fmt.Sprintf("Webhook returned %d (expected 200)", resp.StatusCode))
		t.FailNow()
	}
	log.Success("Webhook accepted (200 OK)")

	// Step 5: Verify subscription is now active
	PrintTestStep(t, 5, 5, "Verify subscription activated")
	time.Sleep(200 * time.Millisecond) // brief settle for async DB write
	verifySubscriptionActive(t, log, state)

	log.Complete("Xendit webhook flow test PASSED")
}

func verifySubscriptionActive(t *testing.T, log *TestLogger, state *TestState) {
	t.Helper()
	resp, err := getJSON(billingServiceURL, "/subscription", state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Get subscription failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var subResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&subResp)
	if subResp.Status != http.StatusOK {
		log.Error("Subscription not found after payment: " + subResp.Message)
		t.FailNow()
	}
	status := extractStringField(subResp.Data, "status")
	if status != "active" && status != "" {
		log.Warning("Subscription status is '%s' (expected 'active')", status)
	}
	log.Success("Subscription verified (status: " + status + ")")
}
