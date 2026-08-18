package main

import (
	"encoding/json"
	"net/http"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

type VisionRequest struct {
	TenantID string `json:"tenant_id"`
	ImageURL string `json:"image_url"`
	Prompt   string `json:"prompt"`
	Model    string `json:"model"`
}

func HandleVision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: response.MethodNotAllowed})
		return
	}
	var req VisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: response.InvalidRequest})
		return
	}
	if req.ImageURL == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "image_url cannot be empty"})
		return
	}

	// F034: Wallet gate — get tenant from context (set by tenantContextMiddleware)
	tenantID := ""
	if t, ok := r.Context().Value(auth.TenantIDKey).(string); ok {
		tenantID = t
	}
	if tenantID != "" {
		price := auth.AddonPricePerUnit(r.Context(), "ai_vision")
		if price > 0 && !auth.CheckWalletBalance(r.Context(), tenantID, price) {
			writeJSON(w, http.StatusPaymentRequired, APIResponse{
				Success: false,
				Message: "Insufficient wallet balance for AI Vision. Please top up.",
				Data:    map[string]any{"wallet_url": "/wallet"},
			})
			return
		}
		auth.ConsumeWalletAddon(r.Context(), tenantID, "ai_vision")
	}

	// MOCK Implementation for C1 Real Count Form Processing.
	// In production: Claude Sonnet Vision or OpenAI GPT-4o Vision API.
	var respText string
	switch req.Prompt {
	case "Extract C1 numbers":
		respText = `{"candidate_votes": 125, "opponent_votes": 85, "invalid_votes": 2}`
	case "Extract KTP data":
		respText = `{"nik": "3171234567890123", "name": "BUDI SANTOSO", "address": "JL. MERDEKA NO 45", "gender": "LAKI-LAKI", "age": 35}`
	default:
		respText = "[MOCK VISION] Gambar diterima: " + req.ImageURL + " | Prompt: " + req.Prompt
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Success",
		Data:    map[string]any{"text": respText},
	})
}
