package handlers

import (
	"encoding/json"
	"net/http"
)

type BotFAQRequest struct {
	Question string `json:"question"`
}

// HandleBotFAQ simulates pgvector RAG for campaign guidelines
func HandleBotFAQ(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Unauthorized - Tenant ID missing",
		})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req BotFAQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	mockAnswer := "Berdasarkan visi-misi kandidat, fokus utama di desa ini adalah perbaikan irigasi dan subsidi pupuk gratis. (Sumber: Dokumen Visi-Misi Bab 3)"

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"question": req.Question,
			"answer":   mockAnswer,
		},
	})
}
