// mocks/mock_db.go
// ============================================================
// Mock Database untuk Unit Testing UMKM Services
// ============================================================
//
// Layer mock ini menggantikan koneksi database nyata.
// Setiap test yang butuh DB bisa menggunakan mock ini
// untuk validasi query tanpa menyentuh database asli.
//
// Cara pakai:
//   mockDB := NewMockDB()
//   rows := mockDB.MustRows("SELECT * FROM products WHERE tenant_id = $1", tenantID)
//   count, err := mockDB.Count("SELECT COUNT(*) FROM products WHERE tenant_id = $1", tenantID)
//
// Author: Claude Code AI
// Created: 2026-06-13
// ============================================================

package mocks

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockDB adalah mock untuk *pgxpool.Pool.
// Menyimpan data dalam memory map per tabel.
type MockDB struct {
	mu      sync.RWMutex
	Tables  map[string][]map[string]interface{}
	queries []string // log semua query untuk debugging

	// Konfigurasi
	shouldError bool   // jika true, semua operasi akan return error
	errorMsg    string // pesan error
}

// NewMockDB membuat instance MockDB baru.
func NewMockDB() *MockDB {
	return &MockDB{
		Tables: make(map[string][]map[string]interface{}),
	}
}

// ============================================================
// Seed Data - Memasang data awal untuk testing
// ============================================================

// SeedProducts menambah data produk ke tabel products.
func (m *MockDB) SeedProducts(tenantID string, products []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range products {
		if p["tenant_id"] == nil {
			p["tenant_id"] = tenantID
		}
		if p["created_at"] == nil {
			p["created_at"] = time.Now()
		}
	}
	m.Tables["products"] = append(m.Tables["products"], products...)
}

// SeedTenants menambah data tenant.
func (m *MockDB) SeedTenants(tenants []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range tenants {
		if t["created_at"] == nil {
			t["created_at"] = time.Now()
		}
	}
	m.Tables["tenants"] = append(m.Tables["tenants"], tenants...)
}

// SeedChartOfAccounts menambah data chart of accounts.
func (m *MockDB) SeedChartOfAccounts(tenantID string, accounts []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range accounts {
		if a["tenant_id"] == nil {
			a["tenant_id"] = tenantID
		}
	}
	m.Tables["chart_of_accounts"] = append(m.Tables["chart_of_accounts"], accounts...)
}

// SeedStores menambah data stores.
func (m *MockDB) SeedStores(stores []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range stores {
		if s["created_at"] == nil {
			s["created_at"] = time.Now()
		}
	}
	m.Tables["stores"] = append(m.Tables["stores"], stores...)
}

// SeedAutomations menambah data automations.
func (m *MockDB) SeedAutomations(tenantID string, automations []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range automations {
		if a["tenant_id"] == nil {
			a["tenant_id"] = tenantID
		}
	}
	m.Tables["tenant_automations"] = append(m.Tables["tenant_automations"], automations...)
}

// SeedFAQs menambah data FAQs.
func (m *MockDB) SeedFAQs(tenantID string, faqs []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range faqs {
		if f["tenant_id"] == nil {
			f["tenant_id"] = tenantID
		}
	}
	m.Tables["tenant_faqs"] = append(m.Tables["tenant_faqs"], faqs...)
}

// SeedForwarders menambah data WhatsApp forwarders.
func (m *MockDB) SeedForwarders(tenantID string, forwarders []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, f := range forwarders {
		if f["tenant_id"] == nil {
			f["tenant_id"] = tenantID
		}
	}
	m.Tables["tenant_forwarders"] = append(m.Tables["tenant_forwarders"], forwarders...)
}

// SeedBusinessTypes menambah data business types.
func (m *MockDB) SeedBusinessTypes(businessTypes []map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Tables["business_types"] = append(m.Tables["business_types"], businessTypes...)
}

// ============================================================
// Query Helpers - Membaca data dari mock tables
// ============================================================

// QueryByTenant menjalankan query sederhana dan filter by tenant_id.
// Ini mock sederhana - untuk testing validasi logika bisnis.
func (m *MockDB) QueryByTenant(table, tenantID string) []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, ok := m.Tables[table]
	if !ok {
		return nil
	}

	var result []map[string]interface{}
	for _, row := range rows {
		if tid, ok := row["tenant_id"].(string); ok && tid == tenantID {
			result = append(result, row)
		}
	}
	return result
}

// QueryByField menjalankan query dan filter by satu field.
func (m *MockDB) QueryByField(table, fieldName string, fieldValue interface{}) []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, ok := m.Tables[table]
	if !ok {
		return nil
	}

	var result []map[string]interface{}
	for _, row := range rows {
		if val, ok := row[fieldName]; ok && fmt.Sprintf("%v", val) == fmt.Sprintf("%v", fieldValue) {
			result = append(result, row)
		}
	}
	return result
}

// Count menghitung jumlah row di tabel.
func (m *MockDB) Count(table string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Tables[table])
}

// GetLastInsert mengembalikan row terakhir yang di-insert.
// Berguna untuk validasi ID generation.
func (m *MockDB) GetLastInsert(table string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rows := m.Tables[table]
	if len(rows) == 0 {
		return nil
	}
	return rows[len(rows)-1]
}

// ============================================================
// Mutation Helpers - Insert/Update/Delete
// ============================================================

// Insert menambah row ke tabel.
func (m *MockDB) Insert(table string, row map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if row["created_at"] == nil {
		row["created_at"] = time.Now()
	}
	row["updated_at"] = time.Now()
	m.Tables[table] = append(m.Tables[table], row)
}

// Update mencari row by ID dan merge perubahan.
func (m *MockDB) Update(table, id string, changes map[string]interface{}) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows := m.Tables[table]
	for i, row := range rows {
		if idVal, ok := row["id"].(string); ok && idVal == id {
			for k, v := range changes {
				rows[i][k] = v
			}
			rows[i]["updated_at"] = time.Now()
			m.Tables[table] = rows
			return true
		}
	}
	return false
}

// Delete menghapus row by ID.
func (m *MockDB) Delete(table, id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	rows := m.Tables[table]
	for i, row := range rows {
		if idVal, ok := row["id"].(string); ok && idVal == id {
			m.Tables[table] = append(rows[:i], rows[i+1:]...)
			return true
		}
	}
	return false
}

// ============================================================
// Error Simulation - Untuk testing error handling
// ============================================================

// SetError mengaktifkan mode error untuk testing error handling.
func (m *MockDB) SetError(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = true
	m.errorMsg = msg
}

// ClearError menonaktifkan mode error.
func (m *MockDB) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = false
	m.errorMsg = ""
}

// ============================================================
// Reset - Membersihkan semua data (untuk reuse antar test)
// ============================================================

// Reset membersihkan semua data di semua tabel.
func (m *MockDB) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Tables = make(map[string][]map[string]interface{})
	m.queries = nil
	m.shouldError = false
	m.errorMsg = ""
}

// GetQueries mengembalikan log semua query (untuk debugging).
func (m *MockDB) GetQueries() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.queries
}

// ============================================================
// MockRow - Standalone mock untuk *pgxpool.Rows
// ============================================================

// MockRows adalah mock untuk hasil query.
type MockRows struct {
	mu      sync.RWMutex
	rows    []map[string]interface{}
	current int
	closed  bool
}

// NewMockRows membuat MockRows dari slice map.
func NewMockRows(data []map[string]interface{}) *MockRows {
	return &MockRows{rows: data}
}

// Next implements iterator interface.
func (r *MockRows) Next() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current < len(r.rows) {
		r.current++
		return true
	}
	return false
}

// Scan mengekstrak kolom ke target.
// Target harus pointer ke struct atau map[string]interface{}.
func (r *MockRows) Scan(dest ...interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.current > len(r.rows) || r.current == 0 {
		return fmt.Errorf("no more rows to scan")
	}

	row := r.rows[r.current-1]
	for _, d := range dest {
		switch v := d.(type) {
		case *string:
			if val, ok := row["value"].(string); ok {
				*v = val
			}
		case *int64:
			if val, ok := row["value"].(int64); ok {
				*v = val
			}
		case *int:
			if val, ok := row["value"].(int); ok {
				*v = val
			}
		}
	}
	return nil
}

// Close menutup rows.
func (r *MockRows) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

// Err mengembalikan error terakhir.
func (r *MockRows) Err() error {
	return nil
}

// ============================================================
// Context dengan timeout - Helper untuk test
// ============================================================

// TestContext membuat context dengan timeout 30 detik.
// Cocok untuk test yang butuh context dengan deadline.
func TestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
