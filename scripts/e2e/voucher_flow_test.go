package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestVoucherFlowIntegration(t *testing.T) {
	// Setup: Get auth token and tenant ID
	t.Log("Setup: Authenticate and get tenant")
	accessToken, _ := setupAuth(t)

	// 1. Create voucher program
	t.Log("Step 1: Create voucher program")
	createProgramReq := map[string]interface{}{
		"name":              fmt.Sprintf("Test Promo %d", time.Now().Unix()),
		"description":       "Test voucher program",
		"voucher_type":      "discount_percent",
		"discount_value":    10,
		"target_plan_id":    "lite", // Assuming lite plan exists
		"duration_months":   1,
		"max_uses":          100,
		"starts_at":         time.Now().Format("2006-01-02T15:04:05Z"),
		"expires_at":        time.Now().AddDate(0, 1, 0).Format("2006-01-02T15:04:05Z"),
	}

	programBody, _ := json.Marshal(createProgramReq)
	req, _ := http.NewRequest(
		"POST",
		billingServiceURL+"/admin/voucher-programs",
		bytes.NewBuffer(programBody),
	)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Create program failed: %v", err)
	}
	defer resp.Body.Close()

	var createResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&createResp)
	if createResp.Status != http.StatusOK && createResp.Status != http.StatusCreated {
		t.Logf("Create program response: %+v", createResp)
		t.Fatalf("Create program failed: %s", createResp.Message)
	}
	t.Log("✓ Voucher program created")

	// 2. Generate voucher links
	t.Log("Step 2: Generate voucher links")
	programData := createResp.Data.(map[string]interface{})
	programID := programData["id"].(string)

	generateReq := map[string]interface{}{
		"program_id": programID,
		"count":      10,
		"valid_days": 30,
		"base_url":   "https://app.wch.id",
	}

	generateBody, _ := json.Marshal(generateReq)
	req, _ = http.NewRequest(
		"POST",
		billingServiceURL+"/admin/voucher-links/generate",
		bytes.NewBuffer(generateBody),
	)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Generate links failed: %v", err)
	}
	defer resp.Body.Close()

	var generateResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&generateResp)
	if generateResp.Status != http.StatusOK && generateResp.Status != http.StatusCreated {
		t.Fatalf("Generate links failed: %s", generateResp.Message)
	}

	generateData := generateResp.Data.(map[string]interface{})
	links := generateData["links"].([]interface{})
	if len(links) == 0 {
		t.Fatal("No voucher links generated")
	}
	t.Logf("✓ Generated %d voucher links", len(links))

	// 3. List voucher links
	t.Log("Step 3: List voucher links")
	req, _ = http.NewRequest(
		"GET",
		fmt.Sprintf("%s/admin/voucher-links?program_id=%s", billingServiceURL, programID),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("List links failed: %v", err)
	}
	defer resp.Body.Close()

	var listResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&listResp)
	if listResp.Status != http.StatusOK {
		t.Fatalf("List links failed: %s", listResp.Message)
	}
	t.Log("✓ Voucher links listed")

	// 4. Get voucher analytics
	t.Log("Step 4: Get voucher analytics")
	req, _ = http.NewRequest(
		"GET",
		fmt.Sprintf("%s/admin/voucher-analytics?program_id=%s", billingServiceURL, programID),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Get analytics failed: %v", err)
	}
	defer resp.Body.Close()

	var analyticsResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&analyticsResp)
	if analyticsResp.Status != http.StatusOK {
		t.Fatalf("Get analytics failed: %s", analyticsResp.Message)
	}
	t.Log("✓ Voucher analytics retrieved")

	t.Log("\n✅ Voucher flow test PASSED")
}

func setupAuth(t *testing.T) (string, string) {
	superLogin := map[string]string{
		"username": "superadmin",
		"password": "superadmin123",
	}
	superBody, _ := json.Marshal(superLogin)
	resp, err := http.Post(
		authServiceURL+"/superadmin/login",
		"application/json",
		bytes.NewBuffer(superBody),
	)
	if err != nil {
		t.Fatalf("Superadmin login failed: %v", err)
	}
	defer resp.Body.Close()

	var loginResp Response
	json.NewDecoder(resp.Body).Decode(&loginResp)
	if !loginResp.Success {
		t.Fatalf("Superadmin login failed: %s", loginResp.Message)
	}

	loginData := loginResp.Data.(map[string]interface{})
	return loginData["accessToken"].(string), ""
}
