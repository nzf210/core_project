package auth

import (
	"context"
	"testing"
)

func TestIncrementQuota_NoDB(t *testing.T) {
	// Should not panic when DB/cache not initialized
	_, _, err := IncrementQuota(context.Background(), "test-tenant", "ai_text", 1)
	if err != nil {
		// Acceptable to get no error, or a "no DB" error. Both are non-panic outcomes.
		t.Logf("IncrementQuota returned error (acceptable): %v", err)
	}
}

func TestCheckQuotaCounter_NoDB(t *testing.T) {
	// When no cache/DB, should return (true, 0, -1) = allowed
	ok, used, limit := CheckQuotaCounter(context.Background(), "test-tenant", "ai_text")
	if !ok {
		t.Error("expected ok=true when no cache/DB wired")
	}
	if used != 0 {
		t.Errorf("expected used=0, got %d", used)
	}
	if limit != -1 {
		t.Errorf("expected limit=-1 (unlimited), got %d", limit)
	}
}

func TestGetFeatureLimit(t *testing.T) {
	tests := []struct {
		tier     string
		feature  string
		expected int
	}{
		{"ultimate", "ai_text", -1},
		{"ultimate", "ai_vision", 500},
		{"ultimate", "ai_audio_stt", 60},
		{"ultimate", "image_gen", 30},
		{"ultimate", "ai_image", 30}, // F050
		{"pro", "ai_image", 0},      // F050
		{"pro", "ai_text", 5000},
		{"pro", "ai_vision", 50},
		{"pro", "ai_audio_stt", 0},
		{"pro", "image_gen", 0},
		{"lite", "ai_text", 250},
		{"lite", "ai_vision", 0},
		{"lite", "ai_audio_stt", 0},
		{"lite", "image_gen", 0},
		{"inactive", "ai_text", 0},
	}
	for _, tt := range tests {
		t.Run(tt.tier+"_"+tt.feature, func(t *testing.T) {
			plan := PlanFeaturesRow{Tier: tt.tier}
			// Hardcode per spec (matches 000040 seed)
			switch tt.tier {
			case "lite":
				plan.MaxAIText = 250
				plan.MaxAIVision = 0
				plan.MaxAIAudioMinutes = 0
				plan.MaxImageGen = 0
			case "pro":
				plan.MaxAIText = 5000
				plan.MaxAIVision = 50
				plan.MaxAIAudioMinutes = 0
				plan.MaxImageGen = 0
			case "ultimate":
				plan.MaxAIText = -1
				plan.MaxAIVision = 500
				plan.MaxAIAudioMinutes = 60
				plan.MaxImageGen = 30
			}
			got := getFeatureLimit(plan, tt.feature)
			if got != tt.expected {
				t.Errorf("getFeatureLimit(%s, %s) = %d, want %d", tt.tier, tt.feature, got, tt.expected)
			}
		})
	}
}
