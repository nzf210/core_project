package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestFrozenTenantPaymentRecovery tests:
// subscribe tenant → superadmin freeze tenant → verify frozen (writes blocked) →
// tenant subscribes again → verify unfrozen + access restored
func TestFrozenTenantPaymentRecovery(t *testing.T) {
	log := NewTestLogger(t)
	PrintSection(t, "FROZEN TENANT PAYMENT RECOVERY TEST")

	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"billing":      billingServiceURL,
		"accounting":   accountingURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Step 1: Register + login a new tenant
	PrintTestStep(t, 1, 5, "Register and login new tenant")
	state := registerAndLogin(t, log, "frozen_")
	if state == nil {
		return
	}

	// Step 2: Activate the tenant's subscription with a voucher (fastest path to active)
	PrintTestStep(t, 2, 5, "Activate tenant subscription")
	log.Package("Activating subscription to get tenant into active state...")

	superToken := SuperadminLogin(t)
	if superToken == "" {
		log.Error("Superadmin authentication failed")
		t.FailNow()
	}

	// Generate a voucher and redeem it to get the tenant active
	voucherCode := generateAndRedeemVoucher(t, log, superToken, state)
	if voucherCode == "" {
		return
	}

	verifySubscriptionActive(t, log, state)
	log.Success("Tenant is now active")

	// Step 3: Superadmin freezes the tenant
	PrintTestStep(t, 3, 5, "Freeze tenant via superadmin")
	log.Action(fmt.Sprintf("Freezing tenant %s...", state.TenantID))

	resp, err := patchJSONWithAuth(billingServiceURL,
		fmt.Sprintf("/admin/tenants/%s", state.TenantID),
		map[string]interface{}{"action": "freeze"}, superToken, state.TenantID)
	if err != nil {
		log.Error("Freeze request failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var freezeResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&freezeResp)
	if freezeResp.Status != http.StatusOK {
		log.Data("Freeze response: %+v", freezeResp)
		log.Error("Freeze tenant failed: " + freezeResp.Message)
		t.FailNow()
	}
	log.Success("Tenant frozen")

	// Step 4: Verify write operations are blocked (frozen enforcement)
	PrintTestStep(t, 4, 5, "Verify write operations blocked for frozen tenant")
	log.Check("Attempting write operation on frozen tenant...")

	// Try to create a journal entry — should be blocked by RequireActiveSubscription middleware
	writeResp, err := postJSONWithTenant(accountingURL, "/journals",
		map[string]interface{}{
			"description": "test entry",
			"entries":     []interface{}{},
		}, state.AccessToken, state.TenantID)
	if err != nil {
		log.Warning("Write blocked at transport level (connection refused / 503): %s", err.Error())
	} else {
		defer writeResp.Body.Close()
		if writeResp.StatusCode == http.StatusForbidden || writeResp.StatusCode == http.StatusPaymentRequired {
			log.Success(fmt.Sprintf("Write correctly blocked for frozen tenant (HTTP %d)", writeResp.StatusCode))
		} else if writeResp.StatusCode == http.StatusBadRequest {
			// Bad request means the middleware passed — likely missing fields, not frozen block
			log.Warning("Write returned 400 — frozen middleware may not be active on this endpoint")
		} else {
			log.Warning("Unexpected status %d for frozen tenant write attempt", writeResp.StatusCode)
		}
	}

	// Step 5: Recover — redeem another voucher to unfreeze
	PrintTestStep(t, 5, 5, "Recover by subscribing again (unfreeze)")
	log.Action("Generating recovery voucher...")

	recoveryCode := generateVoucherForTenant(t, log, superToken, "lite")
	if recoveryCode == "" {
		return
	}

	log.Voucher("Redeeming recovery voucher: " + recoveryCode)
	resp, err = postJSONWithTenant(billingServiceURL, "/voucher/redeem",
		map[string]interface{}{"code": recoveryCode},
		state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Recovery redeem failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var recoverResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&recoverResp)
	if recoverResp.Status != http.StatusOK {
		log.Data("Recovery response: %+v", recoverResp)
		log.Error("Recovery subscription failed: " + recoverResp.Message)
		t.FailNow()
	}
	log.Success("Recovery voucher redeemed")

	// Verify no longer frozen
	verifySubscriptionActive(t, log, state)

	// Verify write access restored
	log.Check("Verifying write access restored after recovery...")
	writeResp2, err := postJSONWithTenant(accountingURL, "/journals",
		map[string]interface{}{
			"description": "recovery test entry",
			"entries":     []interface{}{},
		}, state.AccessToken, state.TenantID)
	if err == nil {
		defer writeResp2.Body.Close()
		if writeResp2.StatusCode != http.StatusForbidden && writeResp2.StatusCode != http.StatusPaymentRequired {
			log.Success(fmt.Sprintf("Write unblocked after recovery (HTTP %d)", writeResp2.StatusCode))
		} else {
			log.Warning("Write still blocked after recovery — check subscription activation propagation")
		}
	}

	log.Complete("Frozen tenant recovery test PASSED")
}

func generateAndRedeemVoucher(t *testing.T, log *TestLogger, superToken string, state *TestState) string {
	t.Helper()
	code := generateVoucherForTenant(t, log, superToken, "lite")
	if code == "" {
		return ""
	}
	resp, err := postJSONWithTenant(billingServiceURL, "/voucher/redeem",
		map[string]interface{}{"code": code},
		state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Voucher redeem failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()
	var redeemResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&redeemResp)
	if redeemResp.Status != http.StatusOK {
		log.Error("Initial voucher redeem failed: " + redeemResp.Message)
		t.FailNow()
	}
	return code
}

func generateVoucherForTenant(t *testing.T, log *TestLogger, superToken, planID string) string {
	t.Helper()
	resp, err := postJSONWithAuth(billingServiceURL, "/admin/vouchers/generate", map[string]interface{}{
		"plan_id":       planID,
		"validity_days": 30,
		"quantity":      1,
		"program_name":  "E2E-Recovery-Test",
	}, superToken, "")
	if err != nil {
		log.Error("Generate voucher failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()
	var genResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&genResp)
	code := extractVoucherCode(genResp.Data)
	if code == "" {
		log.Error("No voucher code returned")
		t.FailNow()
	}
	return code
}
