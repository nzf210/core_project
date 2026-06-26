package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// handleWASetup returns the current WA provider status for the setup UI.

const (
	headerTenantID = "X-Tenant-ID"
	errMissingTenantID = "Missing X-Tenant-ID"
)

func handleWASetup(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantID})
		return
		}
		ctx := r.Context()

	var waProviderPref string
	_ = DB.QueryRow(ctx, "SELECT wa_provider_preference FROM tenant_chatbot_configs WHERE tenant_id = $1", tenantID).Scan(&waProviderPref)
	if waProviderPref == "" {
		waProviderPref = "auto"
	}

	var whatsmeowStatus struct {
		Connected bool   `json:"connected"`
		Status    string `json:"status"`
	}
	var wmStatus string
	_ = DB.QueryRow(ctx, "SELECT status FROM wa_sessions WHERE tenant_id = $1", tenantID).Scan(&wmStatus)
	switch wmStatus {
	case "connected":
		whatsmeowStatus.Connected = true
		whatsmeowStatus.Status = "connected"
	case "qr_pending":
		whatsmeowStatus.Status = "qr_pending"
	default:
		whatsmeowStatus.Status = "disconnected"
	}

	var cloudAPIStatus struct {
		Active    bool   `json:"active"`
		CreditBal int64  `json:"credit_balance_rupiah"`
		LastSync  string `json:"last_sync_at"`
	}
	var hasCloudAPI bool
	_ = DB.QueryRow(ctx, "SELECT is_active FROM wa_cloud_api_credentials WHERE tenant_id = $1", tenantID).Scan(&hasCloudAPI)
	if hasCloudAPI {
		cloudAPIStatus.Active = true
	}

	var plan string
	_ = DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan)
	var hasWaCloudAPIAddon bool
	_ = DB.QueryRow(ctx, "SELECT is_enabled FROM plan_features WHERE plan_id = $1 AND feature_key = 'wa_cloud_api'", plan).Scan(&hasWaCloudAPIAddon)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"wa_provider_preference": waProviderPref,
			"whatsmeow":             whatsmeowStatus,
			"cloud_api":             cloudAPIStatus,
			"has_cloud_api_addon":    hasWaCloudAPIAddon,
			"can_use_cloud_api":      hasWaCloudAPIAddon,
		},
	})
}

func handleWAConnect(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantID})
		return
	}

	var body struct {
		Provider string `json:"provider"`
		Action   string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	if body.Provider != "" && body.Provider != "auto" && body.Provider != "whatsmeow" && body.Provider != "cloud_api" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Provider must be auto, whatsmeow, or cloud_api"})
		return
	}

	ctx := r.Context()

	if body.Provider == "cloud_api" {
		var plan string
		_ = DB.QueryRow(ctx, "SELECT plan FROM tenants WHERE id = $1", tenantID).Scan(&plan)
		var hasAddon bool
		_ = DB.QueryRow(ctx, "SELECT is_enabled FROM plan_features WHERE plan_id = $1 AND feature_key = 'wa_cloud_api'", plan).Scan(&hasAddon)
		if !hasAddon {
			writeJSON(w, http.StatusForbidden, APIResponse{Message: "Paket Anda tidak mendukung WA Cloud API. Silakan upgrade paket untuk menggunakan fitur ini."})
			return
		}
	}
	_, err := DB.Exec(ctx, `
		INSERT INTO tenant_chatbot_configs (tenant_id, wa_provider_preference)
		VALUES ($1, $2)
		ON CONFLICT (tenant_id)
		DO UPDATE SET wa_provider_preference = $2, updated_at = NOW()
	`, tenantID, body.Provider)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update provider"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Provider preference updated",
		Data: map[string]interface{}{
			"wa_provider_preference": body.Provider,
		},
	})
}

// handleWACloudAPICredential handles GET/POST for Meta Cloud API credentials.
func handleWACloudAPICredential(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantID})
		return
	}
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		handleWACloudAPICredentialGet(ctx, w, tenantID)

	case http.MethodPost:
		handleWACloudAPICredentialPost(ctx, w, r, tenantID)
	}
}

func handleWACloudAPICredentialGet(ctx context.Context, w http.ResponseWriter, tenantID string) {
	var cred struct {
		ID                 string `json:"id"`
		PhoneNumberID      string `json:"phone_number_id"`
		WABAID             string `json:"waba_id"`
		VerifyToken        string `json:"verify_token"`
		IsActive           bool   `json:"is_active"`
		VerificationStatus string `json:"verification_status"`
		VerifiedAt         string `json:"verified_at"`
		CreatedAt          string `json:"created_at"`
		UpdatedAt          string `json:"updated_at"`
	}
	err := DB.QueryRow(ctx, `
		SELECT id, phone_number_id, COALESCE(waba_id,''), COALESCE(verify_token,''),
			   is_active, COALESCE(verification_status,'unverified'),
			   COALESCE(verified_at::text,''), created_at::text, updated_at::text
		FROM wa_cloud_api_credentials WHERE tenant_id = $1
	`, tenantID).Scan(&cred.ID, &cred.PhoneNumberID, &cred.WABAID, &cred.VerifyToken,
		&cred.IsActive, &cred.VerificationStatus, &cred.VerifiedAt, &cred.CreatedAt, &cred.UpdatedAt)
	if err != nil {
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: nil})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: cred})
}

func handleWACloudAPICredentialPost(ctx context.Context, w http.ResponseWriter, r *http.Request, tenantID string) {
	var body struct {
		PhoneNumberID string `json:"phone_number_id"`
		WABAID        string `json:"waba_id"`
		AccessToken   string `json:"access_token"`
		VerifyToken   string `json:"verify_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}
	body.PhoneNumberID = strings.TrimSpace(body.PhoneNumberID)
	body.AccessToken = strings.TrimSpace(body.AccessToken)
	if body.PhoneNumberID == "" || body.AccessToken == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "phone_number_id dan access_token wajib diisi"})
		return
	}
	if body.VerifyToken == "" {
		body.VerifyToken = tenantID
	}

	_, err := DB.Exec(ctx, `
		INSERT INTO wa_cloud_api_credentials (tenant_id, phone_number_id, waba_id, access_token, verify_token)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id) DO UPDATE SET
			phone_number_id = EXCLUDED.phone_number_id,
			waba_id = EXCLUDED.waba_id,
			access_token = EXCLUDED.access_token,
			verify_token = EXCLUDED.verify_token,
			is_active = true,
			verification_status = 'unverified',
			updated_at = NOW()
	`, tenantID, body.PhoneNumberID, body.WABAID, body.AccessToken, body.VerifyToken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal menyimpan credential: " + err.Error()})
		return
	}

	validateCloudAPICredentialsAfterSave(ctx, w, tenantID, body.PhoneNumberID, body.WABAID, body.AccessToken)
}

func validateCloudAPICredentialsAfterSave(ctx context.Context, w http.ResponseWriter, tenantID, phoneID, wabaID, token string) {
	cloudAPIHost := "http://localhost:8210"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		cloudAPIHost = "http://wa-cloud-api:8210"
	}
	vaURL := cloudAPIHost + "/validate"
	vaBody, _ := json.Marshal(map[string]interface{}{
		"access_token":    token,
		"phone_number_id": phoneID,
		"waba_id":         wabaID,
	})
	vaReq, _ := http.NewRequest(http.MethodPost, vaURL, bytes.NewReader(vaBody))
	vaReq.Header.Set("Content-Type", "application/json")
	vaReq.Header.Set(headerTenantID, tenantID)

	vaResp, err := http.DefaultClient.Do(vaReq)
	verificationStatus := "unverified"
	if err == nil {
		defer vaResp.Body.Close()
		if vaResp.StatusCode == http.StatusOK {
			verificationStatus = "verified"
		} else {
			verificationStatus = "error"
		}
	}

	if verificationStatus == "verified" {
		_, _ = DB.Exec(ctx, `UPDATE wa_cloud_api_credentials SET verification_status = $1, verified_at = NOW(), last_checked_at = NOW() WHERE tenant_id = $2`, verificationStatus, tenantID)
	} else {
		_, _ = DB.Exec(ctx, `UPDATE wa_cloud_api_credentials SET verification_status = $1, last_checked_at = NOW(), check_error = $3 WHERE tenant_id = $2`, verificationStatus, tenantID, "Gagal terhubung ke Meta API untuk validasi")
	}

	var respMsg string
	switch verificationStatus {
	case "verified":
		respMsg = "Cloud API credential tersimpan & terverifikasi!"
	case "error":
		respMsg = "Cloud API credential tersimpan. Validasi gagal — periksa credential Anda."
	default:
		respMsg = "Cloud API credential tersimpan"
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: respMsg})
}
