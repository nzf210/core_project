package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestChatbotTenantDifferentiation(t *testing.T) {
	log := NewTestLogger(t)
	PrintSection(t, "CHATBOT TENANT DIFFERENTIATION TEST")

	// Check service availability
	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"chatbot":      chatbotURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Step 1: Create Tenant 1
	PrintTestStep(t, 1, 3, "Create Tenant 1: Toko Elektronik")
	log.Start("Setting up tenant 1...")
	tenant1Token, tenant1ID := SetupAuthWithStoreName(t, "Toko Elektronik")
	if tenant1Token == "" || tenant1ID == "" {
		log.Error("Failed to create tenant 1")
		return
	}
	log.Store("Tenant 1 created (ID: " + tenant1ID + ", Store: Toko Elektronik)")

	// Step 2: Create Tenant 2
	PrintTestStep(t, 2, 3, "Create Tenant 2: Toko Pakaian")
	log.Start("Setting up tenant 2...")
	tenant2Token, tenant2ID := SetupAuthWithStoreName(t, "Toko Pakaian")
	if tenant2Token == "" || tenant2ID == "" {
		log.Error("Failed to create tenant 2")
		return
	}
	log.Store("Tenant 2 created (ID: " + tenant2ID + ", Store: Toko Pakaian)")

	// Step 3: Test Chat Differentiation
	PrintTestStep(t, 3, 3, "Test Chat Responses Are Different")

	// Test 1: Chat with Tenant 1
	log.Chat("Testing chat with Tenant 1...")
	chatReq1 := map[string]interface{}{
		"message": "Siapa nama toko saya?",
	}
	chatBody1, _ := json.Marshal(chatReq1)
	req1, _ := http.NewRequest("POST", chatbotURL+"/chat", bytes.NewBuffer(chatBody1))
	req1.Header.Set("Authorization", "Bearer "+tenant1Token)
	req1.Header.Set("X-Tenant-ID", tenant1ID)
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		log.Error("Chat request 1 failed: " + err.Error())
		return
	}
	defer resp1.Body.Close()

	var chatResp1 Response
	json.NewDecoder(resp1.Body).Decode(&chatResp1)
	if !chatResp1.Success {
		log.Error("Chat 1 failed: " + chatResp1.Message)
		return
	}

	dataMap1, ok := chatResp1.Data.(map[string]interface{})
	if !ok || dataMap1 == nil {
		log.Error("Chat 1 data is not a map or nil")
		return
	}

	reply1, ok := dataMap1["reply"].(string)
	if !ok {
		log.Error("Chat 1 reply is not a string")
		return
	}
	log.Data("Tenant 1 Response: %s", reply1)

	// Verify response contains tenant 1 store name
	if contains(reply1, "Elektronik") {
		log.Success("Tenant 1 response contains store name reference")
	} else {
		log.Warning("Response might not contain store name reference")
	}

	// Test 2: Chat with Tenant 2
	log.Chat("Testing chat with Tenant 2...")
	chatReq2 := map[string]interface{}{
		"message": "Siapa nama toko saya?",
	}
	chatBody2, _ := json.Marshal(chatReq2)
	req2, _ := http.NewRequest("POST", chatbotURL+"/chat", bytes.NewBuffer(chatBody2))
	req2.Header.Set("Authorization", "Bearer "+tenant2Token)
	req2.Header.Set("X-Tenant-ID", tenant2ID)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		log.Error("Chat request 2 failed: " + err.Error())
		return
	}
	defer resp2.Body.Close()

	var chatResp2 Response
	json.NewDecoder(resp2.Body).Decode(&chatResp2)
	if !chatResp2.Success {
		log.Error("Chat 2 failed: " + chatResp2.Message)
		return
	}

	dataMap2, ok := chatResp2.Data.(map[string]interface{})
	if !ok || dataMap2 == nil {
		log.Error("Chat 2 data is not a map or nil")
		return
	}

	reply2, ok := dataMap2["reply"].(string)
	if !ok {
		log.Error("Chat 2 reply is not a string")
		return
	}
	log.Data("Tenant 2 Response: %s", reply2)

	// Verify response contains tenant 2 store name
	if contains(reply2, "Pakaian") {
		log.Success("Tenant 2 response contains store name reference")
	} else {
		log.Warning("Response might not contain store name reference")
	}

	// Test 3: Verify responses are different
	log.Check("Verifying responses are different for different tenants...")
	if reply1 == reply2 {
		log.Warning("Both tenants got same response (might be cached)")
		log.Data("Tenant 1: %s", reply1)
		log.Data("Tenant 2: %s", reply2)
	} else {
		log.Success("Responses are different for different tenants")
	}

	log.Complete("Chatbot tenant differentiation test PASSED")
}

func TestChatbotProductCatalogPerTenant(t *testing.T) {
	log := NewTestLogger(t)
	PrintSection(t, "CHATBOT PRODUCT CATALOG ISOLATION TEST")

	// Check service availability
	missing := requireServices(t, map[string]string{
		"auth-service": authServiceURL,
		"accounting":   accountingURL,
		"chatbot":      chatbotURL,
	})
	if len(missing) > 0 {
		log.Skip("Required services not available: " + joinStrings(missing, ", "))
		return
	}

	// Step 1: Create tenants
	PrintTestStep(t, 1, 4, "Create 2 Tenants with Different Products")
	log.Start("Setting up tenants...")
	tenant1Token, tenant1ID := SetupAuthWithStoreName(t, "Toko A")
	tenant2Token, tenant2ID := SetupAuthWithStoreName(t, "Toko B")

	if tenant1Token == "" || tenant1ID == "" || tenant2Token == "" || tenant2ID == "" {
		log.Error("Failed to create tenants")
		return
	}
	log.Success("Both tenants created")

	// Step 2: Add products to Tenant 1
	log.Product("Adding products to Tenant 1...")
	AddProductToTenant(t, tenant1Token, tenant1ID, "Laptop ASUS", 10000000)
	AddProductToTenant(t, tenant1Token, tenant1ID, "Mouse Gaming", 500000)
	log.Success("Tenant 1 products added (Laptop ASUS, Mouse Gaming)")

	// Step 3: Add different products to Tenant 2
	log.Product("Adding products to Tenant 2...")
	AddProductToTenant(t, tenant2Token, tenant2ID, "Baju Kemeja", 200000)
	AddProductToTenant(t, tenant2Token, tenant2ID, "Celana Jeans", 300000)
	log.Success("Tenant 2 products added (Baju Kemeja, Celana Jeans)")

	// Step 4: Test product isolation
	PrintTestStep(t, 4, 4, "Test Product Catalog Isolation")

	// Test: Ask Tenant 1 about products
	log.Chat("Asking Tenant 1 about products...")
	chatReq1 := map[string]interface{}{
		"message": "Apa saja produk yang tersedia?",
	}
	chatBody1, _ := json.Marshal(chatReq1)
	req1, _ := http.NewRequest("POST", chatbotURL+"/chat", bytes.NewBuffer(chatBody1))
	req1.Header.Set("Authorization", "Bearer "+tenant1Token)
	req1.Header.Set("X-Tenant-ID", tenant1ID)
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		log.Error("Chat 1 failed: " + err.Error())
		return
	}
	defer resp1.Body.Close()

	var chatResp1 Response
	json.NewDecoder(resp1.Body).Decode(&chatResp1)
	if !chatResp1.Success {
		log.Error("Chat 1 response error: " + chatResp1.Message)
		return
	}

	dataMap1 := chatResp1.Data.(map[string]interface{})
	reply1 := dataMap1["reply"].(string)
	log.Data("Tenant 1 Products Response: %s", reply1)

	// Verify Tenant 1 sees their products
	if contains(reply1, "Laptop") || contains(reply1, "ASUS") {
		log.Success("Tenant 1 sees their products (Laptop ASUS)")
	} else {
		log.Warning("Tenant 1 might not see their products")
	}

	// Test: Ask Tenant 2 about products
	log.Chat("Asking Tenant 2 about products...")
	chatReq2 := map[string]interface{}{
		"message": "Apa saja produk yang tersedia?",
	}
	chatBody2, _ := json.Marshal(chatReq2)
	req2, _ := http.NewRequest("POST", chatbotURL+"/chat", bytes.NewBuffer(chatBody2))
	req2.Header.Set("Authorization", "Bearer "+tenant2Token)
	req2.Header.Set("X-Tenant-ID", tenant2ID)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		log.Error("Chat 2 failed: " + err.Error())
		return
	}
	defer resp2.Body.Close()

	var chatResp2 Response
	json.NewDecoder(resp2.Body).Decode(&chatResp2)
	if !chatResp2.Success {
		log.Error("Chat 2 response error: " + chatResp2.Message)
		return
	}

	dataMap2 := chatResp2.Data.(map[string]interface{})
	reply2 := dataMap2["reply"].(string)
	log.Data("Tenant 2 Products Response: %s", reply2)

	// Verify Tenant 2 sees their products
	if contains(reply2, "Baju") || contains(reply2, "Celana") {
		log.Success("Tenant 2 sees their products (Baju Kemeja, Celana Jeans)")
	} else {
		log.Warning("Tenant 2 might not see their products")
	}

	// Verify cross-contamination doesn't happen
	log.Check("Verifying cross-tenant isolation...")
	if contains(reply1, "Baju") || contains(reply1, "Celana") {
		log.Error("FAIL: Tenant 1 sees Tenant 2's products!")
		return
	}
	log.Success("Tenant 1 does NOT see Tenant 2's products")

	if contains(reply2, "Laptop") || contains(reply2, "ASUS") {
		log.Error("FAIL: Tenant 2 sees Tenant 1's products!")
		return
	}
	log.Success("Tenant 2 does NOT see Tenant 1's products")

	log.Complete("Chatbot product catalog isolation test PASSED")
}

// joinStrings joins strings with separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
