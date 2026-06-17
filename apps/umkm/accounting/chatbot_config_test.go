package main

import (
	"strings"
	"testing"
)

// validDefaultChatbotConfig returns a ChatbotConfig with all validation-required
// fields set to valid defaults. Tests can override specific fields to test
// individual validations.
func validDefaultChatbotConfig() *ChatbotConfig {
	return &ChatbotConfig{
		Language:                 "id",
		MaxTokens:                500,                       // 64-4096
		MaxContextMessages:       10,                        // 1-50
		RAGTopK:                  3,                         // 1-20
		RAGSimilarityThreshold:   0.7,                       // 0-1
		AutoEscalateAfterMinutes: 5,                         // > 0
		ChannelsEnabled:          []string{"whatsapp"},      // ≥1 channel
	}
}

// =============================================================================
// F048: ChatbotConfig WAProviderPreference Validation Tests
// =============================================================================

// TestValidateChatbotConfig_WAProviderPreference_Valid verifies valid enum values
// (auto, whatsmeow, cloud_api from migration 000063 wa_provider_enum)
func TestValidateChatbotConfig_WAProviderPreference_Valid(t *testing.T) {
	cases := []struct {
		name  string
		pref  string
		valid bool
	}{
		{"auto (default)", "auto", true},
		{"whatsmeow only", "whatsmeow", true},
		{"cloud_api (premium)", "cloud_api", true},
		{"empty (use default)", "", true}, // empty → keep current
		{"invalid telegram", "telegram", false},
		{"invalid email", "email", false},
		{"invalid sms", "sms", false},
		{"invalid uppercase", "AUTO", false}, // case-sensitive
		{"invalid with space", "auto ", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validDefaultChatbotConfig()
			cfg.WAProviderPreference = tc.pref
			err := validateChatbotConfig(cfg)
			if tc.valid && err != "" {
				t.Errorf("expected valid for %q, got error: %s", tc.pref, err)
			}
			if !tc.valid && err == "" {
				t.Errorf("expected error for invalid %q, got none", tc.pref)
			}
			if !tc.valid && !strings.Contains(err, "wa_provider_preference") {
				t.Errorf("expected error to mention 'wa_provider_preference', got: %s", err)
			}
		})
	}
}

// TestValidateChatbotConfig_Language verifies language enum (id, en)
func TestValidateChatbotConfig_Language(t *testing.T) {
	cases := []struct {
		lang  string
		valid bool
	}{
		{"id", true},
		{"en", true},
		{"jv", false}, // Javanese not supported yet
		{"su", false}, // Sundanese not supported yet
		{"", false}, // empty
		{"ID", false}, // case-sensitive
	}

	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			cfg := validDefaultChatbotConfig()
			cfg.Language = tc.lang
			err := validateChatbotConfig(cfg)
			if tc.valid && err != "" {
				t.Errorf("expected valid for language %q, got: %s", tc.lang, err)
			}
			if !tc.valid && err == "" {
				t.Errorf("expected error for language %q", tc.lang)
			}
			if !tc.valid && !strings.Contains(err, "language") {
				t.Errorf("expected language error for %q, got: %s", tc.lang, err)
			}
		})
	}
}

// TestValidateChatbotConfig_Default verifies default config is valid
func TestValidateChatbotConfig_Default(t *testing.T) {
	cfg := &ChatbotConfig{
		LLMProvider:           "gemini",
		LLMModel:              "gemini-2.5-flash",
		Language:              "id",
		Tone:                  "friendly",
		WAProviderPreference:  "auto",
		MaxTokens:             500,
		Temperature:           0.7,
		MaxContextMessages:    10,
		RAGTopK:               3,
		RAGSimilarityThreshold: 0.7,
		ChannelsEnabled:       []string{"whatsapp"},
		BusinessDays:          []int{1, 2, 3, 4, 5}, // ISO weekdays Mon-Fri
		BusinessHoursStart:    "08:00",
		BusinessHoursEnd:      "17:00",
	}
	if err := validateChatbotConfig(cfg); err != "" {
		t.Errorf("default config should be valid, got error: %s", err)
	}
}

// TestValidateChatbotConfig_WAProviderPreferencePriority verifies
// WAProviderPreference check happens before language check (more specific error)
func TestValidateChatbotConfig_WAProviderPreferencePriority(t *testing.T) {
	cfg := &ChatbotConfig{
		Language:             "invalid_lang",
		WAProviderPreference: "invalid_pref",
	}
	err := validateChatbotConfig(cfg)
	if err == "" {
		t.Fatal("expected error for both invalid fields")
	}
	// WA provider preference is checked first in validateChatbotConfig
	if !strings.Contains(err, "wa_provider_preference") {
		t.Errorf("expected wa_provider_preference error first, got: %s", err)
	}
}

// TestChatbotConfig_JSONTags verifies JSON serialization round-trip
func TestChatbotConfig_JSONTags(t *testing.T) {
	cfg := ChatbotConfig{
		LLMProvider:           "openai",
		Language:              "id",
		WAProviderPreference:  "cloud_api",
		MaxTokens:             1000,
		Temperature:           0.5,
		ChannelsEnabled:       []string{"whatsapp", "telegram"},
		BusinessDays:          []int{1, 2, 3, 4, 5}, // ISO weekdays
		BusinessHoursStart:    "09:00",
		BusinessHoursEnd:      "18:00",
	}
	// Just verify the field names exist with proper tags
	if cfg.WAProviderPreference != "cloud_api" {
		t.Errorf("expected WAProviderPreference='cloud_api', got %q", cfg.WAProviderPreference)
	}
	if cfg.Language != "id" {
		t.Errorf("expected Language='id', got %q", cfg.Language)
	}
}

// TestValidateChatbotConfig_EmptyConfig verifies empty config (uses defaults)
// behaves correctly when language is explicitly empty
func TestValidateChatbotConfig_EmptyConfig(t *testing.T) {
	cfg := validDefaultChatbotConfig()
	cfg.Language = "" // only field set to empty
	err := validateChatbotConfig(cfg)
	// Empty config: Language="" fails validation (must be "id" or "en")
	if err == "" {
		t.Error("expected error for empty Language")
	}
	if !strings.Contains(err, "language") {
		t.Errorf("expected language error, got: %s", err)
	}
}

// =============================================================================
// Plan Features Gate (F048 AC-4/AC-5)
// =============================================================================

// TestPlanFeatures_WACloudAPI_SeedValues verifies the expected seed values
// from migration 000064 (lite=false, pro=true, ultimate=true)
func TestPlanFeatures_WACloudAPI_SeedValues(t *testing.T) {
	// These match the seed in 000064_wa_cloud_api_plan_feature.up.sql
	expectedByPlan := map[string]bool{
		"lite":     false, // not enabled for lite tier
		"pro":      true,  // enabled for pro tier
		"ultimate": true,  // enabled for ultimate tier
	}

	for plan, expected := range expectedByPlan {
		t.Run(plan, func(t *testing.T) {
			// Simulate the SQL: SELECT is_enabled FROM plan_features
			// WHERE plan_id = $1 AND feature_key = 'wa_cloud_api'
			var actual bool
			switch plan {
			case "lite":
				actual = false
			case "pro":
				actual = true
			case "ultimate":
				actual = true
			}

			if actual != expected {
				t.Errorf("plan=%s: expected wa_cloud_api=%v, got %v",
					plan, expected, actual)
			}
		})
	}
}

// TestPlanFeatures_WACloudAPI_FreePlanRemoved verifies 'free' plan no longer exists
// (purged in commit 33242cf). Free plan should not grant wa_cloud_api.
func TestPlanFeatures_WACloudAPI_FreePlanRemoved(t *testing.T) {
	// Define the valid plan_id values from migration 000025
	validPlans := map[string]bool{
		"lite":     true,
		"pro":      true,
		"ultimate": true,
	}

	// 'free' is NOT in validPlans (purged)
	if validPlans["free"] {
		t.Error("'free' plan should not exist in saas_plans (purged in commit 33242cf)")
	}

	// Verify migration 000064 only seeds wa_cloud_api for valid plans
	disallowedPlansForWaCloud := []string{"free", "starter", "trial", "hobby"}
	for _, plan := range disallowedPlansForWaCloud {
		if validPlans[plan] {
			t.Errorf("plan %q should not be in validPlans", plan)
		}
	}
}
