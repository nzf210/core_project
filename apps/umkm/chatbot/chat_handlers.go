package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"core_project/shared/sdk/auth"
	"log/slog"
)

type ChatReq struct {
	SessionID string `json:"session_id"` // Optional
	Message   string `json:"message"`
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	var req ChatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid body"})
		return
	}

	ctx := r.Context()

	// Manage Session
	sessionID := req.SessionID
	if sessionID == "" && DB != nil {
		err := DB.QueryRow(ctx, "INSERT INTO chat_sessions (tenant_id, title) VALUES ($1, $2) RETURNING id", tenantID, "New Chat").Scan(&sessionID)
		if err != nil {
			slog.Error("Failed to create session", "err", err)
		}
	}

	// Save User Message
	if sessionID != "" && DB != nil {
		DB.Exec(ctx, "INSERT INTO chat_messages (session_id, role, content) VALUES ($1, $2, $3)", sessionID, "user", req.Message)
	}

	tenantName := "UMKM WCH"
	if DB != nil {
		DB.QueryRow(ctx, "SELECT name FROM tenants WHERE id = $1", tenantID).Scan(&tenantName)
	}
	systemPrompt := buildSystemPrompt(ctx, tenantID, tenantName, req.Message, "owner", loadChatbotConfig(ctx, tenantID))
	// Call AI Gateway
	aiReqBody := map[string]interface{}{
		"provider":   "minimax",
		"message":    req.Message,
		"system_msg": systemPrompt,
		"tenant_id":  tenantID,
	}
	jsonBody, _ := json.Marshal(aiReqBody)

	aiReqHTTP, _ := http.NewRequestWithContext(ctx, "POST", AIGatewayURL, bytes.NewBuffer(jsonBody))
	aiReqHTTP.Header.Set("Content-Type", "application/json")
	aiReqHTTP.Header.Set("X-Tenant-ID", tenantID)

	aiClient := &http.Client{Timeout: 30 * time.Second}
	aiRespHTTP, err := aiClient.Do(aiReqHTTP)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to contact AI Gateway"})
		return
	}
	defer aiRespHTTP.Body.Close()

	var aiGatewayResp struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
	}
	json.NewDecoder(aiRespHTTP.Body).Decode(&aiGatewayResp)

	if !aiGatewayResp.Success {
		atomicAddInt64(&chatbotErrors, 1)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI Gateway returned error"})
		return
	}

	atomicAddInt64(&chatbotLLMCalls, 1)
	aiAnswer := processAIAnswer(ctx, tenantID, aiGatewayResp.Text, "Web UI", "owner")

	// Save Assistant Message
	if sessionID != "" && DB != nil {
		DB.Exec(ctx, "INSERT INTO chat_messages (session_id, role, content) VALUES ($1, $2, $3)", sessionID, "assistant", aiAnswer)
	}

	if tenantID != "" {
		auth.IncrementQuota(ctx, tenantID, "chatbot_messages", 1)
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"session_id": sessionID,
			"reply":      aiAnswer,
		},
	})
}