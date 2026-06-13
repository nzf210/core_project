// mocks/mock_ai_gateway.go
// ============================================================
// Mock AI Gateway untuk Unit Testing UMKM Services
// ============================================================
//
// Layer mock ini menggantikan AI Gateway service.
// Digunakan untuk testing tanpa perlu service AI Gateway aktif.
//
// Fungsi yang di-mock:
//   - Chat completions (POST /v1/chat)
//   - Embeddings generation (POST /v1/embeddings)
//
// Response yang dikembalikan:
//   - Sukses: JSON response sesuai format AI Gateway
//   - Error: bisa dikonfigurasi untuk testing error handling
//
// Cara pakai:
//   mockAI := NewMockAIGateway()
//   mockAI.SetResponse(`{"choices":[{"message":{"content":"Hai!"}}]}`)
//   // atau untuk testing error:
//   mockAI.SetError(fmt.Errorf("AI Gateway timeout"))
//
// Author: Claude Code AI
// Created: 2026-06-13
// ============================================================

package mocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// MockAIGateway adalah mock untuk AI Gateway service.
// Bisa digunakan sebagai http.RoundTripper atau standalone.
type MockAIGateway struct {
	mu           sync.RWMutex
	chatResponse string
	embedResponse string
	shouldError  bool
	errorMsg    string
	chatCalls   []ChatCall
	requestLog  []string
}

// ChatCall merekam setiap pemanggilan chat.
type ChatCall struct {
	TenantID string `json:"tenant_id"`
	Prompt   string `json:"prompt"`
	Model    string `json:"model,omitempty"`
}

// NewMockAIGateway membuat instance MockAIGateway baru.
func NewMockAIGateway() *MockAIGateway {
	return &MockAIGateway{
		chatResponse:  `{"choices":[{"message":{"content":"Mock AI response"}}]}`,
		embedResponse: `{"data":[{"embedding":[0.1,0.2,0.3]}]}`,
	}
}

// SetChatResponse meng-set response untuk chat endpoint.
func (m *MockAIGateway) SetChatResponse(response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatResponse = response
}

// SetEmbedResponse meng-set response untuk embeddings endpoint.
func (m *MockAIGateway) SetEmbedResponse(response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedResponse = response
}

// SetError mengaktifkan mode error.
func (m *MockAIGateway) SetError(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorMsg = msg
}

// ClearError menonaktifkan mode error.
func (m *MockAIGateway) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = false
	m.errorMsg = ""
}

// GetChatCalls mengembalikan log semua chat calls.
func (m *MockAIGateway) GetChatCalls() []ChatCall {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chatCalls
}

// GetRequestLog mengembalikan log semua request.
func (m *MockAIGateway) GetRequestLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestLog
}

// Reset membersihkan semua log.
func (m *MockAIGateway) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatCalls = nil
	m.requestLog = nil
	m.shouldError = false
	m.errorMsg = ""
}

// ============================================================
// HTTP Handler - Bisa langsung dipakai sebagai http.Handler
// ============================================================

// Handler mengembalikan http.HandlerFunc untuk testing.
// Gunakan sebagai: http.HandlerFunc(mockAI.Handler())
func (m *MockAIGateway) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.requestLog = append(m.requestLog, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		m.mu.Unlock()

		// Parse request body untuk logging
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))

			// Extract tenant_id dari request
			var reqData map[string]any
			json.Unmarshal(body, &reqData)
			if tenantID, ok := reqData["tenant_id"].(string); ok {
				m.mu.Lock()
				m.chatCalls = append(m.chatCalls, ChatCall{
					TenantID: tenantID,
					Prompt:   fmt.Sprintf("%v", reqData["messages"]),
				})
				m.mu.Unlock()
			}
		}

		// Error mode
		if m.shouldError {
			http.Error(w, m.errorMsg, http.StatusInternalServerError)
			return
		}

		// Route based on path
		switch r.URL.Path {
		case "/v1/chat", "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, m.chatResponse)

		case "/v1/embeddings":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, m.embedResponse)

		default:
			http.NotFound(w, r)
		}
	}
}

// Server membuat httptest.Server dari mock ini.
func (m *MockAIGateway) Server() *httptest.Server {
	return httptest.NewServer(m.Handler())
}

// ============================================================
// RoundTripper - Implements http.RoundTripper interface
// ============================================================

// RoundTrip implements http.RoundTripper.
// Bisa digunakan dengan http.Client{Transport: mockAI}
func (m *MockAIGateway) RoundTrip(r *http.Request) (*http.Response, error) {
	m.mu.Lock()
	m.requestLog = append(m.requestLog, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
	m.mu.Unlock()

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMsg)
	}

	// Parse request
	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))

		var reqData map[string]any
		json.Unmarshal(body, &reqData)
		if tenantID, ok := reqData["tenant_id"].(string); ok {
			m.mu.Lock()
			m.chatCalls = append(m.chatCalls, ChatCall{
				TenantID: tenantID,
				Prompt:   fmt.Sprintf("%v", reqData["messages"]),
			})
			m.mu.Unlock()
		}
	}

	// Determine response
	var respBody string
	switch r.URL.Path {
	case "/v1/chat", "/v1/chat/completions":
		respBody = m.chatResponse
	case "/v1/embeddings":
		respBody = m.embedResponse
	default:
		respBody = `{"error":"not found"}`
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(respBody))),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}
