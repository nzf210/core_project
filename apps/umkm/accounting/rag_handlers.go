package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"core_project/shared/sdk/response"
)

const (
	errMissingTenantRAG  = "Missing tenant_id"
	errInvalidBodyRAG    = "Invalid body"
	errMethodNotAllowedRAG = response.MethodNotAllowed
	errMsgTenantRequired = "tenant_id required"
	errMsgFailedCreateSession = "Failed to create session"
	errMsgDBError        = response.DBError
)

func handleInternalRAGSearch(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantRAG})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowedRAG})
		return
	}

	var req struct {
		Query     string  `json:"query"`
		TenantID  string  `json:"tenant_id"`
		TopK      int     `json:"top_k"`
		Threshold float64 `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errInvalidBodyRAG})
		return
	}
	if req.Query == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "query cannot be empty"})
		return
	}
	applyRAGDefaults(&req, tenantID)

	queryEmb, err := generateEmbedding(r.Context(), req.Query)
	if err != nil {
		slog.Warn("Failed to generate query embedding, returning empty RAG results", "error", err)
		writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: []any{}})
		return
	}

	results, err := searchRAGEmbeddings(r.Context(), req.TenantID, queryEmb, req.Threshold, req.TopK)
	if err != nil {
		slog.Error("RAG search error", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Search error"})
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: results})
}

func applyRAGDefaults(req *struct {
	Query     string  `json:"query"`
	TenantID  string  `json:"tenant_id"`
	TopK      int     `json:"top_k"`
	Threshold float64 `json:"threshold"`
}, tenantID string) {
	if req.TenantID == "" {
		req.TenantID = tenantID
	}
	if req.TopK == 0 {
		req.TopK = 5
	}
	if req.Threshold == 0 {
		req.Threshold = 0.7
	}
}

func searchRAGEmbeddings(ctx context.Context, tenantID string, queryEmb []float64, threshold float64, topK int) ([]map[string]any, error) {
	rows, err := DB.Query(ctx, `
		SELECT id, source_type, source_id, content, metadata,
		       1 - (embedding <=> $1::vector) AS similarity
		FROM vector_embeddings
		WHERE tenant_id = $2
		  AND (1 - (embedding <=> $1::vector)) >= $3
		ORDER BY embedding <=> $1::vector
		LIMIT $4
	`, queryEmb, tenantID, threshold, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var id, sourceType string
		var sourceID *string
		var content string
		var metaJSON []byte
		var similarity float64
		if err := rows.Scan(&id, &sourceType, &sourceID, &content, &metaJSON, &similarity); err == nil {
			var meta map[string]any
			json.Unmarshal(metaJSON, &meta)
			sID := ""
			if sourceID != nil {
				sID = *sourceID
			}
			results = append(results, map[string]any{
				"id": id, "source_type": sourceType, "source_id": sID,
				"content": content, "similarity": similarity, "metadata": meta,
			})
		}
	}
	if results == nil {
		results = []map[string]any{}
	}
	return results, nil
}

func generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	payload := map[string]any{
		"input": text,
		"model": "text-embedding-ada-002",
	}
	body, _ := json.Marshal(payload)
	reqHTTP, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if Cfg.AI.OpenAIApiKey != "" {
		reqHTTP.Header.Set("Authorization", "Bearer "+Cfg.AI.OpenAIApiKey)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(reqHTTP)
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
		return nil, fmt.Errorf("no embedding returned")
	}
	return result.Data[0].Embedding, nil
}

func handleInternalConversationLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowedRAG})
		return
	}

	var req struct {
		TenantID     string                   `json:"tenant_id"`
		CustomerID   string                   `json:"customer_id"`
		Channel      string                   `json:"channel"`
		UserMessage  string                   `json:"user_message"`
		AssistantMsg string                   `json:"assistant_message"`
		LLMProvider  string                   `json:"llm_provider"`
		LLMModel     string                   `json:"llm_model"`
		TokensUsed   int                      `json:"tokens_used"`
		SessionID    string                   `json:"session_id,omitempty"`
		Confidence   float64                  `json:"confidence,omitempty"`
		RAGSources   []map[string]any `json:"rag_sources,omitempty"`
		LatencyMs    int                      `json:"latency_ms,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errInvalidBodyRAG})
		return
	}
	if req.TenantID == "" || req.CustomerID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "tenant_id and customer_id required"})
		return
	}
	if req.Channel == "" {
		req.Channel = "whatsapp"
	}

	ctx := r.Context()

	var sessionID string
	if req.SessionID != "" {
		sessionID = req.SessionID
	} else {
		err := DB.QueryRow(ctx,
			`SELECT id FROM conversation_sessions
			 WHERE tenant_id = $1 AND customer_id = $2 AND status = 'active'
			 ORDER BY last_message_at DESC LIMIT 1`,
			req.TenantID, req.CustomerID).Scan(&sessionID)
		if err != nil {
			err = DB.QueryRow(ctx,
				`INSERT INTO conversation_sessions (tenant_id, customer_id, channel, status, last_message_at)
				 VALUES ($1, $2, $3, 'active', NOW()) RETURNING id`,
				req.TenantID, req.CustomerID, req.Channel).Scan(&sessionID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errMsgFailedCreateSession})
				return
			}
		}
	}

	ragSrcJSON, _ := json.Marshal(req.RAGSources)
	DB.QueryRow(ctx,
		`INSERT INTO conversation_logs (session_id, tenant_id, role, content, channel,
		 llm_provider, llm_model, tokens_used, latency_ms, confidence, rag_sources)
		 VALUES ($1, $2, 'user', $3, $4, NULL, NULL, 0, 0, NULL, '[]')
		 RETURNING id`,
		sessionID, req.TenantID, req.UserMessage, req.Channel)

	DB.QueryRow(ctx,
		`INSERT INTO conversation_logs (session_id, tenant_id, role, content, channel,
		 llm_provider, llm_model, tokens_used, latency_ms, confidence, rag_sources)
		 VALUES ($1, $2, 'assistant', $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		sessionID, req.TenantID, req.AssistantMsg, req.Channel,
		req.LLMProvider, req.LLMModel, req.TokensUsed, req.LatencyMs,
		req.Confidence, string(ragSrcJSON))

	DB.Exec(ctx,
		`UPDATE conversation_sessions
		 SET message_count = message_count + 2, last_message_at = NOW()
		 WHERE id = $1`,
		sessionID)

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"session_id": sessionID},
	})
}

func handleInternalEscalationLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowedRAG})
		return
	}

	var req struct {
		SessionID              string `json:"session_id"`
		TenantID               string `json:"tenant_id"`
		Reason                 string `json:"reason"`
		TriggerMessage         string `json:"trigger_message"`
		ChatwootConversationID string `json:"chatwoot_conversation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errInvalidBodyRAG})
		return
	}
	if req.TenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMsgTenantRequired})
		return
	}

	ctx := r.Context()

	if req.SessionID != "" {
		DB.Exec(ctx,
			`UPDATE conversation_sessions
			 SET status = 'escalated', escalated_to = $1, escalated_at = NOW()
			 WHERE id = $2`,
			req.ChatwootConversationID, req.SessionID)
	}

	var id string
	err := DB.QueryRow(ctx,
		`INSERT INTO escalation_history
		 (session_id, tenant_id, reason, trigger_message, chatwoot_conversation_id)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		 RETURNING id`,
		req.SessionID, req.TenantID, req.Reason, req.TriggerMessage,
		req.ChatwootConversationID).Scan(&id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to log escalation"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"escalation_id": id},
	})
}

func handleInternalFAQs(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantRAG})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowedRAG})
		return
	}

	rows, err := DB.Query(r.Context(),
		`SELECT id, question, answer FROM tenant_faqs WHERE tenant_id = $1 ORDER BY created_at ASC`,
		tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errMsgDBError})
		return
	}
	defer rows.Close()

	var faqs []map[string]any
	for rows.Next() {
		var id, question, answer string
		if rows.Scan(&id, &question, &answer) == nil {
			faqs = append(faqs, map[string]any{
				"id": id, "question": question, "answer": answer,
			})
		}
	}
	if faqs == nil {
		faqs = []map[string]any{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: faqs})
}

func handleInternalProducts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantRAG})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowedRAG})
		return
	}

	rows, err := DB.Query(r.Context(),
		`SELECT id, name, price, description, COALESCE(category, 'Umum'), COALESCE(stock_quantity, 0)
		 FROM products WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: errMsgDBError})
		return
	}
	defer rows.Close()

	var products []map[string]any
	for rows.Next() {
		var id, name, desc, category string
		var price float64
		var stock int
		if rows.Scan(&id, &name, &price, &desc, &category, &stock) == nil {
			products = append(products, map[string]any{
				"id": id, "name": name, "price": int64(price * 100),
				"description": desc, "category": category, "stock": stock,
			})
		}
	}
	if products == nil {
		products = []map[string]any{}
	}

	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: products})
}

func handleInternalRAGSingle(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenant_id")
	if tenantID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errMissingTenantRAG})
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: errMethodNotAllowedRAG})
		return
	}

	var req struct {
		TenantID   string `json:"tenant_id"`
		SourceType string `json:"source_type"`
		SourceID   string `json:"source_id"`
		Content    string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: errInvalidBodyRAG})
		return
	}
	if req.Content == "" || req.SourceType == "" || req.SourceID == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{Message: "content, source_type, source_id required"})
		return
	}
	if req.TenantID == "" {
		req.TenantID = tenantID
	}

	emb, err := generateEmbedding(r.Context(), req.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to generate embedding"})
		return
	}

	var id string
	err = DB.QueryRow(r.Context(),
		`INSERT INTO vector_embeddings (tenant_id, source_type, source_id, content, embedding)
		 VALUES ($1, $2, $3, $4, $5::vector)
		 ON CONFLICT (tenant_id, source_type, source_id) DO UPDATE
		   SET content = EXCLUDED.content, embedding = EXCLUDED.embedding, updated_at = NOW()
		 RETURNING id`,
		req.TenantID, req.SourceType, req.SourceID, req.Content, vectorFromSlice(emb)).Scan(&id)
	if err != nil {
		slog.Error("Failed to upsert vector embedding", "error", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to store embedding"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    map[string]string{"id": id},
	})
}

func vectorFromSlice(v []float64) string {
	b := &strings.Builder{}
	b.WriteString("[")
	for i, f := range v {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(b, "%.6f", f)
	}
	b.WriteString("]")
	return b.String()
}
