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
	log := NewTestLogger(t)
	PrintSection(t, "VOUCHER FLOW INTEGRATION TEST")

	// Check service availability
	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"billing":      billingServiceURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Setup: Get superadmin token
	log.Start("Authenticating as superadmin...")
	superToken := SuperadminLogin(t)
	if superToken == "" {
		log.Error("Superadmin authentication failed")
		return
	}
	log.Auth("Superadmin authenticated")

	// Step 1: Create voucher program
	PrintTestStep(t, 1, 4, "Create Voucher Program")
	log.Voucher("Creating voucher program...")
	programName := fmt.Sprintf("Test Promo %d", time.Now().Unix())

	createProgramReq := map[string]interface{}{
		"name":            programName,
		"description":     "Test voucher program",
		"voucher_type":    "discount_percent",
		"discount_value":  10,
		"target_plan_id": "lite",
		"duration_months": 1,
		"max_uses":        100,
		"starts_at":       time.Now().Format("2006-01-02T15:04:05Z"),
		"expires_at":      time.Now().AddDate(0, 1, 0).Format("2006-01-02T15:04:05Z"),
	}

	programBody, _ := json.Marshal(createProgramReq)
	req, _ := http.NewRequest(
		"POST",
		billingServiceURL+"/admin/voucher-programs",
		bytes.NewBuffer(programBody),
	)
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("Create program failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var createResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&createResp)
	if createResp.Status != http.StatusOK && createResp.Status != http.StatusCreated {
		log.Data("Create program response: %+v", createResp)
		log.Error("Create program failed: " + createResp.Message)
		return
	}
	log.Success("Voucher program created: " + programName)

	// Step 2: Generate voucher links
	PrintTestStep(t, 2, 4, "Generate Voucher Links")
	programData := createResp.Data.(map[string]interface{})
	programID := programData["id"].(string)

	log.Voucher("Generating voucher links...")
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
	req.Header.Set("Authorization", "Bearer "+superToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Error("Generate links failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var generateResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&generateResp)
	if generateResp.Status != http.StatusOK && generateResp.Status != http.StatusCreated {
		log.Error("Generate links failed: " + generateResp.Message)
		return
	}

	generateData := generateResp.Data.(map[string]interface{})
	links := generateData["links"].([]interface{})
	if len(links) == 0 {
		log.Error("No voucher links generated")
		return
	}
	log.Success(fmt.Sprintf("Generated %d voucher links", len(links)))

	// Step 3: List voucher links
	PrintTestStep(t, 3, 4, "List Voucher Links")
	log.Voucher("Fetching voucher links...")
	req, _ = http.NewRequest(
		"GET",
		fmt.Sprintf("%s/admin/voucher-links?program_id=%s", billingServiceURL, programID),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+superToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Error("List links failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var listResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&listResp)
	if listResp.Status != http.StatusOK {
		log.Error("List links failed: " + listResp.Message)
		return
	}
	log.Success("Voucher links listed")

	// Step 4: Get voucher analytics
	PrintTestStep(t, 4, 4, "Get Voucher Analytics")
	log.Voucher("Fetching voucher analytics...")
	req, _ = http.NewRequest(
		"GET",
		fmt.Sprintf("%s/admin/voucher-analytics?program_id=%s", billingServiceURL, programID),
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+superToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Error("Get analytics failed: " + err.Error())
		return
	}
	defer resp.Body.Close()

	var analyticsResp BillingResponse
	json.NewDecoder(resp.Body).Decode(&analyticsResp)
	if analyticsResp.Status != http.StatusOK {
		log.Error("Get analytics failed: " + analyticsResp.Message)
		return
	}
	log.Success("Voucher analytics retrieved")

	log.Complete("Voucher flow test PASSED")
}
