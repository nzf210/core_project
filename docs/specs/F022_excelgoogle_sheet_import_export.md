# F022: Excel/Google Sheet Import & Export

**Date:** 2026-06-13  
**Status:** ✅ Approved  
**Implementation:** ✅ Done  
**Related:** [F021](../FEATURE_MAP.md) (PDF Export — reuse excelize dependency)

---

## 🎯 Objectives

UMKM dapat backup, migrate, dan bulk-edit data via spreadsheet untuk kolaborasi dengan akuntan dan compliance pajak.

**Tujuan eksplisit:**
1. Export data (journal, products, contacts) ke Excel (.xlsx) atau CSV untuk backup, audit, dan kolaborasi dengan akuntan
2. Import data dari spreadsheet untuk bulk-add atau bulk-update tanpa manual entry satu-satu
3. Support 3 entity critical: journal_entries (transaksi), products (katalog), contacts (pelanggan/forwarder)

**Problem yang diselesaikan:**
- Manual data entry untuk ratusan produk sangat time-consuming — UMKM butuh bulk import dari supplier spreadsheet
- Akuntan/tax consultant butuh Excel export untuk laporan pajak — tidak bisa copy-paste dari UI
- Migrate dari aplikasi lama (Excel manual accounting) ke WCH butuh import tool — tanpa ini, adoption barrier tinggi

---

## 📋 Acceptance Criteria (AC)

- [x] **AC-1: Export Products to XLSX**
  - *Verification:* `GET /export/products?format=xlsx` return file .xlsx valid dengan header row + data rows
  - *Example:* Excel file dengan kolom: name, sku, category, price_cents, stock, description, image_url

- [x] **AC-2: Export Products to CSV**
  - *Verification:* `GET /export/products?format=csv` return file .csv UTF-8 encoded, comma-separated
  - *Example:* CSV file open di Google Sheets tanpa encoding issue

- [x] **AC-3: Export Journal Entries**
  - *Verification:* `GET /export/journal?from=2026-01-01&to=2026-01-31&format=xlsx` return multi-line entries dengan same reference grouped
  - *Example:* Entry dengan 3 lines (1 debit, 2 credit) → 3 rows dengan reference sama

- [x] **AC-4: Export Contacts**
  - *Verification:* `GET /export/contacts?format=xlsx` return file dengan customer, forwarder, supplier
  - *Example:* Excel file dengan kolom: name, phone, email, role, notes

- [x] **AC-5: Import Products (Upsert by SKU)**
  - *Verification:* `POST /import/products` (multipart/form-data) → upsert products, return `{ imported, skipped, errors }`
  - *Example:* Upload 100 rows, 95 success, 3 skipped (duplicate SKU), 2 errors (invalid price) → response breakdown

- [x] **AC-6: Import Contacts (Upsert by Phone)**
  - *Verification:* `POST /import/contacts` → upsert by phone (natural key), ignore duplicate phone
  - *Example:* Phone `08123456789` already exists → update name/email, skip duplicate

- [x] **AC-7: Import Journal Entries with Balance Validation**
  - *Verification:* `POST /import/journal` → validate sum(debit) == sum(credit) per reference, reject unbalanced
  - *Example:* Reference `REF-001` debit Rp 100.000, credit Rp 90.000 → reject row, error message "Unbalanced entry"

- [x] **AC-8: Frontend DataTransfer Page**
  - *Verification:* `/data-transfer` page dengan 3 tabs (Jurnal, Produk, Kontak), tombol Download Template + Export + Import per tab
  - *Example:* Tab Produk → click "Download Template XLSX" → download template dengan header saja

- [x] **AC-9: Inline Import/Export in ProductCatalog**
  - *Verification:* ProductCatalog.vue memiliki tombol "Import" dan "Export" di header toolbar
  - *Example:* Click "Export" → dropdown xlsx/csv → download instant

- [x] **AC-10: File Validation (Size, Row Count, Extension)**
  - *Verification:* Upload file >10MB → 413 Payload Too Large | Upload >5000 rows → 400 Bad Request | Upload .txt → 400 Invalid Format
  - *Example:* Error message: "File size exceeds 10MB limit"

- [x] **AC-11: Build & Test Pass**
  - *Verification:* `go build ./...`, `go vet ./...`, `go test ./...`, `npm run type-check` (umkm-web) — all pass
  - *Example:* CI/CD green check

---

## 🛠️ Technical Specification

### Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│         Frontend (DataTransfer.vue / ProductCatalog) │
│  1. User click "Export" → GET /export/products?fmt  │
│  2. Browser download file (Content-Disposition)     │
│                                                      │
│  3. User upload file → POST /import/products         │
│  4. Preview result table → Confirm                   │
└──────────────────────┬──────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│         Backend (apps/umkm/accounting)              │
│  Export:                                             │
│    1. Query DB (with tenant_id filter)              │
│    2. Marshal to [][]string (CSV) or excelize (XLSX)│
│    3. Write HTTP response (binary blob)             │
│                                                      │
│  Import:                                             │
│    1. Parse multipart form → read file              │
│    2. Detect format (.xlsx → excelize, .csv → csv)  │
│    3. Validate: max 5000 rows, max 10MB             │
│    4. Parse rows → validate each row                │
│    5. Upsert DB (SKU/phone as natural key)          │
│    6. Return { imported, skipped, errors }          │
└─────────────────────────────────────────────────────┘
```

### Database Schema

**No migration needed** — uses existing tables (`products`, `contacts`, `journal_entries`).

### API Endpoints

#### `GET /export/products?format=xlsx|csv`

**Response (200 OK):**
- Content-Type: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` (xlsx) atau `text/csv` (csv)
- Content-Disposition: `attachment; filename="products_2026-06-13.xlsx"`
- Body: Binary file blob

**XLSX Structure:**
```
Row 1 (Header): name | sku | category | price_cents | stock | description | image_url
Row 2: Nasi Goreng | SKU001 | Makanan | 1500000 | 50 | Nasi goreng spesial | https://...
Row 3: Teh Botol | SKU002 | Minuman | 500000 | 100 | Teh manis dingin | https://...
```

**CSV Structure:**
```csv
name,sku,category,price_cents,stock,description,image_url
"Nasi Goreng",SKU001,Makanan,1500000,50,"Nasi goreng spesial",https://...
"Teh Botol",SKU002,Minuman,500000,100,"Teh manis dingin",https://...
```

**Error Cases:**
- `400 Bad Request` — Invalid format parameter (not `xlsx` or `csv`)
- `401 Unauthorized` — Missing/invalid JWT token

#### `GET /export/journal?from=YYYY-MM-DD&to=YYYY-MM-DD&format=xlsx|csv`

**Query Params:**
- `from` (required): Start date (inclusive)
- `to` (required): End date (inclusive)
- `format` (required): `xlsx` or `csv`

**Response (200 OK):**
- Same structure as products export
- Rows: `date | description | reference | debit_account_code | credit_account_code | amount_cents`

**Multi-Line Entry Example:**
```
date       | description     | reference | debit_account_code | credit_account_code | amount_cents
2026-01-15 | Beli Inventori  | REF-001   | 1100 (Inventori)   | 2100 (Hutang)       | 50000000
2026-01-15 | Beli Inventori  | REF-001   | 5100 (HPP)         | 1100 (Inventori)    | 30000000
```

**Error Cases:**
- `400 Bad Request` — Missing `from` or `to` parameter, or invalid date format

#### `GET /export/contacts?format=xlsx|csv`

**Response (200 OK):**
- Rows: `name | phone | email | role | notes`
- `role` values: `customer`, `forwarder`, `supplier`

#### `POST /import/products`

**Request:**
- Content-Type: `multipart/form-data`
- Field name: `file`
- File: .xlsx or .csv (max 10MB, max 5000 rows)

**Request Example (curl):**
```bash
curl -X POST http://localhost:8201/import/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "file=@products.xlsx"
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "imported": 95,
    "skipped": 3,
    "errors": [
      { "row": 5, "error": "SKU kosong" },
      { "row": 12, "error": "Harga tidak valid (bukan integer)" }
    ]
  }
}
```

**Error Cases:**
- `400 Bad Request` — Invalid file format, file too large (>10MB), too many rows (>5000)
- `413 Payload Too Large` — File >10MB
- `415 Unsupported Media Type` — File extension not `.xlsx` or `.csv`

#### `POST /import/contacts`

**Request:** Same as `/import/products`

**Validation:**
- `phone` wajib (primary key untuk upsert)
- `role` ∈ {`customer`, `forwarder`, `supplier`}. Default `customer`.

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "imported": 42,
    "skipped": 5,
    "errors": [
      { "row": 8, "error": "Phone number kosong" },
      { "row": 15, "error": "Invalid role value: 'admin'" }
    ]
  }
}
```

#### `POST /import/journal`

**Request:** Same as `/import/products`

**Validation:**
- `date`, `debit_account_code`, `credit_account_code`, `amount_cents` wajib
- `reference` optional (auto-generated UUID per batch jika kosong)
- Validate balanced per `reference`: `sum(debit) == sum(credit)`
- Validate account codes exist di chart of accounts

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "imported": 120,
    "skipped": 0,
    "errors": [
      { "row": 25, "error": "Unbalanced entry: debit=100000, credit=90000" },
      { "row": 50, "error": "Account code '9999' not found" }
    ]
  }
}
```

### File Format Specs

**CSV Header Row (Required):**

**Products:**
```csv
name,sku,category,price_cents,stock,description,image_url
```

**Contacts:**
```csv
name,phone,email,role,notes
```

**Journal:**
```csv
date,description,reference,debit_account_code,credit_account_code,amount_cents
```

**XLSX:**
- 1 sheet per file (sheet name ignored)
- Header row in row 1
- Date cells: Excel date format (auto-parsed by excelize)
- Money cells: number (no currency symbol, no comma separator)

**Limits:**
- Max 5000 rows per import
- Max file size 10 MB
- File extension whitelist: `.xlsx`, `.csv`

---

## 🧪 Testing Strategy

### Unit Tests

**Backend (apps/umkm/accounting):**
```go
// import_test.go
func TestParseCSV_ValidProducts(t *testing.T) {
    // Mock CSV with 3 rows
    // Expect: 3 product structs parsed correctly
}

func TestParseXLSX_ValidContacts(t *testing.T) {
    // Mock XLSX with 5 rows
    // Expect: 5 contact structs parsed correctly
}

func TestValidateJournalBalance_Unbalanced(t *testing.T) {
    // Reference "REF-001": debit 100000, credit 90000
    // Expect: validation error "Unbalanced entry"
}

func TestImportProducts_UpsertBySKU(t *testing.T) {
    // Product SKU001 already exists
    // Import same SKU with updated price
    // Expect: UPDATE query, not INSERT
}

func TestImportProducts_MaxRowsExceeded(t *testing.T) {
    // Mock file with 5001 rows
    // Expect: 400 Bad Request
}
```

### Integration Tests

```bash
# 1. Export products
curl -X GET "http://localhost:8201/export/products?format=xlsx" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -o products.xlsx
# → Verify file valid (open dengan Excel/LibreOffice)

# 2. Export journal (date range)
curl -X GET "http://localhost:8201/export/journal?from=2026-01-01&to=2026-01-31&format=csv" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -o journal.csv
# → Verify CSV valid (open dengan Excel/Google Sheets)

# 3. Import products (xlsx)
curl -X POST http://localhost:8201/import/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "file=@products_bulk.xlsx"
# → Verify response: { imported, skipped, errors }

# 4. Import products (csv)
curl -X POST http://localhost:8201/import/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "file=@products_bulk.csv"
# → Verify upsert by SKU

# 5. Import journal with balance validation
curl -X POST http://localhost:8201/import/journal \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "file=@journal_unbalanced.csv"
# → Expect errors array with "Unbalanced entry" messages

# 6. File size limit
curl -X POST http://localhost:8201/import/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -F "file=@large_file_11MB.xlsx"
# → 413 Payload Too Large
```

### Manual Testing Checklist

- [ ] Export products XLSX → open di Excel → verify data correct
- [ ] Export products CSV → open di Google Sheets → verify UTF-8 encoding
- [ ] Export journal (1 month) → verify multi-line entries grouped by reference
- [ ] Export contacts → verify customer + forwarder + supplier included
- [ ] Import 100 products (XLSX) → verify upsert by SKU (update existing, insert new)
- [ ] Import 50 contacts (CSV) → verify upsert by phone
- [ ] Import journal balanced → success
- [ ] Import journal unbalanced → error message per row
- [ ] Upload file >10MB → 413 error
- [ ] Upload file >5000 rows → 400 error
- [ ] Upload .txt file → 415 error
- [ ] DataTransfer.vue page → 3 tabs work, download template work
- [ ] ProductCatalog.vue inline Export/Import → work correctly

---

## 📊 Monitoring & Observability

**Logs:**
```go
slog.Info("Export request", 
  "entity", "products",
  "format", "xlsx",
  "tenant_id", tenantID,
  "row_count", rowCount)

slog.Info("Import completed", 
  "entity", "products",
  "tenant_id", tenantID,
  "imported", imported,
  "skipped", skipped,
  "errors", len(errors))
```

**Metrics to track:**
- Export request count per entity per day
- Import success/failure rate
- Average import time per 1000 rows
- File size distribution (detect if users hitting 10MB limit often)

**Alerts:**
- Import failure rate > 20% → investigate data quality or validation logic
- Export latency > 10s for 1000 rows → DB query optimization needed

---

## 🚀 Rollout Plan

### Phase 1: Backend Handlers (Done ✅)
- Deploy `apps/umkm/accounting` dengan 6 handlers (3 export, 3 import)
- Deploy `shared/sdk/xlsx/` package (excelize wrappers)
- Test via cURL (export + import) → verify response structure

### Phase 2: Frontend DataTransfer Page (Done ✅)
- Deploy umkm-web dengan `/data-transfer` route + `DataTransfer.vue` component
- 3 tabs: Jurnal, Produk, Kontak
- Tombol: Download Template, Export, Import per tab
- Test: end-to-end export → edit → import workflow

### Phase 3: Inline Import/Export in ProductCatalog (Done ✅)
- Add Import/Export buttons di `ProductCatalog.vue` header
- Quick action untuk power users (skip `/data-transfer` page)
- Test: export → edit 1 product → import → verify update

### Phase 4: Audit Log (Future)
- Migration: `import_logs` table (`tenant_id`, `entity`, `file_name`, `imported`, `skipped`, `errors`, `created_at`)
- INSERT log setiap import request
- Superadmin dashboard: import activity timeline

### Rollback
- **Phase 1 rollback:** Remove handlers dari `main.go` routing → export/import endpoints 404
- **Phase 2 rollback:** Remove route `/data-transfer` dari router → page not accessible
- **Emergency:** Feature flag `ENABLE_IMPORT_EXPORT=false` → return 503 Service Unavailable

---

## 🔮 Future Enhancements (Out of Scope)

- **Background Import:** Upload large file → queue job → email notification saat selesai (untuk >5000 rows)
- **Import Preview:** Preview first 10 rows sebelum confirm import (detect column mapping error early)
- **Column Mapping UI:** User bisa map custom column names ke expected fields (e.g., "Nama Produk" → `name`)
- **Export Filter:** Advanced filter di UI (category, price range, stock level) sebelum export
- **Scheduled Export:** Cron job export journal tiap akhir bulan → email ke akuntan otomatis
- **Multi-Sheet XLSX:** Support multiple sheets per file (e.g., sheet 1 = products, sheet 2 = contacts)

---

## 📚 References

- [excelize Library Docs](https://xuri.me/excelize/) — Go library untuk read/write XLSX
- [RFC 4180 CSV Spec](https://tools.ietf.org/html/rfc4180) — CSV format standard
- [Google Sheets Import Guide](https://support.google.com/docs/answer/40608) — CSV import best practices
- [F021: PDF Export](../FEATURE_MAP.md) — Reuse excelize dependency dari PDF export feature

---

## 📝 Notes & Decisions

**2026-06-13:** Decision: Upsert by natural key (SKU/phone) bukan replace entire table — allow user re-import untuk update tanpa delete existing data.  
**2026-06-13:** Max 5000 rows limit untuk prevent timeout + memory spike. Background job untuk large import defer ke future enhancement.  
**2026-06-13:** CSV UTF-8 encoding mandatory — avoid Windows-1252/ISO-8859-1 encoding issue di Google Sheets.  
**2026-06-13:** Journal import validate balance per `reference` bukan per row — support multi-line entry (1 debit, multiple credit lines).  
**2026-06-13:** No audit log table untuk MVP — reuse `subscription_tickets` atau defer to Phase 4. Import activity tracking tidak blocking untuk user adoption.
