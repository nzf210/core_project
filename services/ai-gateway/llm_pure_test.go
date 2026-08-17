package main

import (
	"testing"
)

func TestContainsCapability_Match(t *testing.T) {
	if !containsCapability("general,vision,audio", "vision") {
		t.Error("expected match for 'vision'")
	}
}

func TestContainsCapability_NoMatch(t *testing.T) {
	if containsCapability("general,audio", "vision") {
		t.Error("expected no match for 'vision'")
	}
}

func TestContainsCapability_WithSpaces(t *testing.T) {
	if !containsCapability("general, vision , audio", "vision") {
		t.Error("expected match despite spaces")
	}
}

func TestContainsCapability_Empty(t *testing.T) {
	if containsCapability("", "vision") {
		t.Error("expected no match for empty capabilities")
	}
}

func TestParseModelID_WithColon(t *testing.T) {
	provider, model := parseModelID("Anthropic:claude-3-sonnet")
	if provider != "Anthropic" {
		t.Errorf("expected Anthropic, got %s", provider)
	}
	if model != "claude-3-sonnet" {
		t.Errorf("expected claude-3-sonnet, got %s", model)
	}
}

func TestParseModelID_NoColon(t *testing.T) {
	provider, model := parseModelID("gpt-4")
	if provider != "unknown" {
		t.Errorf("expected unknown, got %s", provider)
	}
	if model != "gpt-4" {
		t.Errorf("expected gpt-4, got %s", model)
	}
}

func TestParseModelID_MultipleColons(t *testing.T) {
	provider, model := parseModelID("Anthropic:claude:3")
	if provider != "Anthropic" {
		t.Errorf("expected Anthropic, got %s", provider)
	}
	if model != "claude:3" {
		t.Errorf("expected claude:3, got %s", model)
	}
}

func TestEstimateCost_AnthropicFallback(t *testing.T) {
	cost := estimateCost("Anthropic:unknown-model", 1_000_000, 1_000_000)
	if cost <= 0 {
		t.Error("expected positive cost for Anthropic model")
	}
}

func TestEstimateCost_GeminiFallback(t *testing.T) {
	cost := estimateCost("gemini-pro", 1_000_000, 1_000_000)
	if cost <= 0 {
		t.Error("expected positive cost for gemini model")
	}
}

func TestEstimateCost_OpenAIFallback(t *testing.T) {
	cost := estimateCost("openai:gpt-4", 1_000_000, 1_000_000)
	if cost <= 0 {
		t.Error("expected positive cost for openai model")
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	cost := estimateCost("unknown:model", 1_000_000, 1_000_000)
	if cost != 0 {
		t.Errorf("expected 0 cost for unknown model, got %f", cost)
	}
}

func TestBuildCacheKey_Deterministic(t *testing.T) {
	k1 := buildCacheKey("Anthropic", "claude-3", "system", "hello")
	k2 := buildCacheKey("Anthropic", "claude-3", "system", "hello")
	if k1 != k2 {
		t.Error("cache key should be deterministic")
	}
}

func TestBuildCacheKey_DifferentInputs(t *testing.T) {
	k1 := buildCacheKey("Anthropic", "claude-3", "system", "hello")
	k2 := buildCacheKey("Anthropic", "claude-3", "system", "world")
	if k1 == k2 {
		t.Error("different messages should produce different cache keys")
	}
}

func TestBuildCacheKey_HasPrefix(t *testing.T) {
	k := buildCacheKey("p", "m", "s", "msg")
	if len(k) == 0 {
		t.Error("expected non-empty cache key")
	}
	if k[:9] != "ai:cache:" {
		t.Errorf("expected ai:cache: prefix, got %s", k[:9])
	}
}

func TestCheckCache_NilRedis(t *testing.T) {
	Redis = nil
	val, ok := checkCache(nil, "some-key")
	if ok || val != "" {
		t.Error("expected cache miss with nil Redis")
	}
}

func TestStoreCache_NilRedis(t *testing.T) {
	Redis = nil
	// Should not panic
	storeCache(nil, "key", "val", 60)
}
