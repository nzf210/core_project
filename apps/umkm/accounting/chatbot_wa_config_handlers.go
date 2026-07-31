package main

import (
	"context"
	"encoding/json"
	"net/http"
	"core_project/shared/sdk/response"
)

func handleChatbotConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		getChatbotConfig(w, r, tenantID)
	case http.MethodPut:
		updateChatbotConfig(w, r, tenantID)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
	}
}

func getChatbotConfig(w http.ResponseWriter, r *http.Request, tenantID string) {
	cfg, err := loadChatbotConfigByTenant(r.Context(), tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal memuat konfigurasi: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: cfg})
}

func updateChatbotConfig(w http.ResponseWriter, r *http.Request, tenantID string) {
	current, err := loadChatbotConfigByTenant(r.Context(), tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal memuat konfigurasi: " + err.Error()})
		return
	}

	var body ChatbotConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Body tidak valid: " + err.Error()})
		return
	}

	merged := mergeChatbotConfig(current, &body)

	if msg := validateChatbotConfig(&merged); msg != "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: msg})
		return
	}

	if merged.IsActive {
		if err := validateWAConnectionForChatbot(r.Context(), DB, tenantID); err != nil {
			writeJSON(w, http.StatusBadRequest, APIResponse{Message: err.Error()})
			return
		}
	}

	if err := saveChatbotConfig(r.Context(), tenantID, &merged); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal update: " + err.Error()})
		return
	}

	cleanupWAProviderData(r.Context(), tenantID, merged.WAProviderPreference)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Konfigurasi chatbot berhasil diperbarui",
		Data:    merged,
	})
}

func mergeStr(merged, body *string) {
	if *body != "" {
		*merged = *body
	}
}

func mergeChatbotConfig(current, body *ChatbotConfig) ChatbotConfig {
	merged := *current
	mergeStr(&merged.LLMProvider, &body.LLMProvider)
	mergeStr(&merged.LLMModel, &body.LLMModel)
	mergeStr(&merged.SystemPrompt, &body.SystemPrompt)
	mergeStr(&merged.Tone, &body.Tone)
	mergeStr(&merged.Language, &body.Language)
	mergeStr(&merged.WelcomeMessage, &body.WelcomeMessage)
	mergeStr(&merged.FallbackMessage, &body.FallbackMessage)
	mergeStr(&merged.OutsideHoursMessage, &body.OutsideHoursMessage)
	mergeStr(&merged.BusinessHoursStart, &body.BusinessHoursStart)
	mergeStr(&merged.BusinessHoursEnd, &body.BusinessHoursEnd)
	mergeStr(&merged.WAProviderPreference, &body.WAProviderPreference)
	mergeStr(&merged.VoiceModel, &body.VoiceModel)
	if body.Temperature != 0 {
		merged.Temperature = body.Temperature
	}
	if body.MaxTokens != 0 {
		merged.MaxTokens = body.MaxTokens
	}
	if body.MaxContextMessages != 0 {
		merged.MaxContextMessages = body.MaxContextMessages
	}
	if body.AutoEscalateAfterMinutes != 0 {
		merged.AutoEscalateAfterMinutes = body.AutoEscalateAfterMinutes
	}
	if body.RAGTopK != 0 {
		merged.RAGTopK = body.RAGTopK
	}
	if body.RAGSimilarityThreshold != 0 {
		merged.RAGSimilarityThreshold = body.RAGSimilarityThreshold
	}
	if body.BusinessDays != nil {
		merged.BusinessDays = body.BusinessDays
	}
	if body.EscalationKeywords != nil {
		merged.EscalationKeywords = body.EscalationKeywords
	}
	if body.ChannelsEnabled != nil {
		merged.ChannelsEnabled = body.ChannelsEnabled
	}
	merged.EscalationEnabled = body.EscalationEnabled
	merged.RAGEnabled = body.RAGEnabled
	merged.IsActive = body.IsActive
	merged.EnableVision = body.EnableVision
	merged.EnableVoiceReply = body.EnableVoiceReply
	return merged
}

func saveChatbotConfig(ctx context.Context, tenantID string, cfg *ChatbotConfig) error {
	kwJSON, _ := json.Marshal(cfg.EscalationKeywords)
	daysJSON, _ := json.Marshal(cfg.BusinessDays)
	channelsJSON, _ := json.Marshal(cfg.ChannelsEnabled)

	_, err := DB.Exec(ctx, `
		UPDATE tenant_chatbot_configs SET
			llm_provider = $1, llm_model = $2, temperature = $3, max_tokens = $4,
			system_prompt = $5, tone = $6, language = $7, max_context_messages = $8,
			welcome_message = $9, fallback_message = $10, outside_hours_message = $11,
			business_hours_start = $12, business_hours_end = $13, business_days = $14,
			escalation_enabled = $15, escalation_keywords = $16,
			escalation_confidence_threshold = $17, auto_escalate_after_minutes = $18,
			rag_enabled = $19, rag_top_k = $20, rag_similarity_threshold = $21,
			channels_enabled = $22, is_active = $23, enable_vision = $24, enable_voice_reply = $25, voice_model = $26,
			wa_provider_preference = $27, updated_at = NOW()
		WHERE tenant_id = $28
	`, cfg.LLMProvider, cfg.LLMModel, cfg.Temperature, cfg.MaxTokens,
		nullString(cfg.SystemPrompt), cfg.Tone, cfg.Language, cfg.MaxContextMessages,
		cfg.WelcomeMessage, cfg.FallbackMessage, cfg.OutsideHoursMessage,
		cfg.BusinessHoursStart, cfg.BusinessHoursEnd, daysJSON,
		cfg.EscalationEnabled, kwJSON, cfg.EscalationConfidenceThreshold,
		cfg.AutoEscalateAfterMinutes, cfg.RAGEnabled, cfg.RAGTopK, cfg.RAGSimilarityThreshold,
		channelsJSON, cfg.IsActive, cfg.EnableVision, cfg.EnableVoiceReply, cfg.VoiceModel,
		cfg.WAProviderPreference, tenantID)
	return err
}

func cleanupWAProviderData(ctx context.Context, tenantID, preference string) {
	switch preference {
	case "whatsmeow":
		DB.Exec(ctx, `DELETE FROM wa_cloud_api_credentials WHERE tenant_id = $1`, tenantID)
	case "cloud_api":
		DB.Exec(ctx, `DELETE FROM wa_sessions WHERE tenant_id = $1`, tenantID)
	}
}

func handleChatbotPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: response.MethodNotAllowed})
		return
	}
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	var plan string
	err := DB.QueryRow(r.Context(), `SELECT plan FROM tenants WHERE id = $1`, tenantID).Scan(&plan)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to read tenant plan: " + err.Error()})
		return
	}

	var enabled bool
	err = DB.QueryRow(r.Context(), `
		SELECT is_enabled FROM plan_features
		WHERE plan_id = $1 AND feature_key = 'wa_cloud_api'
	`, plan).Scan(&enabled)

	hasWaCloudAPI := err == nil && enabled

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"plan":                plan,
			"has_wa_cloud_api":    hasWaCloudAPI,
			"available_providers": []string{"auto", "whatsmeow", "cloud_api"},
		},
	})
}
