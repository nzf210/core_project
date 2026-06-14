package main

import (
    "encoding/json"
    "net/http"

    "core_project/shared/sdk/auth"
)

type VisionRequest struct {
    TenantID string `json:"tenant_id"`
    ImageURL string `json:"image_url"`
    Prompt   string `json:"prompt"`
    Model    string `json:"model"`
}

func HandleVision(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
        return
    }
    var req VisionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }
    
    // MOCK implementation for now
    respText := "[MOCK VISION] Gambar diterima: " + req.ImageURL + " | Prompt: " + req.Prompt
    
    if tenantID := r.Context().Value(auth.TenantIDKey).(string); tenantID != "" {
        auth.IncrementQuota(r.Context(), tenantID, "ai_vision", 1)
    }
    writeJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Message: "Success", 
		Data: map[string]interface{}{"text": respText},
	})
}
