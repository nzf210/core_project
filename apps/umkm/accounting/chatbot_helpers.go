package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)


func loadChatbotConfigByTenant(ctx context.Context, tenantID string) (*ChatbotConfig, error) {
	// Try insert-if-not-exists with default row, ignore conflict
	_, _ = DB.Exec(ctx, `
		INSERT INTO tenant_chatbot_configs (tenant_id)
		VALUES ($1)
		ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID)

	rows, err := DB.Query(ctx,
		`SELECT llm_provider, llm_model, temperature, max_tokens, system_prompt,
		        tone, language, max_context_messages, welcome_message, fallback_message,
			outside_hours_message, business_hours_start, business_hours_end, business_days,
			escalation_enabled, escalation_keywords, escalation_confidence_threshold,
			auto_escalate_after_minutes, rag_enabled, rag_top_k, rag_similarity_threshold,
			channels_enabled, is_active, enable_vision, enable_voice_reply, voice_model, wa_provider_preference
		 FROM tenant_chatbot_configs WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("chatbot config not found after upsert")
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
		return nil, err
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
	return &cfg, nil
}

func validateChatbotConfig(c *ChatbotConfig) string {
	if err := validateWAAndLanguageConfig(c); err != "" {
		return err
	}
	if err := validateToneAndLLMConfig(c); err != "" {
		return err
	}
	if err := validateRAGAndEscalationConfig(c); err != "" {
		return err
	}
	return validateBusinessAndChannelConfig(c)
}

func validateWAAndLanguageConfig(c *ChatbotConfig) string {
	if c.WAProviderPreference != "" && c.WAProviderPreference != "auto" && c.WAProviderPreference != "whatsmeow" && c.WAProviderPreference != "cloud_api" {
		return "wa_provider_preference harus 'auto', 'whatsmeow', atau 'cloud_api'"
	}
	if c.Language != "id" && c.Language != "en" {
		return "language harus 'id' atau 'en'"
	}
	return ""
}

func validateToneAndLLMConfig(c *ChatbotConfig) string {
	if c.Tone != "friendly" && c.Tone != "formal" && c.Tone != "casual" && c.Tone != "professional" && c.Tone != "" {
		return "tone tidak valid"
	}
	if c.Temperature < 0 || c.Temperature > 1 {
		return "temperature harus di antara 0.0 dan 1.0"
	}
	if c.MaxTokens < 64 || c.MaxTokens > 4096 {
		return "max_tokens harus di antara 64 dan 4096"
	}
	if c.MaxContextMessages < 1 || c.MaxContextMessages > 50 {
		return "max_context_messages harus di antara 1 dan 50"
	}
	return ""
}

func validateRAGAndEscalationConfig(c *ChatbotConfig) string {
	if c.RAGTopK < 1 || c.RAGTopK > 20 {
		return "rag_top_k harus di antara 1 dan 20"
	}
	if c.RAGSimilarityThreshold < 0 || c.RAGSimilarityThreshold > 1 {
		return "rag_similarity_threshold harus di antara 0.0 dan 1.0"
	}
	if c.EscalationConfidenceThreshold < 0 || c.EscalationConfidenceThreshold > 1 {
		return "escalation_confidence_threshold harus di antara 0.0 dan 1.0"
	}
	if c.EscalationEnabled && len(c.EscalationKeywords) == 0 {
		return "escalation_keywords tidak boleh kosong jika escalation_enabled = true"
	}
	return ""
}

func validateBusinessAndChannelConfig(c *ChatbotConfig) string {
	if c.BusinessHoursStart != "" && c.BusinessHoursEnd != "" &&
		c.BusinessHoursStart >= c.BusinessHoursEnd {
		return "business_hours_start harus lebih awal dari business_hours_end"
	}
	for _, d := range c.BusinessDays {
		if d < 0 || d > 6 {
			return "business_days harus berisi angka 0-6"
		}
	}
	if len(c.ChannelsEnabled) == 0 {
		return "channels_enabled minimal 1 channel"
	}
	return ""
}

func renderSystemPromptFromConfig(cfg *ChatbotConfig, tenantID, tenantName, role string) string {
	if !cfg.IsActive {
		return cfg.OutsideHoursMessage
	}
	// Honor language & tone
	langHint := "Jawab dalam bahasa Indonesia"
	if cfg.Language == "en" {
		langHint = "Respond in English"
	}
	toneHint := "Gunakan nada yang ramah dan helpful"
	switch cfg.Tone {
	case "formal":
		toneHint = "Gunakan nada formal dan profesional"
	case "casual":
		toneHint = "Gunakan nada santai dan akrab"
	case "professional":
		toneHint = "Gunakan nada profesional dan solutif"
	case "friendly":
		toneHint = "Gunakan nada ramah, hangat, dan bersahabat"
	}

	base := fmt.Sprintf("Anda adalah asisten virtual untuk toko '%s'. %s. %s.", tenantName, langHint, toneHint)

	// If owner set a custom system_prompt, use it as primary and append hints
	if strings.TrimSpace(cfg.SystemPrompt) != "" {
		base = cfg.SystemPrompt + "\n\n" + langHint + ". " + toneHint + "."
	}

	// Add escalation instructions
	if cfg.EscalationEnabled && len(cfg.EscalationKeywords) > 0 {
		base += fmt.Sprintf(
			"\n\nJika pelanggan menggunakan kata kunci seperti [%s], atau secara eksplisit minta bicara dengan admin/pemilik, Anda WAJIB membalas dengan marker [FORWARD_TO_ADMIN] di awal pesan Anda.",
			strings.Join(cfg.EscalationKeywords, ", "),
		)
	}

	return base
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}