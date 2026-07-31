package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"core_project/shared/sdk/response"
)


func handleInternalChatbotConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing tenant_id"})
		return
	}

	rows, err := DB.Query(r.Context(),
		`SELECT llm_provider, llm_model, temperature, max_tokens, system_prompt,
		        tone, language, max_context_messages, welcome_message, fallback_message,
			outside_hours_message, business_hours_start, business_hours_end, business_days,
			escalation_enabled, escalation_keywords, escalation_confidence_threshold,
			auto_escalate_after_minutes, rag_enabled, rag_top_k, rag_similarity_threshold,
			channels_enabled, is_active, enable_vision, enable_voice_reply, voice_model, wa_provider_preference
		 FROM tenant_chatbot_configs WHERE tenant_id = $1`, tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: response.DBError})
		return
	}
	defer rows.Close()

	if !rows.Next() {
		writeJSON(w, http.StatusNotFound, APIResponse{Message: "Chatbot config not found"})
		return
	}

	var cfg ChatbotConfig
	var sysPrompt, welcome, fallback, outsideHrs *string
	var escalationKW []string
	var bizHoursStart, bizHoursEnd time.Time
	if err := rows.Scan(
		&cfg.LLMProvider, &cfg.LLMModel, &cfg.Temperature, &cfg.MaxTokens,
		&sysPrompt, &cfg.Tone, &cfg.Language, &cfg.MaxContextMessages,
		&welcome, &fallback, &outsideHrs,
		&bizHoursStart, &bizHoursEnd, &cfg.BusinessDays,
		&cfg.EscalationEnabled, &escalationKW, &cfg.EscalationConfidenceThreshold,
		&cfg.AutoEscalateAfterMinutes, &cfg.RAGEnabled, &cfg.RAGTopK, &cfg.RAGSimilarityThreshold,
		&cfg.ChannelsEnabled, &cfg.IsActive, &cfg.EnableVision, &cfg.EnableVoiceReply, &cfg.VoiceModel,
		&cfg.WAProviderPreference,
	); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Scan error"})
		return
	}
	cfg.BusinessHoursStart = bizHoursStart.Format("15:04:05")
	cfg.BusinessHoursEnd = bizHoursEnd.Format("15:04:05")

	if sysPrompt != nil {
		cfg.SystemPrompt = *sysPrompt
	}
	if welcome != nil {
		cfg.WelcomeMessage = *welcome
	}
	if fallback != nil {
		cfg.FallbackMessage = *fallback
	}
	if outsideHrs != nil {
		cfg.OutsideHoursMessage = *outsideHrs
	}
	cfg.EscalationKeywords = escalationKW

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: cfg})
}

func handleChatbotConfigTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}
	tenantID := r.Header.Get(response.XTenantID)
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Body tidak valid"})
		return
	}
	if strings.TrimSpace(body.Message) == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "message wajib diisi"})
		return
	}

	cfg, err := loadChatbotConfigByTenant(r.Context(), tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal memuat konfigurasi: " + err.Error()})
		return
	}

	// Render the system prompt the same way the chatbot runtime does.
	systemPrompt := renderSystemPromptFromConfig(cfg, tenantID, "UMKM WCH", "owner")

	// Check escalation keywords (case-insensitive substring).
	msgLower := strings.ToLower(body.Message)
	wouldEscalate := false
	if cfg.EscalationEnabled {
		for _, kw := range cfg.EscalationKeywords {
			if strings.Contains(msgLower, strings.ToLower(kw)) {
				wouldEscalate = true
				break
			}
		}
	}

	// Call AI Gateway (best-effort: if AI is down, return error but still report
	// escalation state so the FE preview stays useful).
	aiReqBody, _ := json.Marshal(map[string]interface{}{
		"provider":   cfg.LLMProvider,
		"message":    body.Message,
		"system_msg": systemPrompt,
		"tenant_id":  tenantID,
	})
	aiReq, _ := http.NewRequestWithContext(r.Context(), "POST", AIGatewayURL, bytes.NewBuffer(aiReqBody))
	aiReq.Header.Set("Content-Type", "application/json")
	aiReq.Header.Set(response.XTenantID, tenantID)
	client := &http.Client{Timeout: 25 * time.Second}
	aiResp, err := client.Do(aiReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, APIResponse{Message: "AI Gateway tidak dapat dihubungi: " + err.Error()})
		return
	}
	defer aiResp.Body.Close()
	var aiBody struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
	}
	json.NewDecoder(aiResp.Body).Decode(&aiBody)
	if !aiBody.Success {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "AI Gateway mengembalikan error"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"reply":          aiBody.Text,
			"would_escalate": wouldEscalate,
			"system_prompt":  systemPrompt,
		},
	})
}
