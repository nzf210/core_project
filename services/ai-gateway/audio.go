package main

import (
    "encoding/json"
    "net/http"

    "core_project/shared/sdk/auth"
)

type TranscribeRequest struct {
    TenantID string `json:"tenant_id"`
    AudioURL string `json:"audio_url"`
    Language string `json:"language"`
}

func HandleTranscribe(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
        return
    }
    var req TranscribeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }
    
    // MOCK implementation
    respText := "[MOCK STT] Audio ditranskripsi dari: " + req.AudioURL
    
    if tenantID := r.Context().Value(auth.TenantIDKey).(string); tenantID != "" {
    }
    writeJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Message: "Success", 
		Data: map[string]interface{}{"text": respText},
	})
}

type SpeakRequest struct {
    TenantID string `json:"tenant_id"`
    Text     string `json:"text"`
    Voice    string `json:"voice"`
}

func HandleSpeak(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
        return
    }
    var req SpeakRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSON(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Invalid request"})
        return
    }
    
    // MOCK implementation
    audioURL := "https://example.com/mock-tts.ogg"
    
    if tenantID := r.Context().Value(auth.TenantIDKey).(string); tenantID != "" {
    }
    writeJSON(w, http.StatusOK, APIResponse{
		Success: true, 
		Message: "Success", 
		Data: map[string]interface{}{"audio_url": audioURL},
	})
}
