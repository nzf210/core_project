package main

import (
	"encoding/json"
	"net/http"

	"core_project/shared/sdk/auth"
	"core_project/shared/sdk/response"
)

type TranscribeRequest struct {
	TenantID string `json:"tenant_id"`
	AudioURL string `json:"audio_url"`
	Language string `json:"language"`
}

func HandleTranscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: response.MethodNotAllowed})
		return
	}
	var req TranscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: response.InvalidRequest})
		return
	}

	// F034: Wallet gate
	tenantID := ""
	if t, ok := r.Context().Value(auth.TenantIDKey).(string); ok {
		tenantID = t
	}
	if tenantID != "" {
		auth.ConsumeWalletAddon(r.Context(), tenantID, "ai_audio_stt")
	}

	// MOCK implementation
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Success",
		Data:    map[string]any{"text": "[MOCK STT] Audio ditranskripsi dari: " + req.AudioURL},
	})
}

type SpeakRequest struct {
	TenantID string `json:"tenant_id"`
	Text     string `json:"text"`
	Voice    string `json:"voice"`
}

func HandleSpeak(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: response.MethodNotAllowed})
		return
	}
	var req SpeakRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: response.InvalidRequest})
		return
	}

	// F034: Wallet gate
	tenantID := ""
	if t, ok := r.Context().Value(auth.TenantIDKey).(string); ok {
		tenantID = t
	}
	if tenantID != "" {
		auth.ConsumeWalletAddon(r.Context(), tenantID, "ai_audio_tts")
	}

	// MOCK implementation
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Success",
		Data:    map[string]any{"audio_url": "https://example.com/mock-tts.ogg"},
	})
}
