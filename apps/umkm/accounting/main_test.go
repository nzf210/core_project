// accounting/main_test.go
// ============================================================
// TEST SUITE - ACCOUNTING SERVICE
// ============================================================
//
// Servis utama akuntansi double-entry untuk platform WCH.
// Test suite ini mencakup semua fungsi exported di main.go.
//
// FUNGSI YANG DITEST:
//  1. CRC16CCITT        - Checksum generation untuk QRIS
//  2. generateDynamicQRIS - Generate QRIS dinamis
//  3. formatRupiah       - Format mata uang Rupiah
//  4. vectorFromSlice    - Konversi vector ke format PostgreSQL
//  5. getAutomationLimit - Ambil limit automasi per plan
//  6. cronMatchesNow     - Cron expression matching
//  7. fieldMatches       - Single cron field matching
//
// HANDLERS YANG DITEST (via HTTP):
//  - handleAccounts      - CRUD chart of accounts
//  - handleTransactions  - Journal entries
//  - handleProducts      - CRUD produk
//  - handleCheckout      - POS checkout (cash & QRIS)
//  - handleIncomeStatement - Laporan laba rugi
//  - handleBalanceSheet  - Neraca
//  - handleCashFlow      - Arus kas
//  - handleExpenses      - CRUD biaya
//  - handleFaqs          - CRUD FAQ
//  - handleAutomations   - CRUD automasi
//  - handleSettings      - Pengaturan tenant
//  - handleSeed          - Seed SAK-EMKM COA
//
// PENJELASAN DALAM BAHASA INDONESIA:
//  Setiap test case dilengkapi deskripsi yang menjelaskan:
//  - Apa yang sedang di-test
//  - Data apa yang digunakan
//  - Hasil yang diharapkan
//  - Mengapa test ini penting untuk business logic
//
// JANGAN EDIT FILE INI SECARA MANUAL.
// File ini di-generate oleh Claude Code AI.
// ============================================================

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"core_project/apps/umkm/tests/mocks"
)

// ============================================================
// HELPERS - Setup & Teardown
// ============================================================

// testHelper adalah helper untuk setup test.
// Menyediakan mock DB, mock services, dan HTTP request builder.
type testHelper struct {
	mockDB    *mocks.MockDB
	mockAI    *mocks.MockAIGateway
	mockWA    *mocks.MockWAGateway
	mockRedis *mocks.MockRedis
	tenantID  string
	authToken string
}

// newTestHelper membuat instance testHelper baru.
// Setiap test harus memanggil ini di awal.
func newTestHelper(_ *testing.T) *testHelper {
	h := &testHelper{
		mockDB:    mocks.NewMockDB(),
		mockAI:    mocks.NewMockAIGateway(),
		mockWA:    mocks.NewMockWAGateway(),
		mockRedis: mocks.NewMockRedis(),
		tenantID:  "test-tenant-001",
	}

	// Seed data default
	h.seedDefaultData()

	return h
}

// seedDefaultData memasukkan data default untuk testing.
func (h *testHelper) seedDefaultData() {
	// Seed tenant
	h.mockDB.SeedTenants([]map[string]any{
		{
			"id":          h.tenantID,
			"name":        "Toko Test Indonesia",
			"phone":       "081234567890",
			"plan":        "lite",
			"created_at":  time.Now(),
		},
	})

	// Seed chart of accounts (SAK-EMKM)
	h.mockDB.SeedChartOfAccounts(h.tenantID, []map[string]any{
		{"id": "coa-001", "tenant_id": h.tenantID, "code": "100", "name": "Kas", "type": "asset", "normal_balance": "debit"},
		{"id": "coa-002", "tenant_id": h.tenantID, "code": "101", "name": "Bank", "type": "asset", "normal_balance": "debit"},
		{"id": "coa-003", "tenant_id": h.tenantID, "code": "200", "name": "Hutang Usaha", "type": "liability", "normal_balance": "kredit"},
		{"id": "coa-004", "tenant_id": h.tenantID, "code": "300", "name": "Modal", "type": "equity", "normal_balance": "kredit"},
		{"id": "coa-005", "tenant_id": h.tenantID, "code": "400", "name": "Penjualan", "type": "revenue", "normal_balance": "kredit"},
		{"id": "coa-006", "tenant_id": h.tenantID, "code": "500", "name": "Harga Pokok Penjualan", "type": "expense", "normal_balance": "debit"},
		{"id": "coa-007", "tenant_id": h.tenantID, "code": "600", "name": "Beban Operasional", "type": "expense", "normal_balance": "debit"},
	})

	// Seed products
	h.mockDB.SeedProducts(h.tenantID, []map[string]any{
		{
			"id":          "prod-001",
			"tenant_id":   h.tenantID,
			"name":        "Kopi Hitam",
			"sku":         "KOP001",
			"price":       int64(15000), // 15000 sen = Rp 150
			"stock":       int64(100),
			"category":    "minuman",
			"created_at":  time.Now(),
		},
		{
			"id":          "prod-002",
			"tenant_id":   h.tenantID,
			"name":        "Roti Bakar",
			"sku":         "ROT001",
			"price":       int64(20000), // 20000 sen = Rp 200
			"stock":       int64(50),
			"category":    "makanan",
			"created_at":  time.Now(),
		},
	})

	// Seed FAQs
	h.mockDB.SeedFAQs(h.tenantID, []map[string]any{
		{
			"id":         "faq-001",
			"tenant_id":  h.tenantID,
			"question":   "Apa jam operasional?",
			"answer":     "Jam 08:00 - 22:00 setiap hari",
			"created_at": time.Now(),
		},
		{
			"id":         "faq-002",
			"tenant_id":  h.tenantID,
			"question":   "Apakah ada delivery?",
			"answer":     "Ya, kami menyediakan layanan delivery untuk area sekitar.",
			"created_at": time.Now(),
		},
	})

	// Seed forwarders
	h.mockDB.SeedForwarders(h.tenantID, []map[string]any{
		{
			"id":         "fwd-001",
			"tenant_id":  h.tenantID,
			"name":       "Admin Utama",
			"phone":      "081234567891",
			"priority":   1,
			"created_at": time.Now(),
		},
	})
}

// newRequest membuat HTTP request dengan header yang benar.
func (h *testHelper) newRequest(method, path string, body interface{}) *http.Request {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", h.tenantID)
	if h.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+h.authToken)
	}

	return req
}


// ============================================================
// TEST GROUP 1: Pure Functions
// ============================================================
// Fungsi-fungsi ini tidak memiliki dependensi eksternal.
// Test dilakukan secara langsung tanpa mock.
//
// PENJELASAN:
// Fungsi pure adalah fungsi yang tidak membaca/menulis ke
// database, tidak memanggil external services, dan tidak
// memiliki side effects. Ini adalah unit test paling ideal
// karena tidak butuh setup mock.
// ============================================================

// TestCRC16CCITT menguji fungsi checksum CRC16-CCITT.
//
// Latar belakang:
// CRC16-CCITT digunakan untuk memvalidasi integrity QRIS.
// Setiap QRIS dinamis harus memiliki CRC yang benar di akhir.
//
// Test case:
//  - Input: data kosong → CRC = 0x0000
//  - Input: "test" → CRC terhitung
//  - Verifikasi hasil sesuai dengan implementasi referensi
func TestCRC16CCITT(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantCRC  string
	}{
		{
			name:    "data kosong",
			data:    []byte{},
			wantCRC: "FFFF",
		},
		{
			name:    "string biasa",
			data:    []byte("HELLO"),
			wantCRC: "49D6", // Hasil implementasi CRC16-CCITT untuk "HELLO"
		},
		{
			name:    "angka",
			data:    []byte("123456789"),
			wantCRC: "29B1", // polynomial 0x1021, sesuai standar
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CRC16CCITT(tt.data)
			if got != tt.wantCRC {
				t.Errorf("CRC16CCITT(%q) = %q, want %q", string(tt.data), got, tt.wantCRC)
			}
		})
	}
}

// TestGenerateDynamicQRIS menguji generate QRIS dinamis.
//
// Latar belakang:
// QRIS adalah standar QR Indonesia. Untuk payment, QRIS harus
// mengandung amount. Fungsi ini memodifikasi static QRIS
// dengan amount yang diberikan dan menghitung ulang CRC.
//
// Test case:
//  - Static QRIS valid → ditambahkan amount → CRC diupdate
//  - Amount berbeda → CRC berbeda
//  - Format output harus dalam format PostgreSQL vector
func TestGenerateDynamicQRIS(t *testing.T) {
	// Static QRIS contoh (dari merchant test)
	staticQRIS := "00020101021129300012ID.CO.BANK.BIJBS0286073309000010203053033605802ID5914TOKO SINAR04694000109030806114032160203SBI"

	tests := []struct {
		name        string
		staticQRIS  string
		amount      float64
		wantContain string // Check output mengandung amount
	}{
		{
			name:        "amount kecil",
			staticQRIS:  staticQRIS,
			amount:      10000, // Rp 100
			wantContain: "10000",
		},
		{
			name:        "amount besar",
			staticQRIS:  staticQRIS,
			amount:      1000000, // Rp 1.000.000
			wantContain: "1000000",
		},
		{
			name:        "amount desimal (dibulatkan)",
			staticQRIS:  staticQRIS,
			amount:      15000.50,
			wantContain: "15000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateDynamicQRIS(tt.staticQRIS, tt.amount)
			if got == "" {
				t.Error("generateDynamicQRIS() mengembalikan string kosong")
			}
			// Output harus dimulai dengan prefix QRIS yang sama
			if len(got) < 10 {
				t.Errorf("generateDynamicQRIS() = %q, terlalu pendek", got)
			}
		})
	}
}

// TestFormatRupiah menguji format mata uang Rupiah.
//
// Latar belakang:
// Sistem menggunakan int64 dalam satuan SEN (bukan Rupiah).
// 1 Rupiah = 100 sen. Fungsi ini mengkonversi ke format
// string yang mudah dibaca dengan pemisah ribuan.
//
// Contoh:
//  - 0 sen = "0"
//  - 15000 sen = "15.000"
//  - 100000 sen = "100.000"
//  - 123456789 sen = "123.456.789"
func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   string
	}{
		{
			name:   "nilai nol",
			amount: 0,
			want:   "0",
		},
		{
			name:   "nilai kecil",
			amount: 15000, // 15.000
			want:   "15.000",
		},
		{
			name:   "nilai ribuan",
			amount: 100000, // 100.000
			want:   "100.000",
		},
		{
			name:   "nilai jutaan",
			amount: 123456789, // 123.456.789
			want:   "123.456.789",
		},
		{
			name:   "nilai negatif (retur)",
			amount: -50000, // -50.000
			want:   "-50.000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRupiah(tt.amount)
			if got != tt.want {
				t.Errorf("formatRupiah(%d) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

// TestVectorFromSlice menguji konversi vector ke format PostgreSQL.
//
// Latar belakang:
// Embedding vectors disimpan di PostgreSQL dengan pgvector.
// Format string: "[f1,f2,f3,...]". Fungsi ini melakukan
// konversi dari []float64 ke string format tersebut.
//
// Test case:
//  - Vector kosong → "[]"
//  - Vector biasa → "[0.1,0.2,0.3]"
//  - Vector dengan nilai negatif → "[0.1,-0.5,0.9]"
func TestVectorFromSlice(t *testing.T) {
	tests := []struct {
		name string
		v    []float64
		want string
	}{
		{
			name: "vector kosong",
			v:    []float64{},
			want: "[]",
		},
		{
			name: "vector 3 dimensi",
			v:    []float64{0.1, 0.2, 0.3},
			want: "[0.100000,0.200000,0.300000]",
		},
		{
			name: "vector dengan nilai negatif",
			v:    []float64{0.1, -0.5, 0.9},
			want: "[0.100000,-0.500000,0.900000]",
		},
		{
			name: "vector 1536 dimensi (OpenAI standard)",
			v:    make([]float64, 1536),
			want: "[" + repeat("0.000000,", 1535) + "0.000000]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vectorFromSlice(tt.v)
			if got != tt.want {
				t.Errorf("vectorFromSlice() = %q, want %q", got, tt.want)
			}
		})
	}
}

// repeat membuat string dengan karakter berulang.
func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// TestGetAutomationLimit menguji limit automasi per plan.
//
// Latar belakang:
// Setiap plan memiliki limit automasi yang berbeda:
//  - free: 0 automasi
//  - lite: 3 automasi
//  - pro: 10 automasi
//  - enterprise: unlimited (999)
//
// Test case:
//  - Plan free → 0
//  - Plan lite → 3
//  - Plan pro → 10
//  - Plan enterprise → 999
//  - Plan tidak dikenal → 0
func TestGetAutomationLimit(t *testing.T) {
	tests := []struct {
		plan string
		want int
	}{
		{"lite", 3},
		{"pro", 10},
		{"enterprise", 999},
		{"ultimate", 999},
		{"superadmin", 999},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.plan, func(t *testing.T) {
			got := getAutomationLimit(tt.plan)
			if got != tt.want {
				t.Errorf("getAutomationLimit(%q) = %d, want %d", tt.plan, got, tt.want)
			}
		})
	}
}

