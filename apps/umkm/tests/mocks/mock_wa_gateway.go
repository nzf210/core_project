// mocks/mock_wa_gateway.go
// ============================================================
// Mock WA Gateway untuk Unit Testing UMKM Services
// ============================================================
//
// Layer mock ini menggantikan WA Gateway service.
// Digunakan untuk testing pengiriman WhatsApp tanpa perlu
// WA Gateway aktif atau WhatsApp device connected.
//
// Fungsi yang di-mock:
//   - Send message (POST /api/wa/send)
//   - Send OTP (POST /api/wa/send dengan X-Message-Type: otp)
//
// Response yang dikembalikan:
//   - Sukses: message_id untuk tracking
//   - Error: bisa dikonfigurasi untuk testing error handling
//
// Cara pakai:
//   mockWA := NewMockWAGateway()
//   mockWA.SetResponse(`{"message_id":"mock-msg-123"}`)
//   // atau untuk testing error:
//   mockWA.SetError(fmt.Errorf("WA Gateway timeout"))
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
	"time"
)

// MockWAGateway adalah mock untuk WA Gateway service.
type MockWAGateway struct {
	mu            sync.RWMutex
	sendResponse  string
	shouldError   bool
	errorMsg      string
	sentMessages  []SentMessage
	requestLog    []string
	messageIDSeq  int64
}

// SentMessage merekam setiap pesan yang dikirim.
type SentMessage struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	Phone      string                 `json:"phone"`
	Message    string                 `json:"message"`
	MessageType string                `json:"message_type"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewMockWAGateway membuat instance MockWAGateway baru.
func NewMockWAGateway() *MockWAGateway {
	return &MockWAGateway{
		sendResponse: `{"message_id":"mock-msg-001","status":"sent"}`,
	}
}

// SetSendResponse meng-set response untuk send endpoint.
func (m *MockWAGateway) SetSendResponse(response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendResponse = response
}

// SetError mengaktifkan mode error.
func (m *MockWAGateway) SetError(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorMsg = msg
}

// ClearError menonaktifkan mode error.
func (m *MockWAGateway) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = false
	m.errorMsg = ""
}

// GetSentMessages mengembalikan log semua pesan yang dikirim.
func (m *MockWAGateway) GetSentMessages() []SentMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sentMessages
}

// GetRequestLog mengembalikan log semua request.
func (m *MockWAGateway) GetRequestLog() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestLog
}

// Reset membersihkan semua log.
func (m *MockWAGateway) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = nil
	m.requestLog = nil
	m.shouldError = false
	m.errorMsg = ""
	m.messageIDSeq = 0
}

// ============================================================
// HTTP Handler
// ============================================================

// Handler mengembalikan http.HandlerFunc untuk testing.
func (m *MockWAGateway) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.requestLog = append(m.requestLog, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		m.mu.Unlock()

		// Parse request body
		var reqData map[string]any
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
			json.Unmarshal(body, &reqData)
		}

		// Error mode
		if m.shouldError {
			http.Error(w, m.errorMsg, http.StatusInternalServerError)
			return
		}

		// Route based on path
		switch r.URL.Path {
		case "/api/wa/send":
			// Record sent message
			m.mu.Lock()
			m.messageIDSeq++
			msg := SentMessage{
				ID:          fmt.Sprintf("mock-msg-%03d", m.messageIDSeq),
				TenantID:    fmt.Sprintf("%v", reqData["tenant_id"]),
				Phone:       fmt.Sprintf("%v", reqData["phone"]),
				Message:     fmt.Sprintf("%v", reqData["message"]),
				MessageType: r.Header.Get("X-Message-Type"),
				Timestamp:   time.Now(),
				Metadata:    reqData,
			}
			m.sentMessages = append(m.sentMessages, msg)
			m.mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, m.sendResponse)

		default:
			http.NotFound(w, r)
		}
	}
}

// Server membuat httptest.Server dari mock ini.
func (m *MockWAGateway) Server() *httptest.Server {
	return httptest.NewServer(m.Handler())
}

// ============================================================
// RoundTripper - Implements http.RoundTripper
// ============================================================

// RoundTrip implements http.RoundTripper.
func (m *MockWAGateway) RoundTrip(r *http.Request) (*http.Response, error) {
	m.mu.Lock()
	m.requestLog = append(m.requestLog, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
	m.mu.Unlock()

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMsg)
	}

	var respBody string
	if r.URL.Path == "/api/wa/send" {
		respBody = m.sendResponse
	} else {
		respBody = `{"error":"not found"}`
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(respBody))),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

// ============================================================
// Helper untuk validasi
// ============================================================

// AssertMessageSent helper untuk assertion di test.
func (m *MockWAGateway) AssertMessageSent(phone string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msg := range m.sentMessages {
		if msg.Phone == phone {
			return true
		}
	}
	return false
}

// GetLastMessage mengambil pesan terakhir yang dikirim ke nomor tertentu.
func (m *MockWAGateway) GetLastMessage(phone string) *SentMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i := len(m.sentMessages) - 1; i >= 0; i-- {
		if m.sentMessages[i].Phone == phone {
			return &m.sentMessages[i]
		}
	}
	return nil
}
