package main

import (
	"encoding/json"
	"net/http"

	"core_project/shared/sdk/response"
)

type N8NPayload struct {
	TenantID  string                 `json:"tenant_id"`
	Phone     string                 `json:"phone"` // For WA
	Email     string                 `json:"email"` // For Email
	Template  string                 `json:"template"`
	Data      map[string]interface{} `json:"data"`
	MediaURL  string                 `json:"media_url"`  // F028: Optional file URL
	MediaName string                 `json:"media_name"` // F028: Optional file name
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
	err := sendWAMedia(payload.TenantID, payload.Phone, message, payload.MediaURL, payload.MediaName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to queue WhatsApp notification", err)
		return
	}

	response.JSON(w, http.StatusOK, "WhatsApp notification queued", map[string]string{"message": message})
}

func handleN8NTelegram(w http.ResponseWriter, r *http.Request) {
	var payload N8NPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid JSON payload", err)
		return
	}

	if payload.TenantID == "" || payload.Template == "" { // Phone not strictly needed for Telegram if we rely on Data.chat_id, but usually it's passed differently or we use a designated chat_id
		response.Error(w, http.StatusBadRequest, "Missing required fields: tenant_id, template", nil)
		return
	}
	
	chatID := ""
	if cid, ok := payload.Data["chat_id"].(string); ok {
		chatID = cid
	} else if payload.Phone != "" {
		chatID = payload.Phone // sometimes N8N might put it here
	}
	
	if chatID == "" {
		response.Error(w, http.StatusBadRequest, "Missing chat_id for telegram", nil)
		return
	}

	message := RenderTemplate(payload.Template, payload.Data)

	err := sendTelegramMedia(chatID, message, payload.MediaURL, payload.MediaName)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to send Telegram notification", err)
		return
	}

	response.JSON(w, http.StatusOK, "Telegram notification queued", map[string]string{"message": message})
}
