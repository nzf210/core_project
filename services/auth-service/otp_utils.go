package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func sendWAGatewayOTP(senderTenant, authWAProvider, target, otp string) {
	formData := url.Values{}
	formData.Set("tenant_id", senderTenant)
	formData.Set("target", target)
	formData.Set("message", "Kode OTP registrasi WCH Anda: "+otp)

	waURL := os.Getenv("WA_GATEWAY_URL")
	if waURL == "" {
		waURL = waGatewayDefault
	}

	var resp *http.Response
	var err error
	for i := 0; i < 3; i++ {
		payload := strings.NewReader(formData.Encode())
		req, _ := http.NewRequestWithContext(context.Background(), "POST", waURL+"/api/wa/send", payload)
		req.Header.Set(headerContentType, contentTypeFormURLEncoded)
		req.Header.Set("X-Message-Type", "otp")
		req.Header.Set("X-Source", "auth-service")
		req.Header.Set("X-WA-Provider-Override", authWAProvider)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Failed to send OTP via WA Gateway", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if resp.StatusCode == 409 {
			resp.Body.Close()
			slog.Warn("WA Gateway returned 409 (delegated), retrying...", "attempt", i+1)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}

	if err != nil {
		slog.Error("Failed to send OTP after retries", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("WA Gateway rejected OTP", "status", resp.StatusCode, "body", string(body))
	}
}

func getSystemTenantID() string {
	systemTenant := os.Getenv("WA_SYSTEM_TENANT_ID")
	if systemTenant == "" {
		systemTenant = "system"
	}
	return systemTenant
}

func generateOTP() string {
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}
