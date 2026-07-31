package main

import (
	"encoding/json"
	"net/http"
	"core_project/shared/sdk/response"
)

const (
	headerTenantSettings = "X-Tenant-ID"
)

func handleSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get(headerTenantSettings)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		settingsGet(w, r, tenantID)
	case http.MethodPut:
		settingsPut(w, r, tenantID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func settingsGet(w http.ResponseWriter, r *http.Request, tenantID string) {
	var waNumber, xenditApiKey, xenditWebhookToken, xenditMerchantID, reportTime *string
	var qrisEnabled, reportEnabled *bool
	err := DB.QueryRow(r.Context(),
		`SELECT wa_number, xendit_api_key, xendit_webhook_token, xendit_merchant_id,
		 qris_enabled, report_enabled, report_time FROM tenants WHERE id = $1`,
		tenantID).Scan(&waNumber, &xenditApiKey, &xenditWebhookToken, &xenditMerchantID, &qrisEnabled, &reportEnabled, &reportTime)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: response.DBError})
		return
	}

	ptrOr := func(v *string, dflt string) string {
		if v != nil {
			return *v
		}
		return dflt
	}
	ptrOrBool := func(v *bool, dflt bool) bool {
		if v != nil {
			return *v
		}
		return dflt
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"wa_number":            ptrOr(waNumber, ""),
			"xendit_api_key":      ptrOr(xenditApiKey, ""),
			"xendit_webhook_token": ptrOr(xenditWebhookToken, ""),
			"xendit_merchant_id":  ptrOr(xenditMerchantID, ""),
			"qris_enabled":         ptrOrBool(qrisEnabled, false),
			"report_enabled":       ptrOrBool(reportEnabled, false),
			"report_time":         ptrOr(reportTime, "07:00"),
		},
	})
}

func settingsPut(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req struct {
		WaNumber           string `json:"wa_number"`
		XenditApiKey       string `json:"xendit_api_key"`
		XenditWebhookToken string `json:"xendit_webhook_token"`
		XenditMerchantID   string `json:"xendit_merchant_id"`
		QrisEnabled        bool   `json:"qris_enabled"`
		ReportEnabled      bool   `json:"report_enabled"`
		ReportTime         string `json:"report_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	var existingWNum *string
	_ = DB.QueryRow(r.Context(), "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&existingWNum)
	if req.WaNumber == "" && existingWNum != nil {
		req.WaNumber = *existingWNum
	}

	_, err := DB.Exec(r.Context(),
		`UPDATE tenants SET wa_number = $1, xendit_api_key = $2, xendit_webhook_token = $3,
		 xendit_merchant_id = $4, qris_enabled = $5, report_enabled = $6, report_time = $7, updated_at = NOW()
		 WHERE id = $8`,
		req.WaNumber, req.XenditApiKey, req.XenditWebhookToken, req.XenditMerchantID,
		req.QrisEnabled, req.ReportEnabled, req.ReportTime, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update settings"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Settings updated successfully"})
}
