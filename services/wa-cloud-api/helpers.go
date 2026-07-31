package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"core_project/shared/sdk/config"
	"core_project/shared/sdk/encryption"
)

func getCredential(ctx context.Context, tenantID string) (*CloudAPICredential, error) {
	var cred CloudAPICredential
	var encryptedToken string
	err := DB.QueryRow(ctx, `
		SELECT id, tenant_id, phone_number_id, COALESCE(waba_id, ''), access_token, verify_token, is_active, created_at, updated_at
		FROM wa_cloud_api_credentials WHERE tenant_id = $1 AND is_active = true
	`, tenantID).Scan(&cred.ID, &cred.TenantID, &cred.PhoneNumberID, &cred.WABAID, &encryptedToken, &cred.VerifyToken, &cred.IsActive, &cred.CreatedAt, &cred.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Decrypt access token using EncryptionKey from config
	if encryptedToken != "" {
		decrypted, err := encryption.Decrypt(encryptedToken, config.GlobalConfig.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt access_token: %w", err)
		}
		cred.AccessToken = decrypted
	}

	return &cred, nil
}

func verifyWebhookToken(ctx context.Context, token string) bool {
	var expected string
	err := DB.QueryRow(ctx, `SELECT verify_token FROM wa_cloud_api_credentials WHERE verify_token = $1 LIMIT 1`, token).Scan(&expected)
	return err == nil && token == expected
}

func sendToMeta(ctx context.Context, phoneNumberID, accessToken string, payload MetaSendPayload) (*MetaResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s/messages", graphBaseURL, graphVersion, phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set(headerContentType, contentTypeJSON)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("meta API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result MetaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func normalizeTo(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}
	if !strings.HasPrefix(phone, "62") {
		phone = "62" + phone
	}
	return phone
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
