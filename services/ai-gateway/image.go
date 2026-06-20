package main

import (
	"encoding/json"
	"net/http"

	"core_project/shared/sdk/auth"
)

type GenerateImageRequest struct {
	TenantID string `json:"tenant_id"`
	Prompt   string `json:"prompt"`
	Size     string `json:"size"`
}

func HandleGenerateImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	var req GenerateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
		return
	}

	// F034: Wallet gate
	tenantID := ""
	if t, ok := r.Context().Value(auth.TenantIDKey).(string); ok {
		tenantID = t
	}
	if tenantID != "" {
		auth.ConsumeWalletAddon(r.Context(), tenantID, "image_gen")
	}

	// MOCK implementation
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Success",
		Data:    map[string]any{"image_url": "https://placehold.co/1024x1024/png?text=MOCK+IMAGE+GEN"},
	})
}
