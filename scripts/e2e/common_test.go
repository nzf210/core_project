package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	authServiceURL    = "http://localhost:8001"
	billingServiceURL = "http://localhost:8003"
	accountingURL     = "http://localhost:8201"
	chatbotURL        = "http://localhost:8203"
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

type BillingResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Helper: Setup auth with custom store name
func setupAuthWithStoreName(t *testing.T, storeName string) (string, string) {
	nano := time.Now().UnixNano()
	phone := fmt.Sprintf("628%d", nano%9000000000)
	username := fmt.Sprintf("testuser_%d", nano)

	registerReq := RegisterReq{
		Username:    username,
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

	// Verify OTP
	verifyReq := map[string]string{
		"phoneNumber": phone,
		"otp":         "000000",
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

	// Login
	loginReq := LoginReq{
		Username: username,
		Password: "TestPassword123!",
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

	loginData, ok := loginResp.Data.(map[string]interface{})
	if !ok || loginData == nil {
		t.Fatalf("Login data is not a map or nil: %T", loginResp.Data)
	}

	accessToken, _ := loginData["accessToken"].(string)
	tenantID, _ := loginData["tenantId"].(string)

	if accessToken == "" || tenantID == "" {
		t.Fatalf("Missing token or tenantID in login response data: %+v", loginData)
	}

	// Update tenant name via Superadmin
	superLogin := map[string]string{
		"username": "superadmin",
		"password": "superadmin123",
	}
	superBody, _ := json.Marshal(superLogin)
	respSuper, err := http.Post(authServiceURL+"/superadmin/login", "application/json", bytes.NewBuffer(superBody))
	if err != nil {
		t.Fatalf("Superadmin login failed: %v", err)
	}
	defer respSuper.Body.Close()

	var superResp Response
	json.NewDecoder(respSuper.Body).Decode(&superResp)
	if !superResp.Success {
		t.Fatalf("Superadmin login failed: %s", superResp.Message)
	}

	superData := superResp.Data.(map[string]interface{})
	superToken := superData["accessToken"].(string)

	updateReq := map[string]interface{}{
		"tenant_id":     tenantID,
		"name":          storeName,
		"plan":          "free",
		"business_type": "umum",
	}
	updateBody, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", authServiceURL+"/superadmin/tenants/profile", bytes.NewBuffer(updateBody))
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")

	respUpdate, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Update tenant profile failed: %v", err)
	}
	defer respUpdate.Body.Close()

	var updateResp Response
	json.NewDecoder(respUpdate.Body).Decode(&updateResp)
	if !updateResp.Success {
		t.Fatalf("Update tenant profile failed: %s", updateResp.Message)
	}

	return accessToken, tenantID
}

// Helper: Add product to tenant
func addProductToTenant(t *testing.T, token, tenantID, productName string, price int64) {
	productReq := map[string]interface{}{
		"name":  productName,
		"price": float64(price),
	}
	productBody, _ := json.Marshal(productReq)
	req, _ := http.NewRequest("POST", accountingURL+"/products", bytes.NewBuffer(productBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to add product: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		t.Fatalf("Add product failed with status %d: %+v", resp.StatusCode, errResp)
	}
}

// Helper: Check if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
