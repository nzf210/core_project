package auth

import (
	"testing"
)

func TestPlanFeaturesRow_IsUnlimited(t *testing.T) {
	tests := []struct {
		name     string
		row      PlanFeaturesRow
		field    string
		expected bool
	}{
		{"max_users=-1 is unlimited", PlanFeaturesRow{MaxUsers: -1}, "max_users", true},
		{"max_users=10 not unlimited", PlanFeaturesRow{MaxUsers: 10}, "max_users", false},
		{"max_ai_vision=-1 unlimited", PlanFeaturesRow{MaxAIVision: -1}, "max_ai_vision", true},
		{"max_ai_vision=0 NOT unlimited (zero = feature disabled)", PlanFeaturesRow{MaxAIVision: 0}, "max_ai_vision", false},
		{"max_ai_vision=50 not unlimited", PlanFeaturesRow{MaxAIVision: 50}, "max_ai_vision", false},
		{"unknown field returns false", PlanFeaturesRow{}, "nonexistent", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.row.IsUnlimited(tt.field)
			if got != tt.expected {
				t.Errorf("IsUnlimited(%s) = %v, want %v", tt.field, got, tt.expected)
			}
		})
	}
}

func TestPlanFeaturesRow_TierDefaults(t *testing.T) {
	// Zero-value PlanFeaturesRow should have Tier="inactive" interpretation
	// Caller must set Tier explicitly, but IsUnlimited should not crash
	p := PlanFeaturesRow{}
	if p.IsUnlimited("max_users") {
		t.Error("zero-value row should not be unlimited")
	}
}
