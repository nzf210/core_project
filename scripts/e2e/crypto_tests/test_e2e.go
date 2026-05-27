package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := []byte("your_32_character_super_secret_jwt_key_here")
	claims := jwt.MapClaims{
		"tenant_id": "tenant-test-1",
		"user_id":   "user-test-1",
		"exp":       time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		fmt.Println("Error generating token:", err)
		return
	}

	baseURL := "http://localhost:8000/api/crypto"

	// 1. Get Dashboard
	fmt.Println("Testing Dashboard...")
	resp, err := getReq(baseURL+"/dashboard", tokenString)
	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("Dashboard failed: %v, status: %d\n", err, resp.StatusCode)
	} else {
		fmt.Println("Dashboard OK!")
	}

	// 2. Create Paper Trading Bot
	fmt.Println("Creating Paper Trading Bot...")
	botPayload := map[string]interface{}{
		"name": "E2E Test Bot",
		"bot_type": "dca",
		"pair": "BTCUSDT",
		"is_paper_trading": true,
		"dca_interval": "daily",
		"dca_amount": 50,
	}
	botBody, _ := json.Marshal(botPayload)
	resp, err = postReq(baseURL+"/bots", tokenString, botBody)
	if err != nil || resp.StatusCode != 201 {
		fmt.Printf("Create Bot failed: %v, status: %d\n", err, resp.StatusCode)
		return
	}

	var createRes struct{ Data struct { ID string `json:"id"` } `json:"data"` }
	json.NewDecoder(resp.Body).Decode(&createRes)
	botID := createRes.Data.ID
	fmt.Println("Create Bot OK! ID:", botID)

	// 3. List Bots
	fmt.Println("Listing Bots...")
	resp, err = getReq(baseURL+"/bots", tokenString)
	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("List Bots failed: %v, status: %d\n", err, resp.StatusCode)
	} else {
		fmt.Println("List Bots OK!")
	}

	// 4. Update Bot Status (Stop)
	fmt.Println("Stopping Bot...")
	patchPayload := map[string]interface{}{ "status": "stopped" }
	patchBody, _ := json.Marshal(patchPayload)
	resp, err = putReq(fmt.Sprintf("%s/bots/%s/status", baseURL, botID), tokenString, patchBody)
	if err != nil || resp.StatusCode != 200 {
		fmt.Printf("Stop Bot failed: %v, status: %d\n", err, resp.StatusCode)
	} else {
		fmt.Println("Stop Bot OK!")
	}

	fmt.Println("E2E Test Completed Successfully!")
}

func getReq(url, token string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

func postReq(url, token string, body []byte) (*http.Response, error) {
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func putReq(url, token string, body []byte) (*http.Response, error) {
	req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}
