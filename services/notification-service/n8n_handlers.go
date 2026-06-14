package main

import (
	"encoding/json"
	"net/http"

	"core_project/shared/sdk/response"
)

type N8NPayload struct {
	TenantID string                 `json:"tenant_id"`
	Phone    string                 `json:"phone"` // For WA
	Email    string                 `json:"email"` // For Email
	Template string                 `json:"template"`
	Data     map[string]interface{} `json:"data"`
}

func handleN8NWhatsApp(w http.ResponseWriter, r *http.Request) {
	var payload N8NPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	if payload.TenantID == "" || payload.Phone == "" || payload.Template == "" {
		response.Error(w, http.StatusBadRequest, "Missing required fields: tenant_id, phone, template", nil)
		return
	}

	// Render message using the template engine
	message := RenderTemplate(payload.Template, payload.Data)

	// Send to WA Gateway queue (mimic existing sendWA logic in notification-service)
	err := sendWA(payload.TenantID, payload.Phone, message)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to queue WhatsApp notification", err)
		return
	}

	response.JSON(w, http.StatusOK, "WhatsApp notification queued", map[string]string{"message": message})
}
