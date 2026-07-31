package main

import (
	"encoding/json"
	"net/http"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
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
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: response.InvalidRequest})
		return
	}

	// F034: Wallet gate (image_gen) + F050: per-modality ai_image counter
	tenantID := ""
	if t, ok := r.Context().Value(auth.TenantIDKey).(string); ok {
		tenantID = t
	}
	if tenantID != "" {
		// Increment quota first (best-effort) so quota drift doesn't outlive wallet state.
		_, _, _ = auth.IncrementQuota(r.Context(), tenantID, "ai_image", 1)
		auth.ConsumeWalletAddon(r.Context(), tenantID, "image_gen")
	}

	// MOCK implementation
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Success",
		Data:    map[string]any{"image_url": "https://placehold.co/1024x1024/png?text=MOCK+IMAGE+GEN"},
	})
}
