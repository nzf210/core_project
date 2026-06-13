package mocks

import (
	"sync"
	"testing"
	"time"
)

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// MOCK DB TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestNewMockDB(t *testing.T) {
	db := NewMockDB()
	if db == nil {
		t.Fatal("NewMockDB() returned nil")
	}
	if db.Tables == nil {
		t.Error("Tables map should be initialized")
	}
}

func TestMockDBSeedProducts(t *testing.T) {
	db := NewMockDB()
	tenantID := "tenant-123"

	products := []map[string]interface{}{
		{"id": "prod-1", "name": "Laptop", "price": int64(10000000)},
		{"id": "prod-2", "name": "Mouse", "price": int64(500000)},
	}

	db.SeedProducts(tenantID, products)

	if db.Count("products") != 2 {
		t.Errorf("Expected 2 products, got %d", db.Count("products"))
	}

	// Verify tenant_id was set
	rows := db.QueryByTenant("products", tenantID)
	if len(rows) != 2 {
		t.Errorf("Expected 2 products for tenant %s, got %d", tenantID, len(rows))
	}
}

func TestMockDBSeedTenants(t *testing.T) {
	db := NewMockDB()

	tenants := []map[string]interface{}{
		{"id": "tenant-1", "name": "Toko A"},
		{"id": "tenant-2", "name": "Toko B"},
	}

	db.SeedTenants(tenants)

	if db.Count("tenants") != 2 {
		t.Errorf("Expected 2 tenants, got %d", db.Count("tenants"))
	}
}

func TestMockDBQueryByField(t *testing.T) {
	db := NewMockDB()

	db.SeedProducts("tenant-1", []map[string]interface{}{
		{"id": "prod-1", "name": "Laptop", "price": int64(10000000)},
		{"id": "prod-2", "name": "Mouse", "price": int64(500000)},
	})

	rows := db.QueryByField("products", "name", "Laptop")
	if len(rows) != 1 {
		t.Errorf("Expected 1 product named Laptop, got %d", len(rows))
	}
}

func TestMockDBInsert(t *testing.T) {
	db := NewMockDB()

	db.Insert("products", map[string]interface{}{
		"id": "prod-1",
		"name": "Test Product",
	})

	if db.Count("products") != 1 {
		t.Errorf("Expected 1 product after insert, got %d", db.Count("products"))
	}

	// Verify created_at was set
	last := db.GetLastInsert("products")
	if last["created_at"] == nil {
		t.Error("created_at should be set on insert")
	}
}

func TestMockDBUpdate(t *testing.T) {
	db := NewMockDB()

	db.Insert("products", map[string]interface{}{
		"id": "prod-1",
		"name": "Original Name",
	})

	updated := db.Update("products", "prod-1", map[string]interface{}{
		"name": "Updated Name",
	})

	if !updated {
		t.Error("Update should return true for existing ID")
	}

	rows := db.QueryByField("products", "id", "prod-1")
	if len(rows) != 1 || rows[0]["name"] != "Updated Name" {
		t.Error("Product name should be updated")
	}
}

func TestMockDBUpdateNotFound(t *testing.T) {
	db := NewMockDB()

	updated := db.Update("products", "nonexistent", map[string]interface{}{
		"name": "Should Fail",
	})

	if updated {
		t.Error("Update should return false for non-existing ID")
	}
}

func TestMockDBDelete(t *testing.T) {
	db := NewMockDB()

	db.Insert("products", map[string]interface{}{"id": "prod-1"})
	db.Insert("products", map[string]interface{}{"id": "prod-2"})

	deleted := db.Delete("products", "prod-1")
	if !deleted {
		t.Error("Delete should return true for existing ID")
	}

	if db.Count("products") != 1 {
		t.Errorf("Expected 1 product after delete, got %d", db.Count("products"))
	}
}

func TestMockDBDeleteNotFound(t *testing.T) {
	db := NewMockDB()

	deleted := db.Delete("products", "nonexistent")
	if deleted {
		t.Error("Delete should return false for non-existing ID")
	}
}

func TestMockDBErrorSimulation(t *testing.T) {
	db := NewMockDB()

	db.SetError("test error")
	if !db.shouldError {
		t.Error("shouldError should be true after SetError")
	}

	db.ClearError()
	if db.shouldError {
		t.Error("shouldError should be false after ClearError")
	}
}

func TestMockDBReset(t *testing.T) {
	db := NewMockDB()

	db.Insert("products", map[string]interface{}{"id": "prod-1"})
	db.Insert("tenants", map[string]interface{}{"id": "tenant-1"})

	db.Reset()

	if db.Count("products") != 0 {
		t.Error("Products should be empty after reset")
	}
	if db.Count("tenants") != 0 {
		t.Error("Tenants should be empty after reset")
	}
}

func TestMockDBConcurrentAccess(t *testing.T) {
	db := NewMockDB()
	tenantID := "tenant-concurrent"

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			db.Insert("products", map[string]interface{}{
				"id":       "prod-" + string(rune('0'+idx)),
				"name":     "Product",
				"tenant_id": tenantID,
			})
		}(i)
	}
	wg.Wait()

	// All inserts should succeed without panic
	if db.Count("products") != 10 {
		t.Errorf("Expected 10 products, got %d", db.Count("products"))
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// MOCK ROWS TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestNewMockRows(t *testing.T) {
	data := []map[string]interface{}{
		{"name": "Product 1"},
		{"name": "Product 2"},
	}

	rows := NewMockRows(data)
	if rows == nil {
		t.Fatal("NewMockRows() returned nil")
	}
	if len(rows.rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(rows.rows))
	}
}

func TestMockRowsNext(t *testing.T) {
	data := []map[string]interface{}{
		{"name": "Product 1"},
		{"name": "Product 2"},
	}

	rows := NewMockRows(data)

	if !rows.Next() {
		t.Error("Next() should return true for first row")
	}
	if !rows.Next() {
		t.Error("Next() should return true for second row")
	}
	if rows.Next() {
		t.Error("Next() should return false after last row")
	}
}

func TestMockRowsScan(t *testing.T) {
	data := []map[string]interface{}{
		{"value": "test-value"},
	}

	rows := NewMockRows(data)
	rows.Next()

	var dest string
	err := rows.Scan(&dest)
	if err != nil {
		t.Errorf("Scan() error = %v", err)
	}
	if dest != "test-value" {
		t.Errorf("Scan() dest = %q, want %q", dest, "test-value")
	}
}

func TestMockRowsClose(t *testing.T) {
	data := []map[string]interface{}{
		{"name": "Product 1"},
	}

	rows := NewMockRows(data)
	err := rows.Close()
	if err != nil {
		t.Error("Close() should not return error")
	}
	if !rows.closed {
		t.Error("closed flag should be true after Close()")
	}
}

func TestMockRowsErr(t *testing.T) {
	data := []map[string]interface{}{}
	rows := NewMockRows(data)

	if rows.Err() != nil {
		t.Error("Err() should return nil")
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// CONTEXT HELPER TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestTestContext(t *testing.T) {
	ctx, cancel := TestContext()
	defer cancel()

	if ctx == nil {
		t.Error("TestContext() returned nil context")
	}

	// Verify timeout is set
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have deadline")
	}

	// Deadline should be in the future
	if deadline.Before(time.Now()) {
		t.Error("Deadline should be in the future")
	}
}

// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
// SEED DATA HELPER TESTS
// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

func TestSeedChartOfAccounts(t *testing.T) {
	db := NewMockDB()
	tenantID := "tenant-123"

	accounts := []map[string]interface{}{
		{"code": "1100", "name": "Cash"},
		{"code": "1200", "name": "Accounts Receivable"},
	}

	db.SeedChartOfAccounts(tenantID, accounts)

	if db.Count("chart_of_accounts") != 2 {
		t.Errorf("Expected 2 accounts, got %d", db.Count("chart_of_accounts"))
	}
}

func TestSeedStores(t *testing.T) {
	db := NewMockDB()

	stores := []map[string]interface{}{
		{"id": "store-1", "name": "Main Store"},
		{"id": "store-2", "name": "Branch Store"},
	}

	db.SeedStores(stores)

	if db.Count("stores") != 2 {
		t.Errorf("Expected 2 stores, got %d", db.Count("stores"))
	}
}

func TestSeedAutomations(t *testing.T) {
	db := NewMockDB()
	tenantID := "tenant-123"

	automations := []map[string]interface{}{
		{"id": "auto-1", "name": "Auto Reply"},
		{"id": "auto-2", "name": "Auto Forward"},
	}

	db.SeedAutomations(tenantID, automations)

	if db.Count("tenant_automations") != 2 {
		t.Errorf("Expected 2 automations, got %d", db.Count("tenant_automations"))
	}
}

func TestSeedFAQs(t *testing.T) {
	db := NewMockDB()
	tenantID := "tenant-123"

	faqs := []map[string]interface{}{
		{"id": "faq-1", "question": "What is your return policy?"},
	}

	db.SeedFAQs(tenantID, faqs)

	if db.Count("tenant_faqs") != 1 {
		t.Errorf("Expected 1 FAQ, got %d", db.Count("tenant_faqs"))
	}
}

func TestSeedForwarders(t *testing.T) {
	db := NewMockDB()
	tenantID := "tenant-123"

	forwarders := []map[string]interface{}{
		{"id": "fwd-1", "name": "WhatsApp Forwarder"},
	}

	db.SeedForwarders(tenantID, forwarders)

	if db.Count("tenant_forwarders") != 1 {
		t.Errorf("Expected 1 forwarder, got %d", db.Count("tenant_forwarders"))
	}
}

func TestSeedBusinessTypes(t *testing.T) {
	db := NewMockDB()

	businessTypes := []map[string]interface{}{
		{"id": "bt-1", "name": "Retail"},
		{"id": "bt-2", "name": "Food & Beverage"},
	}

	db.SeedBusinessTypes(businessTypes)

	if db.Count("business_types") != 2 {
		t.Errorf("Expected 2 business types, got %d", db.Count("business_types"))
	}
}
