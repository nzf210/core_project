package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func transcribeAudio(tenantID, mediaPath string) (string, error) {
	reqBody := map[string]interface{}{
		"tenant_id": tenantID,
		"audio_url": "file://" + mediaPath,
		"language":  "id",
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Post to AI Gateway
	gatewayURL := strings.Replace(AIGatewayURL, "/v1/chat", "/v1/audio/transcribe", 1)

	req, err := http.NewRequest("POST", gatewayURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if !res.Success {
		return "", fmt.Errorf("api returned success=false")
	}

	return res.Data.Text, nil
}

func analyzeImage(tenantID, mediaPath, prompt string) (string, error) {
	if prompt == "" {
		prompt = "Deskripsikan gambar ini untuk membantu pencatatan atau stok. Jika ini struk, baca total dan item."
	}
	reqBody := map[string]interface{}{
		"tenant_id": tenantID,
		"image_url": "file://" + mediaPath,
		"prompt":    prompt,
		"model":     "MiniMax-M3-Vision",
	}
	jsonBody, _ := json.Marshal(reqBody)

	gatewayURL := strings.Replace(AIGatewayURL, "/v1/chat", "/v1/vision", 1)

	req, err := http.NewRequest("POST", gatewayURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if !res.Success {
		return "", fmt.Errorf("api returned success=false")
	}

	return res.Data.Text, nil
}