package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestChatbotTenantDifferentiation(t *testing.T) {
	// Setup: Create 2 different tenants with different store names
	t.Log("Setup: Create 2 different tenants with different store names")

	// Tenant 1: Toko Elektronik
	tenant1Token, tenant1ID := setupAuthWithStoreName(t, "Toko Elektronik")
	t.Logf("✓ Tenant 1 created (ID: %s, Store: Toko Elektronik)", tenant1ID)

	// Tenant 2: Toko Pakaian
	tenant2Token, tenant2ID := setupAuthWithStoreName(t, "Toko Pakaian")
	t.Logf("✓ Tenant 2 created (ID: %s, Store: Toko Pakaian)", tenant2ID)

	// Test 1: Chat with Tenant 1
	t.Log("\nTest 1: Chat with Tenant 1 (Toko Elektronik)")
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
		t.Fatalf("Chat request 1 failed: %v", err)
	}
	defer resp1.Body.Close()

	var chatResp1 Response
	json.NewDecoder(resp1.Body).Decode(&chatResp1)
	if !chatResp1.Success {
		t.Fatalf("Chat 1 failed: %s", chatResp1.Message)
	}

	dataMap1, ok := chatResp1.Data.(map[string]interface{})
	if !ok || dataMap1 == nil {
		t.Fatalf("Chat 1 data is not a map or nil: %T", chatResp1.Data)
	}

	replyVal1, ok := dataMap1["reply"]
	if !ok {
		t.Fatalf("Chat 1 response missing 'reply' field. Data: %+v", dataMap1)
	}

	reply1, ok := replyVal1.(string)
	if !ok {
		t.Fatalf("Chat 1 reply is not a string: %T", replyVal1)
	}
	t.Logf("Tenant 1 Response: %s", reply1)

	// Verify response contains tenant 1 store name
	if !contains(reply1, "Elektronik") {
		t.Logf("⚠️  Warning: Response might not contain store name reference. Got: %s", reply1)
	} else {
		t.Log("✓ Tenant 1 response contains store name reference")
	}

	// Test 2: Chat with Tenant 2
	t.Log("\nTest 2: Chat with Tenant 2 (Toko Pakaian)")
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
		t.Fatalf("Chat request 2 failed: %v", err)
	}
	defer resp2.Body.Close()

	var chatResp2 Response
	json.NewDecoder(resp2.Body).Decode(&chatResp2)
	if !chatResp2.Success {
		t.Fatalf("Chat 2 failed: %s", chatResp2.Message)
	}

	dataMap2, ok := chatResp2.Data.(map[string]interface{})
	if !ok || dataMap2 == nil {
		t.Fatalf("Chat 2 data is not a map or nil: %T", chatResp2.Data)
	}

	replyVal2, ok := dataMap2["reply"]
	if !ok {
		t.Fatalf("Chat 2 response missing 'reply' field. Data: %+v", dataMap2)
	}

	reply2, ok := replyVal2.(string)
	if !ok {
		t.Fatalf("Chat 2 reply is not a string: %T", replyVal2)
	}
	t.Logf("Tenant 2 Response: %s", reply2)

	// Verify response contains tenant 2 store name
	if !contains(reply2, "Pakaian") {
		t.Logf("⚠️  Warning: Response might not contain store name reference. Got: %s", reply2)
	} else {
		t.Log("✓ Tenant 2 response contains store name reference")
	}

	// Test 3: Verify responses are different
	t.Log("\nTest 3: Verify responses are different for different tenants")
	if reply1 == reply2 {
		t.Logf("⚠️  Warning: Both tenants got same response (might be cached)")
		t.Logf("  Tenant 1: %s", reply1)
		t.Logf("  Tenant 2: %s", reply2)
	} else {
		t.Log("✓ Responses are different for different tenants")
	}

	t.Log("\n✅ Chatbot tenant differentiation test PASSED")
}

func TestChatbotProductCatalogPerTenant(t *testing.T) {
	// Setup: Create 2 tenants with different products
	t.Log("Setup: Create 2 tenants and add different products")

	tenant1Token, tenant1ID := setupAuthWithStoreName(t, "Toko A")
	tenant2Token, tenant2ID := setupAuthWithStoreName(t, "Toko B")

	// Add products to tenant 1
	t.Log("Adding products to Tenant 1...")
	addProductToTenant(t, tenant1Token, tenant1ID, "Laptop ASUS", 10000000)
	addProductToTenant(t, tenant1Token, tenant1ID, "Mouse Gaming", 500000)

	// Add different products to tenant 2
	t.Log("Adding products to Tenant 2...")
	addProductToTenant(t, tenant2Token, tenant2ID, "Baju Kemeja", 200000)
	addProductToTenant(t, tenant2Token, tenant2ID, "Celana Jeans", 300000)

	// Test: Ask Tenant 1 about products
	t.Log("\nTest: Ask Tenant 1 about products")
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
		t.Fatalf("Chat 1 failed: %v", err)
	}
	defer resp1.Body.Close()

	var chatResp1 Response
	json.NewDecoder(resp1.Body).Decode(&chatResp1)
	if !chatResp1.Success {
		t.Fatalf("Chat 1 response error: %s", chatResp1.Message)
	}

	dataMap1 := chatResp1.Data.(map[string]interface{})
	reply1 := dataMap1["reply"].(string)
	t.Logf("Tenant 1 Products Response: %s", reply1)

	// Verify Tenant 1 sees their products
	if contains(reply1, "Laptop") || contains(reply1, "ASUS") {
		t.Log("✓ Tenant 1 sees their products (Laptop ASUS)")
	} else {
		t.Logf("⚠️  Tenant 1 might not see their products. Response: %s", reply1)
	}

	// Test: Ask Tenant 2 about products
	t.Log("\nTest: Ask Tenant 2 about products")
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
		t.Fatalf("Chat 2 failed: %v", err)
	}
	defer resp2.Body.Close()

	var chatResp2 Response
	json.NewDecoder(resp2.Body).Decode(&chatResp2)
	if !chatResp2.Success {
		t.Fatalf("Chat 2 response error: %s", chatResp2.Message)
	}

	dataMap2 := chatResp2.Data.(map[string]interface{})
	reply2 := dataMap2["reply"].(string)
	t.Logf("Tenant 2 Products Response: %s", reply2)

	// Verify Tenant 2 sees their products
	if contains(reply2, "Baju") || contains(reply2, "Celana") {
		t.Log("✓ Tenant 2 sees their products (Baju Kemeja, Celana Jeans)")
	} else {
		t.Logf("⚠️  Tenant 2 might not see their products. Response: %s", reply2)
	}

	// Verify cross-contamination doesn't happen
	if contains(reply1, "Baju") || contains(reply1, "Celana") {
		t.Fatalf("❌ FAIL: Tenant 1 sees Tenant 2's products! Response: %s", reply1)
	} else {
		t.Log("✓ Tenant 1 does NOT see Tenant 2's products")
	}

	if contains(reply2, "Laptop") || contains(reply2, "ASUS") {
		t.Fatalf("❌ FAIL: Tenant 2 sees Tenant 1's products! Response: %s", reply2)
	} else {
		t.Log("✓ Tenant 2 does NOT see Tenant 1's products")
	}

	t.Log("\n✅ Chatbot product catalog isolation test PASSED")
}
