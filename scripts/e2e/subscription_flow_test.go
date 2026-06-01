package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	authServiceURL    = "http://localhost:8001"
	billingServiceURL = "http://localhost:8003"
	accountingURL     = "http://localhost:8201"
)

type RegisterReq struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SubscribeReq struct {
	PlanID      string `json:"plan_id"`
	VoucherCode string `json:"voucher_code,omitempty"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func TestSubscriptionFlow(t *testing.T) {
	// 1. Register new tenant
	t.Log("Step 1: Register new tenant")
	registerReq := RegisterReq{
		Username:    fmt.Sprintf("testuser_%d", time.Now().Unix()),
		Password:    "TestPassword123!",
		Email:       fmt.Sprintf("test_%d@example.com", time.Now().Unix()),
		PhoneNumber: "6281234567890",
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
	t.Log("✓ Register successful")

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
	accessToken := loginData["access_token"].(string)
	tenantID := loginData["tenant_id"].(string)
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

	var plansResp Response
	json.NewDecoder(resp.Body).Decode(&plansResp)
	if !plansResp.Success {
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

	var subscribeResp Response
	json.NewDecoder(resp.Body).Decode(&subscribeResp)
	if !subscribeResp.Success {
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

	var subResp Response
	json.NewDecoder(resp.Body).Decode(&subResp)
	if !subResp.Success {
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
