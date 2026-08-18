package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"core_project/shared/sdk/config"
)

func init() {
	// Mock config for tests
	config.GlobalConfig = &config.Config{
		AI: struct {
			OpenAIApiKey   string
			GeminiApiKey   string
			MiniMaxAPIKey  string
			MiniMaxBaseURL string
			MiniMaxModel   string
			CacheEnabled   bool
			CacheTTL       int
			CostAlertUSD   float64
			LLM            config.LLMConfig
		}{
			MiniMaxModel: "MiniMax-M2.7",
			LLM: config.LLMConfig{
				Models: []config.LLMModel{
					{
						ID:        "minimax:MiniMax-M2.7",
						Provider:  "minimax",
						Model:     "MiniMax-M2.7",
						IsEnabled: true,
					},
				},
			},
		},
	}
	cfg = config.GlobalConfig
}

func TestHandleHealth(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleHealth)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestHandleChat_InvalidMethod(t *testing.T) {
	req, err := http.NewRequest("GET", "/v1/chat", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleChat)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func BenchmarkHandleChat(b *testing.B) {
	payload := []byte(`{"message": "Hello world", "provider": "gemini"}`)
	
	b.ResetTimer()
	for b.Loop() {
		req, _ := http.NewRequest("POST", "/v1/chat", bytes.NewBuffer(payload))
		rr := httptest.NewRecorder()
		handleChat(rr, req)
	}
}
