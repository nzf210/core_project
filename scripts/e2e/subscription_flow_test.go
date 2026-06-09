package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestSubscriptionFlow(t *testing.T) {
	// 1. Register new tenant
	t.Log("Step 1: Register new tenant")
	nano := time.Now().UnixNano()
	phone := fmt.Sprintf("628%d", nano%9000000000)
	registerReq := RegisterReq{
		Username:    fmt.Sprintf("testuser_%d", nano),
		Password:    "TestPassword123!",
		Email:       fmt.Sprintf("test_%d@example.com", nano),
		PhoneNumber: phone,
	}

	registerBody, _ := json.Marshal(registerReq)
	resp, err := http.Post(
		authServiceURL+"/register",
		"application/json",
		bytes.NewBuffer(registerBody),
	)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer resp.Body.Close()

	var registerResp Response
	json.NewDecoder(resp.Body).Decode(&registerResp)
	if !registerResp.Success {
		t.Fatalf("Register failed: %s", registerResp.Message)
	}
	t.Log("✓ Register OTP sent")

	// 1b. Verify OTP (using test OTP "000000" for dev)
	t.Log("Step 1b: Verify OTP")
	verifyReq := map[string]string{
		"phoneNumber": phone,
		"otp":         "000000", // Dev mode accepts any OTP
	}
	verifyBody, _ := json.Marshal(verifyReq)
	resp, err = http.Post(
		authServiceURL+"/verify-otp",
		"application/json",
		bytes.NewBuffer(verifyBody),
	)
	if err != nil {
		t.Fatalf("Verify OTP failed: %v", err)
	}
	defer resp.Body.Close()

	var verifyResp Response
	json.NewDecoder(resp.Body).Decode(&verifyResp)
	if !verifyResp.Success {
		t.Logf("Verify OTP response: %+v", verifyResp)
		t.Fatalf("Verify OTP failed: %s", verifyResp.Message)
	}
	t.Log("✓ OTP verified, account created")

	// 2. Login
	t.Log("Step 2: Login")
	loginReq := LoginReq{
		Username: registerReq.Username,
		Password: registerReq.Password,
	}

	loginBody, _ := json.Marshal(loginReq)
	resp, err = http.Post(
		authServiceURL+"/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	defer resp.Body.Close()

	var loginResp Response
	json.NewDecoder(resp.Body).Decode(&loginResp)
	if !loginResp.Success {
		t.Fatalf("Login failed: %s", loginResp.Message)
	}

	loginData := loginResp.Data.(map[string]interface{})
	accessToken := loginData["accessToken"].(string)
	tenantID := loginData["tenantId"].(string)
	t.Logf("✓ Login successful (tenant_id: %s)", tenantID)

	// 3. List plans
	t.Log("Step 3: List plans")
	req, _ := http.NewRequest("GET", billingServiceURL+"/plans", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("List plans failed: %v", err)
	}
	defer resp.Body.Close()

	var plansResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&plansResp)
	if plansResp.Status != http.StatusOK {
		t.Fatalf("List plans failed: %s", plansResp.Message)
	}

	plans := plansResp.Data.([]interface{})
	if len(plans) == 0 {
		t.Fatal("No plans available")
	}
	planID := plans[0].(map[string]interface{})["id"].(string)
	t.Logf("✓ Plans listed (first plan: %s)", planID)

	// 4. Subscribe to plan
	t.Log("Step 4: Subscribe to plan")
	subscribeReq := SubscribeReq{
		PlanID: planID,
	}

	subscribeBody, _ := json.Marshal(subscribeReq)
	req, _ = http.NewRequest(
		"POST",
		billingServiceURL+"/subscribe",
		bytes.NewBuffer(subscribeBody),
	)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer resp.Body.Close()

	var subscribeResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&subscribeResp)
	if subscribeResp.Status != http.StatusOK {
		t.Fatalf("Subscribe failed: %s", subscribeResp.Message)
	}
	t.Log("✓ Subscribe successful")

	// 5. Verify subscription
	t.Log("Step 5: Verify subscription")
	req, _ = http.NewRequest("GET", billingServiceURL+"/subscription", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Get subscription failed: %v", err)
	}
	defer resp.Body.Close()

	var subResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&subResp)
	if subResp.Status != http.StatusOK {
		t.Fatalf("Get subscription failed: %s", subResp.Message)
	}
	t.Log("✓ Subscription verified")

	// 6. Test accounting access (should work)
	t.Log("Step 6: Test accounting access")
	req, _ = http.NewRequest("GET", accountingURL+"/accounts", nil)
	req.Header.Set("X-Tenant-ID", tenantID)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Accounting access failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Accounting access failed: status %d", resp.StatusCode)
	}
	t.Log("✓ Accounting access successful")

	t.Log("\n✅ Subscription flow test PASSED")
}

func TestVoucherFlow(t *testing.T) {
	t.Log("Voucher flow test - TODO")
}

func TestN8nWorkflows(t *testing.T) {
	t.Log("N8n workflows test - TODO")
}
