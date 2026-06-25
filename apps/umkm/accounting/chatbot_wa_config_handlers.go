package main

import (
	"encoding/json"
	"net/http"
)

// handleChatbotConfig handles GET/PUT for /chatbot/config.
func handleChatbotConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		cfg, err := loadChatbotConfigByTenant(r.Context(), tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal memuat konfigurasi: " + err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: cfg})

	case http.MethodPut:
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
		merged := *current
		if body.LLMProvider != "" {
			merged.LLMProvider = body.LLMProvider
		}
		if body.LLMModel != "" {
			merged.LLMModel = body.LLMModel
		}
		if body.Temperature != 0 {
			merged.Temperature = body.Temperature
		}
		if body.MaxTokens != 0 {
			merged.MaxTokens = body.MaxTokens
		}
		if body.SystemPrompt != "" {
			merged.SystemPrompt = body.SystemPrompt
		}
		if body.Tone != "" {
			merged.Tone = body.Tone
		}
		if body.Language != "" {
			merged.Language = body.Language
		}
		if body.MaxContextMessages != 0 {
			merged.MaxContextMessages = body.MaxContextMessages
		}
		if body.WelcomeMessage != "" {
			merged.WelcomeMessage = body.WelcomeMessage
		}
		if body.FallbackMessage != "" {
			merged.FallbackMessage = body.FallbackMessage
		}
		if body.OutsideHoursMessage != "" {
			merged.OutsideHoursMessage = body.OutsideHoursMessage
		}
		if body.BusinessHoursStart != "" {
			merged.BusinessHoursStart = body.BusinessHoursStart
		}
		if body.BusinessHoursEnd != "" {
			merged.BusinessHoursEnd = body.BusinessHoursEnd
		}
		if body.BusinessDays != nil {
			merged.BusinessDays = body.BusinessDays
		}
		if body.EscalationKeywords != nil {
			merged.EscalationKeywords = body.EscalationKeywords
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
		if body.ChannelsEnabled != nil {
			merged.ChannelsEnabled = body.ChannelsEnabled
		}
		if body.WAProviderPreference != "" {
			merged.WAProviderPreference = body.WAProviderPreference
		}
		merged.EscalationEnabled = body.EscalationEnabled
		merged.RAGEnabled = body.RAGEnabled
		merged.IsActive = body.IsActive
		merged.EnableVision = body.EnableVision
		merged.EnableVoiceReply = body.EnableVoiceReply
		if body.VoiceModel != "" {
			merged.VoiceModel = body.VoiceModel
		}

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

		kwJSON, _ := json.Marshal(merged.EscalationKeywords)
		daysJSON, _ := json.Marshal(merged.BusinessDays)
		channelsJSON, _ := json.Marshal(merged.ChannelsEnabled)

		_, err = DB.Exec(r.Context(), `
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
		`, merged.LLMProvider, merged.LLMModel, merged.Temperature, merged.MaxTokens,
			nullString(merged.SystemPrompt), merged.Tone, merged.Language, merged.MaxContextMessages,
			merged.WelcomeMessage, merged.FallbackMessage, merged.OutsideHoursMessage,
			merged.BusinessHoursStart, merged.BusinessHoursEnd, daysJSON,
			merged.EscalationEnabled, kwJSON, merged.EscalationConfidenceThreshold,
			merged.AutoEscalateAfterMinutes, merged.RAGEnabled, merged.RAGTopK, merged.RAGSimilarityThreshold,
			channelsJSON, merged.IsActive, merged.EnableVision, merged.EnableVoiceReply, merged.VoiceModel,
			merged.WAProviderPreference, tenantID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Gagal update: " + err.Error()})
			return
		}

		if merged.WAProviderPreference == "whatsmeow" {
			_, _ = DB.Exec(r.Context(), `DELETE FROM wa_cloud_api_credentials WHERE tenant_id = $1`, tenantID)
		} else if merged.WAProviderPreference == "cloud_api" {
			_, _ = DB.Exec(r.Context(), `DELETE FROM wa_sessions WHERE tenant_id = $1`, tenantID)
		}

		writeJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Konfigurasi chatbot berhasil diperbarui",
			Data:    merged,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
	}
}

// handleChatbotPermissions returns the tenant's plan and WA Cloud API feature gate.
func handleChatbotPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
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
		Data: map[string]interface{}{
			"plan":                plan,
			"has_wa_cloud_api":    hasWaCloudAPI,
			"available_providers": []string{"auto", "whatsmeow", "cloud_api"},
		},
	})
}
