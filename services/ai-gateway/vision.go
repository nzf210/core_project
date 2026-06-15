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

	if req.ImageURL == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "image_url cannot be empty"})
		return
	}

	// Get tenant from context
	_ = ""
	if t, ok := r.Context().Value(auth.TenantIDKey).(string); ok {
		_ = t
	}

	// MOCK Implementation for C1 Real Count Form Processing:
	// For now, this returns a simulated JSON extracting numbers from the C1 image
	// In production, we would use MiniMax-M3-Vision or OpenAI GPT-4o Vision API
	// and pass the image URL and the prompt.

	var respText string

	// Specific prompt mapping for Real Count C1
	if req.Prompt == "Extract C1 numbers" {
		// Mock response mimicking OCR
		respText = `{"candidate_votes": 125, "opponent_votes": 85, "invalid_votes": 2}`
	} else {
		// General fallback mock
		respText = "[MOCK VISION] Gambar diterima: " + req.ImageURL + " | Prompt: " + req.Prompt
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Success",
		Data:    map[string]interface{}{"text": respText},
	})
}
