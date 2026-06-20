package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"core_project/apps/campaign/api/repository"
)

type BotFAQRequest struct {
	Question string `json:"question"`
}

type BotFAQResponse struct {
	Question string                   `json:"question"`
	Answer   string                   `json:"answer"`
	Sources  []map[string]interface{} `json:"sources"`
}

// HandleBotFAQ — F042: RAG-style retrieval from campaign_documents via vector_embeddings.
// 1. Embed question via AI Gateway /v1/embeddings
// 2. Cosine similarity search in vector_embeddings (top 3)
// 3. Optionally generate final answer via AI Gateway /v1/chat with retrieved context
// Falls back to mock answer if AI Gateway unavailable.
func HandleBotFAQ(w http.ResponseWriter, r *http.Request) {
	tenantID := ExtractTenantID(r)
	if tenantID == "" {
		WriteJSON(w, http.StatusUnauthorized, APIResponse{
			Success: false,
			Message: "Unauthorized - Tenant ID missing",
		})
		return
	}

	if r.Method != http.MethodPost {
		WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
		return
	}

	var req BotFAQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "Invalid JSON payload",
		})
		return
	}

	if req.Question == "" {
		WriteJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Message: "question required",
		})
		return
	}

	ctx := context.Background()

	// Step 1: Get embedding for the question
	embedding, err := embedQuestion(req.Question)
	if err != nil {
		slog.Warn("BotFAQ embedding failed, falling back to keyword search", "error", err)
		// Fallback: simple ILIKE on campaign_documents
		sources, _ := keywordSearch(ctx, tenantID, req.Question, 3)
		WriteJSON(w, http.StatusOK, APIResponse{
			Success: true,
			Data: BotFAQResponse{
				Question: req.Question,
				Answer:   synthesizeFallbackAnswer(req.Question, sources),
				Sources:  sources,
			},
		})
		return
	}

	// Step 2: Vector similarity search (top 3) using pgvector cosine distance
	sources, err := vectorSearch(ctx, tenantID, embedding, 3)
	if err != nil {
		slog.Warn("BotFAQ vector search failed, falling back to keyword search", "error", err)
		sources, _ = keywordSearch(ctx, tenantID, req.Question, 3)
	}

	// Step 3: Compose answer using retrieved context (no LLM call = cheaper + faster)
	answer := synthesizeFallbackAnswer(req.Question, sources)

	WriteJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: BotFAQResponse{
			Question: req.Question,
			Answer:   answer,
			Sources:  sources,
		},
	})
}

// embedQuestion calls AI Gateway /v1/embeddings.
func embedQuestion(question string) ([]float64, error) {
	aiGatewayURL := "http://localhost:8002/v1/embeddings"
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DB_HOST") == "postgres" {
		aiGatewayURL = "http://ai-gateway:8002/v1/embeddings"
	}

	body, _ := json.Marshal(map[string]interface{}{
		"input": question,
		"model": "text-embedding-ada-002",
	})

	httpReq, err := http.NewRequest(http.MethodPost, aiGatewayURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embeddings API returned %d: %s", resp.StatusCode, string(body))
	}

	var embedResp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, err
	}
	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embedResp.Data[0].Embedding, nil
}

// vectorSearch uses pgvector cosine distance operator (<=>) — lower = more similar.
// Returns top K rows with content + metadata + similarity score (1 - distance).
func vectorSearch(ctx context.Context, tenantID string, embedding []float64, topK int) ([]map[string]interface{}, error) {
	// Convert embedding to pgvector literal: '[1.2,3.4,...]'
	embeddingStr := vectorToPgvector(embedding)

	query := `
		SELECT content, source_type, source_id, metadata,
		       1 - (embedding <=> $2::vector) AS similarity
		FROM vector_embeddings
		WHERE tenant_id = $1
		ORDER BY embedding <=> $2::vector
		LIMIT $3
	`

	rows, err := repository.DB.Query(ctx, query, tenantID, embeddingStr, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []map[string]interface{}
	for rows.Next() {
		var content, sourceType, sourceID string
		var metadataBytes []byte
		var similarity float64
		if err := rows.Scan(&content, &sourceType, &sourceID, &metadataBytes, &similarity); err == nil {
			sources = append(sources, map[string]interface{}{
				"content":      content,
				"source_type":  sourceType,
				"source_id":    sourceID,
				"similarity":   similarity,
				"metadata":     string(metadataBytes),
			})
		}
	}
	if sources == nil {
		sources = []map[string]interface{}{}
	}
	return sources, nil
}

// keywordSearch ILIKE fallback when embedding unavailable.
func keywordSearch(ctx context.Context, tenantID, question string, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, title, content
		FROM campaign_documents
		WHERE tenant_id = $1
		  AND (title ILIKE '%' || $2 || '%' OR content ILIKE '%' || $2 || '%')
		LIMIT $3
	`
	rows, err := repository.DB.Query(ctx, query, tenantID, question, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []map[string]interface{}
	for rows.Next() {
		var id, title, content string
		if err := rows.Scan(&id, &title, &content); err == nil {
			sources = append(sources, map[string]interface{}{
				"source_id":   id,
				"source_type": "campaign_document",
				"title":       title,
				"content":     truncate(content, 300),
			})
		}
	}
	if sources == nil {
		sources = []map[string]interface{}{}
	}
	return sources, nil
}

// synthesizeFallbackAnswer composes an answer from retrieved sources without LLM.
// If no sources: returns generic fallback. If 1 source: returns its content.
// If multiple: returns top 1 with attribution.
func synthesizeFallbackAnswer(_ string, sources []map[string]interface{}) string {
	if len(sources) == 0 {
		return "Maaf, saya belum memiliki informasi tersebut dalam dokumen visi-misi kandidat. Silakan tanyakan hal lain atau hubungi koordinator lapangan."
	}

	top := sources[0]
	content, _ := top["content"].(string)
	title, _ := top["title"].(string)

	if title == "" {
		title, _ = top["source_type"].(string)
	}

	if content == "" {
		return "Dokumen ditemukan tapi tidak ada konten yang dapat ditampilkan."
	}

	return fmt.Sprintf("📄 Berdasarkan %s: %s", title, truncate(content, 400))
}

func vectorToPgvector(v []float64) string {
	// Convert []float64 → "[1.2,3.4,5.6]" literal that pgvector accepts as ::vector cast
	buf := bytes.NewBufferString("[")
	for i, f := range v {
		if i > 0 {
			buf.WriteString(",")
		}
		fmt.Fprintf(buf, "%f", f)
	}
	buf.WriteString("]")
	return buf.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}