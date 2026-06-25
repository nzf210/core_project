package main

import (
	"encoding/json"
	"net/http"
)


func handleSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}
	if DB == nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB not initialized"})
		return
	}

	if r.Method == http.MethodGet {
		var waNumber, xenditApiKey, xenditWebhookToken, xenditMerchantID, reportTime *string
		var qrisEnabled, reportEnabled *bool
		err := DB.QueryRow(r.Context(), "SELECT wa_number, xendit_api_key, xendit_webhook_token, xendit_merchant_id, qris_enabled, report_enabled, report_time FROM tenants WHERE id = $1", tenantID).Scan(&waNumber, &xenditApiKey, &xenditWebhookToken, &xenditMerchantID, &qrisEnabled, &reportEnabled, &reportTime)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "DB error"})
			return
		}

		wNum, xApiKey, xWebToken, xMerchantID, rTime := "", "", "", "", "07:00"
		qEnabled, rEnabled := false, false
		if waNumber != nil {
			wNum = *waNumber
		}
		if xenditApiKey != nil {
			xApiKey = *xenditApiKey
		}
		if xenditWebhookToken != nil {
			xWebToken = *xenditWebhookToken
		}
		if xenditMerchantID != nil {
			xMerchantID = *xenditMerchantID
		}
		if qrisEnabled != nil {
			qEnabled = *qrisEnabled
		}
		if reportEnabled != nil {
			rEnabled = *reportEnabled
		}
		if reportTime != nil {
			rTime = *reportTime
		}

		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"wa_number":             wNum,
				"xendit_api_key":        xApiKey,
				"xendit_webhook_token":  xWebToken,
				"xendit_merchant_id":    xMerchantID,
				"qris_enabled":          qEnabled,
				"report_enabled":        rEnabled,
				"report_time":           rTime,
			},
		})
		return
	}

	if r.Method == http.MethodPut {
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

		// Preserve existing WhatsApp number if not provided by frontend
		var existingWNum *string
		_ = DB.QueryRow(r.Context(), "SELECT wa_number FROM tenants WHERE id = $1", tenantID).Scan(&existingWNum)
		if req.WaNumber == "" && existingWNum != nil {
			req.WaNumber = *existingWNum
		}

		_, err := DB.Exec(r.Context(), "UPDATE tenants SET wa_number = $1, xendit_api_key = $2, xendit_webhook_token = $3, xendit_merchant_id = $4, qris_enabled = $5, report_enabled = $6, report_time = $7, updated_at = NOW() WHERE id = $8", req.WaNumber, req.XenditApiKey, req.XenditWebhookToken, req.XenditMerchantID, req.QrisEnabled, req.ReportEnabled, req.ReportTime, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to update settings"})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Settings updated successfully"})
		return
	}
}