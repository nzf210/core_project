package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func callOpenAIEmbeddings(ctx context.Context, apiKey, input, model string) ([]float64, error) {
	if model == "" {
		model = "text-embedding-ada-002"
	}
	payload := map[string]string{"input": input, "model": model}
	body, _ := json.Marshal(payload)
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/embeddings", bytes.NewBuffer(body))
	reqHTTP.Header.Set("Authorization", "Bearer "+apiKey)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(reqHTTP)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, err
	}
	return result.Data[0].Embedding, nil
}

func callMiniMaxEmbeddings(ctx context.Context, apiKey, input string) ([]float64, error) {
	payload := map[string]interface{}{
		"model": "embo1",
		"text":  input,
	}
	body, _ := json.Marshal(payload)
	reqHTTP, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.Anthropic.io/v1/embeddings", bytes.NewBuffer(body))
	reqHTTP.Header.Set("Authorization", "Bearer "+apiKey)
	reqHTTP.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(reqHTTP)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, err
	}
	return result.Data[0].Embedding, nil
}
