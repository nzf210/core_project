package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	errSuperadminRequired = "Superadmin access required"
	errMethodNotAllowed   = "Method not allowed"
	defaultWAGatewayURL   = "http://wa-gateway:8202"
)

func requireSuperAdmin(r *http.Request) (*Claims, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}
	claims, err := validateToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || claims.Role != "superadmin" {
		return nil, false
	}
	return claims, true
}

func handleVerifierStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: errSuperadminRequired})
		return
	}

	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = defaultWAGatewayURL
	}
	verifierTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if verifierTenant == "" {
		verifierTenant = "verifier"
	}

	resp, err := probeWAStatus(waURL, verifierTenant)
	if err != nil {
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status":  "unavailable",
			"message": "WA Gateway tidak berjalan. Verifier WhatsApp tidak tersedia.",
		}})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("WA Gateway returned non-200", "status", resp.StatusCode)
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status":  "unavailable",
			"message": "WA Gateway tidak merespon dengan benar.",
		}})
		return
	}

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		slog.Warn("Failed to decode WA Gateway response", "error", err)
		writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]interface{}{
			"status":  "unavailable",
			"message": "Gagal membaca status WA Gateway.",
		}})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: status})
}

func probeWAStatus(waURL, verifierTenant string) (*http.Response, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error
	for i := 0; i < 3; i++ {
		resp, err := client.Get(waURL + "/api/wa/status?tenant_id=" + verifierTenant)
		if err != nil {
			slog.Warn("WA Gateway not reachable", "error", err)
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var tempStatus map[string]interface{}
			json.Unmarshal(bodyBytes, &tempStatus)
			if tempStatus["status"] == "delegated" {
				if i == 2 {
					tempStatus["status"] = "connected"
					newBody, _ := json.Marshal(tempStatus)
					resp.Body = io.NopCloser(bytes.NewBuffer(newBody))
					return resp, nil
				}
				slog.Warn("WA Gateway status delegated, retrying...", "attempt", i+1)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			return resp, nil
		}
		return resp, nil
	}
	return nil, lastErr
}

func handleVerifierQR(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: errSuperadminRequired})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = defaultWAGatewayURL
	}
	verifierTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if verifierTenant == "" {
		verifierTenant = "verifier"
	}
	resp, err := client.Get(waURL + "/api/wa/qr?tenant_id=" + verifierTenant)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to get QR code"})
		return
	}
	defer resp.Body.Close()

	var qrData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&qrData)

	writeJSON(w, http.StatusOK, Response{Success: true, Data: qrData})
}

func handleVerifierDisconnect(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperAdmin(r); !ok {
		writeJSON(w, http.StatusForbidden, Response{Success: false, Message: errSuperadminRequired})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = defaultWAGatewayURL
	}
	verifierTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if verifierTenant == "" {
		verifierTenant = "verifier"
	}
	req, _ := http.NewRequest(http.MethodPost, waURL+"/api/wa/logout?tenant_id="+verifierTenant, nil)
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Message: "Failed to disconnect verifier"})
		return
	}
	defer resp.Body.Close()

	writeJSON(w, http.StatusOK, Response{Success: true, Message: "Verifier disconnected. Scan QR to reconnect."})
}

func fetchWAStatusWithRetry(client *http.Client, waURL, verifierTenant string) (*http.Response, error) {
	for i := 0; i < 3; i++ {
		resp, err := client.Get(waURL + "/api/wa/status?tenant_id=" + verifierTenant)
		if err == nil {
			return resp, nil
		}
		time.Sleep(1 * time.Second)
	}
	return nil, fmt.Errorf("failed after 3 retries")
}
