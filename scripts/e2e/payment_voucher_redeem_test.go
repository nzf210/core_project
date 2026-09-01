package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestVoucherCodeRedemptionFlow tests tenant-side voucher redemption:
// superadmin generates voucher code → new tenant redeems it → verify subscription activated
func TestVoucherCodeRedemptionFlow(t *testing.T) {
	log := NewTestLogger(t)
	PrintSection(t, "VOUCHER CODE REDEMPTION FLOW TEST")

	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"billing":      billingServiceURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Step 1: Superadmin login
	PrintTestStep(t, 1, 4, "Superadmin login")
	superToken := SuperadminLogin(t)
	if superToken == "" {
		log.Error("Superadmin authentication failed")
		t.FailNow()
	}
	log.Auth("Superadmin authenticated")

	// Step 2: Generate a voucher code via superadmin endpoint
	PrintTestStep(t, 2, 4, "Generate voucher code")
	log.Voucher("Generating voucher code...")

	generateReq := map[string]interface{}{
		"plan_id":       "lite",
		"validity_days": 30,
		"quantity":      1,
		"program_name":  fmt.Sprintf("E2E-Test-%d", time.Now().Unix()),
	}
	resp, err := postJSONWithAuth(billingServiceURL, "/admin/vouchers/generate", generateReq, superToken, "")
	if err != nil {
		log.Error("Generate voucher failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var genResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&genResp)
	if genResp.Status != http.StatusOK && genResp.Status != http.StatusCreated {
		log.Data("Generate response: %+v", genResp)
		log.Error("Generate voucher failed: " + genResp.Message)
		t.FailNow()
	}

	voucherCode := extractVoucherCode(genResp.Data)
	if voucherCode == "" {
		log.Error("No voucher code returned from generate endpoint")
		t.FailNow()
	}
	log.Success("Voucher code generated: " + voucherCode)

	// Step 3: Register a new tenant to redeem the voucher
	PrintTestStep(t, 3, 4, "Register new tenant and redeem voucher")
	state := registerAndLogin(t, log, "voucherredeem_")
	if state == nil {
		return
	}

	log.Voucher("Redeeming voucher code: " + voucherCode)
	resp, err = postJSONWithTenant(billingServiceURL, "/voucher/redeem",
		map[string]interface{}{"code": voucherCode},
		state.AccessToken, state.TenantID)
	if err != nil {
		log.Error("Redeem voucher failed: " + err.Error())
		t.FailNow()
	}
	defer resp.Body.Close()

	var redeemResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&redeemResp)
	if redeemResp.Status != http.StatusOK {
		log.Data("Redeem response: %+v", redeemResp)
		log.Error("Voucher redemption failed: " + redeemResp.Message)
		t.FailNow()
	}
	log.Success("Voucher redeemed successfully")

	// Step 4: Verify subscription is now active
	PrintTestStep(t, 4, 4, "Verify subscription activated after redemption")
	verifySubscriptionActive(t, log, state)

	// Also verify the code is now marked redeemed (trying to redeem again should fail)
	log.Check("Verifying voucher cannot be redeemed twice...")
	resp2, err := postJSONWithTenant(billingServiceURL, "/voucher/redeem",
		map[string]interface{}{"code": voucherCode},
		state.AccessToken, state.TenantID)
	if err == nil {
		defer resp2.Body.Close()
		var dupResp BillingResponse
		json.NewDecoder(resp2.Body).Decode(&dupResp)
		if dupResp.Status == http.StatusOK {
			log.Warning("Voucher was accepted twice — idempotency issue!")
		} else {
			log.Success("Double-redeem correctly rejected (status: " + fmt.Sprintf("%d", dupResp.Status) + ")")
		}
	}

	log.Complete("Voucher code redemption test PASSED")
}
