// mocks/mock_redis.go
// ============================================================
// Mock Redis untuk Unit Testing UMKM Services
// ============================================================
//
// Layer mock ini menggantikan koneksi Redis nyata.
// Digunakan untuk testing queue processing dan caching
// tanpa perlu Redis server.
//
// Fungsi yang di-mock:
//   - Redis queue (chatbot:queue, tenant_events)
//   - Quota tracking (Redis-based counters)
//   - Session caching
//
// Cara pakai:
//   mockRedis := NewMockRedis()
//   mockRedis.Enqueue("chatbot:queue", jobJSON)
//   job := mockRedis.Dequeue("chatbot:queue")
//
// Author: Claude Code AI
// Created: 2026-06-13
// ============================================================

package mocks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MockRedis adalah mock untuk Redis client.
// Menyimpan queue dan cache dalam memory map.
type MockRedis struct {
	mu     sync.RWMutex
	queues map[string][]string        // queue_name -> []json_payload
	cache  map[string]cacheEntry      // key -> value + expiry
	jobs   []MockChatJob              // log semua job untuk debugging
}

// cacheEntry menyimpan value + expiry time.
type cacheEntry struct {
	value    interface{}
	expiryAt time.Time
}

// MockChatJob merepresentasikan job di chatbot queue.
type MockChatJob struct {
	TenantID string                 `json:"tenant_id"`
	Message  string                 `json:"message"`
	Sender   string                 `json:"sender"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewMockRedis membuat instance MockRedis baru.
func NewMockRedis() *MockRedis {
	return &MockRedis{
		queues: make(map[string][]string),
		cache:  make(map[string]cacheEntry),
	}
}

// ============================================================
// Queue Operations - BRPOP/LPUSH style
// ============================================================

// Enqueue menambah item ke queue (LPUSH behavior).
func (m *MockRedis) Enqueue(queue string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var payload string
	switch v := data.(type) {
	case string:
		payload = v
	case []byte:
		payload = string(v)
	default:
		b, _ := json.Marshal(v)
		payload = string(b)
	}

	m.queues[queue] = append(m.queues[queue], payload)

	// Log job untuk debugging
	if queue == "chatbot:queue" {
		var job MockChatJob
		json.Unmarshal([]byte(payload), &job)
		m.jobs = append(m.jobs, job)
	}
}

// Dequeue mengambil item dari queue (BRPOP behavior).
// Mengembalikan nil jika queue kosong.
func (m *MockRedis) Dequeue(queue string) interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	items := m.queues[queue]
	if len(items) == 0 {
		return nil
	}

	// Ambil dari depan (FIFO)
	item := items[0]
	m.queues[queue] = items[1:]
	return item
}

// Peek melihat item pertama tanpa menghapusnya.
func (m *MockRedis) Peek(queue string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := m.queues[queue]
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

// QueueLength mengembalikan panjang queue.
func (m *MockRedis) QueueLength(queue string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.queues[queue])
}

// ============================================================
// Cache Operations - GET/SET/EXPIRE style
// ============================================================

// Set menyimpan value ke cache dengan TTL.
func (m *MockRedis) Set(key string, value interface{}, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	payload := value

	expiryAt := time.Now().Add(ttl)
	if ttl == 0 {
		expiryAt = time.Now().Add(24 * time.Hour * 365) // permanent
	}

	m.cache[key] = cacheEntry{
		value:    payload,
		expiryAt: expiryAt,
	}
}

// Get mengambil value dari cache.
// Mengembalikan nil jika key tidak ada atau sudah expired.
func (m *MockRedis) Get(key string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.cache[key]
	if !ok {
		return nil
	}

	if time.Now().After(entry.expiryAt) {
		return nil
	}

	return entry.value
}

// GetString mengambil value sebagai string.
func (m *MockRedis) GetString(key string) string {
	val := m.Get(key)
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Delete menghapus key dari cache.
func (m *MockRedis) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
}

// Exists cek apakah key ada dan belum expired.
func (m *MockRedis) Exists(key string) bool {
	return m.Get(key) != nil
}

// ============================================================
// Quota Operations - Counter style
// ============================================================

// Incr increment counter (untuk quota tracking).
func (m *MockRedis) Incr(key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.cache[key]
	if !ok {
		m.cache[key] = cacheEntry{
			value:    int64(1),
			expiryAt: time.Now().Add(30 * 24 * time.Hour),
		}
		return 1
	}

	var counter int64
	switch v := entry.value.(type) {
	case int64:
		counter = v
	case int:
		counter = int64(v)
	}
	counter++
	m.cache[key] = cacheEntry{
		value:    counter,
		expiryAt: entry.expiryAt,
	}
	return counter
}

// GetCounter mengambil nilai counter.
func (m *MockRedis) GetCounter(key string) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.cache[key]
	if !ok {
		return 0
	}

	switch v := entry.value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

// ============================================================
// Context dengan timeout - Helper untuk test
// ============================================================

// TestContext membuat context dengan timeout 30 detik.
func RedisTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// ============================================================
// Reset - Membersihkan semua data
// ============================================================

// Reset membersihkan semua queues dan cache.
func (m *MockRedis) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queues = make(map[string][]string)
	m.cache = make(map[string]cacheEntry)
	m.jobs = nil
}

// GetJobs mengembalikan log semua job (untuk debugging).
func (m *MockRedis) GetJobs() []MockChatJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.jobs
}
