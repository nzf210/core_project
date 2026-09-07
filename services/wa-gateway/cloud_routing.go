package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// ─────────────────────────────────────────────
// Cloud API Routing
// ─────────────────────────────────────────────

// resolveProviderPreference determines the final WA provider preference for a request.
// Priority: X-WA-Provider-Override header > DB lookup > "auto"
func resolveProviderPreference(r *http.Request, tenantID string) string {
	if override := r.Header.Get("X-WA-Provider-Override"); override != "" {
		return override
	}
	return getTenantWAProviderPreference(tenantID)
}

// getTenantWAProviderPreference fetches preference from DB
func getTenantWAProviderPreference(tenantID string) string {
	if db == nil {
		return "auto"
	}
	var preference string
	err := db.QueryRow("SELECT wa_provider_preference FROM tenant_chatbot_configs WHERE tenant_id = $1", tenantID).Scan(&preference)
	if err != nil {
		return "auto"
	}
	return preference
}

// isTransactional determines if a message should go via Meta Cloud API
func isTransactional(r *http.Request) bool {
	msgType := r.Header.Get("X-Message-Type")
	switch msgType {
	case "otp", "invoice", "payment", "subscription", "system", "broadcast":
		return true
	}
	source := r.Header.Get("X-Source")
	switch source {
	case "auth-service", "billing-service", "notification-service":
		return true
	}
	return false
}

// cloudAPIHTTPClient is a pooled HTTP client for inter-service communication with wa-cloud-api
var cloudAPIHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	},
}

// routeToCloudAPI sends a message via the wa-cloud-api service (Meta Cloud API)
func routeToCloudAPI(tenantID, target, message, msgType string) (string, error) {
	cloudAPIHost := "http://localhost:8210"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		cloudAPIHost = "http://wa-cloud-api:8210"
	}

	payload := map[string]interface{}{
		"to":   target,
		"type": "text",
		"text": message,
	}
	if msgType != "" {
		payload["type"] = msgType
	}

	body, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cloudAPIHost+"/send", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := cloudAPIHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("cloud API: failed to parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := "unknown error"
		if m, ok := result["message"].(string); ok {
			errMsg = m
		}
		return "", fmt.Errorf("cloud API: %s", errMsg)
	}

	waMsgID := ""
	if id, ok := result["wa_message_id"].(string); ok {
		waMsgID = id
	}

	slog.Info("Routed message via Cloud API", "tenant_id", tenantID, "wa_msg_id", waMsgID)
	return waMsgID, nil
}
