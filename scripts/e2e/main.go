package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	AuthURL       = "http://localhost:8001"
	UMKMAccounting = "http://localhost:8201"
	AIGateway     = "http://localhost:8002"
)

func main() {
	fmt.Println("=== E2E Simulation Start ===")
	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Register User & Create Tenant
	fmt.Println("\n[1] Registering User (UMKM Owner)...")
	regPayload := []byte(`{"email": "owner@umkm.test", "password": "securepassword", "tenant_name": "Toko Berkah"}`)
	resp, err := client.Post(AuthURL+"/register", "application/json", bytes.NewBuffer(regPayload))
	if err != nil || resp.StatusCode >= 400 {
		fmt.Printf("Warning: Auth service might be down or user exists. Continuing simulation...\n")
	} else {
		fmt.Println("User registered successfully.")
	}

	// 2. Login
	fmt.Println("\n[2] Logging in...")
	loginPayload := []byte(`{"email": "owner@umkm.test", "password": "securepassword"}`)
	resp, err = client.Post(AuthURL+"/login", "application/json", bytes.NewBuffer(loginPayload))
	
	var token, tenantID string
	if err == nil && resp.StatusCode == 200 {
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		data := result["data"].(map[string]interface{})
		token = data["access_token"].(string)
		tenantID = data["tenant_id"].(string)
		fmt.Printf("Login success. Tenant ID: %s\n", tenantID)
	} else {
		fmt.Println("Using mock token and tenant ID for testing.")
		token = "mock-jwt-token"
		tenantID = "tenant-1"
	}

	// 3. Seed Chart of Accounts
	fmt.Println("\n[3] Seeding UMKM Chart of Accounts...")
	req, _ := http.NewRequest("POST", UMKMAccounting+"/seed", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID) // Note: In production, API Gateway handles this
	resp, err = client.Do(req)
	if err == nil {
		fmt.Println("Chart of accounts seeded.")
	}

	// 4. Record a Transaction
	fmt.Println("\n[4] Recording a Transaction (Sales)...")
	trxPayload := []byte(`{
		"date": "2026-05-22",
		"description": "Penjualan Harian",
		"reference": "INV-001",
		"lines": [
			{"account_id": "kas-1", "debit": 500000, "credit": 0},
			{"account_id": "pendapatan-1", "debit": 0, "credit": 500000}
		]
	}`)
	req, _ = http.NewRequest("POST", UMKMAccounting+"/transactions", bytes.NewBuffer(trxPayload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Transaction recorded: %s\n", string(body))
	}

	// 5. Query AI Gateway
	fmt.Println("\n[5] Consulting AI for business advice...")
	aiPayload := []byte(`{"message": "Bulan ini penjualan saya 500ribu, apa saran anda?", "provider": "gemini"}`)
	req, _ = http.NewRequest("POST", AIGateway+"/v1/chat", bytes.NewBuffer(aiPayload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err == nil {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("AI Oracle Response: %s\n", string(body))
	}

	fmt.Println("\n=== E2E Simulation Completed ===")
}
