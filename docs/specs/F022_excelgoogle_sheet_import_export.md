# F022: Excel/Google Sheet Import & Export


## F022: Excel/Google Sheet Import & Export

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** UMKM bisa export data ke Excel/CSV (untuk backup, akuntan, laporan pajak) dan import data dari spreadsheet ke aplikasi. Mendukung **3 entity**: journal_entries (transaksi), products (katalog), dan contacts (pelanggan/forwarders). Format file: `.xlsx` (Excel) dan `.csv` (Google Sheet export friendly).

**Spec:**

### Backend (apps/umkm/accounting)

**Export endpoints** (returns file blob with proper Content-Disposition):

| Method | Path | Format | Entity |
|:-------|:-----|:-------|:-------|
| `GET /export/journal?from=YYYY-MM-DD&to=YYYY-MM-DD&format=xlsx|csv` | download | Journal entries (header + lines) |
| `GET /export/products?format=xlsx|csv` | download | Product catalog |
| `GET /export/contacts?format=xlsx|csv` | download | Customer/forwarder contacts |

**Import endpoints** (multipart/form-data, field name `file`):

| Method | Path | Format | Entity |
|:-------|:-----|:-------|:-------|
| `POST /import/products` | xlsx/csv | Upsert products by SKU |
| `POST /import/contacts` | xlsx/csv | Upsert contacts by phone |
| `POST /import/journal` | xlsx/csv | Create journal entries (validate balanced) |

**Response shape import:**
```json
{
  "success": true,
  "data": {
    "imported": 42,
    "skipped": 3,
    "errors": [
      { "row": 5, "error": "SKU kosong" },
      { "row": 12, "error": "Harga tidak valid" }
    ]
  }
}
```

**CSV column spec (header row required):**

**products** (`name, sku, category, price_cents, stock, description, image_url`)
- `price_cents` integer (sen). Comma-as-thousand-separator NOT supported; user harus convert.
- `stock` integer. Default 0.
- `image_url` optional.

**contacts** (`name, phone, email, role, notes`)
- `phone` wajib, unique per tenant
- `role` ∈ {`customer`, `forwarder`, `supplier`}. Default `customer`.
- `email` optional.

**journal** (`date, description, reference, debit_account_code, credit_account_code, amount_cents`)
- Single-line entry per row (debit & credit on same row)
- For multi-line entry: split into multiple rows with same `reference` (UUID auto-generated per batch, same `reference` = same entry)
- `amount_cents` integer
- Validate balanced per `reference` (sum debit == sum credit)

**XLSX support:**
- 1 sheet per file
- Header row in row 1
- Date cells as Excel date (auto-parse)
- Money cells as number

**Limits:**
- Max 5000 rows per import
- Max file size 10 MB
- File extension whitelist: `.xlsx`, `.csv`

### Frontend

**UI entry points:**
- Sidebar menu baru: `Operasi` group → "Impor / Ekspor Data" (icon: 📥) → route `/data-transfer`
- Atau dari `ProductCatalog.vue` → tombol "Export" + "Import" (inline, per entity)

**Page `DataTransfer.vue`:**
- 3 tab: Jurnal, Produk, Kontak
- Tiap tab:
  - Tombol "Download Template" (CSV + XLSX)
  - Tombol "Export Data" (filter tanggal untuk jurnal)
  - Drop zone untuk upload file + tombol "Impor"
  - Preview hasil import (table) sebelum confirm
  - Toast: imported/skipped/errors count

**Inline di ProductCatalog.vue:**
- Tombol "📥 Import" → file picker → preview → confirm
- Tombol "📤 Export" → dropdown xlsx/csv

### Acceptance Criteria (AC):
- [x] AC-1: GET `/export/products?format=xlsx` return file .xlsx valid
- [x] AC-2: GET `/export/products?format=csv` return file .csv valid
- [x] AC-3: GET `/export/journal?from&to` return file dengan multiple baris per entry
- [x] AC-4: GET `/export/contacts` return file customer + forwarder
- [x] AC-5: POST `/import/products` (xlsx/csv) → upsert by SKU, response include imported/skipped/errors
- [x] AC-6: POST `/import/contacts` (xlsx/csv) → upsert by phone
- [x] AC-7: POST `/import/journal` (xlsx/csv) → create entries, validate balanced per reference
- [x] AC-8: Frontend `DataTransfer.vue` page dengan 3 tab + download template
- [x] AC-9: Inline Import/Export di `ProductCatalog.vue`
- [x] AC-10: Validasi 5000 row max, 10MB max, ext whitelist
- [x] AC-11: `go build`, `go vet`, `go test`, `vue-tsc` clean

### Files Changed:
- `apps/umkm/accounting/main.go` — 6 handlers (3 export, 3 import) + helper parseCSV/parseXLSX
- `shared/sdk/xlsx/` — package baru: `reader.go` (read xlsx), `writer.go` (write xlsx), `csv.go` (CSV helpers)
- `frontend/umkm-web/src/api.ts` — methods `api.exportProducts/Contacts/Journal`, `api.importProducts/Contacts/Journal`, `api.downloadTemplate`
- `frontend/umkm-web/src/components/DataTransfer.vue` — page baru
- `frontend/umkm-web/src/components/ProductCatalog.vue` — inline Import/Export
- `frontend/umkm-web/src/router/index.ts` — route `/data-transfer`
- `frontend/umkm-web/src/config/menu.ts` — menu "Impor / Ekspor Data"

### Notes:
- XLSX pakai library `github.com/jung-kurt/gofpdf` untuk write PDF, dan `github.com/xuri/excelize/v2` untuk read/write xlsx (F021 reuse dependency).
- Import = upsert (SKU/phone sebagai natural key), bukan replace. User bisa re-import untuk update.
- Untuk journal import, file CSV/XLSX diasumsikan valid — backend validasi balance + account existence.
- Bukti audit: setiap import catat ke `import_logs` (insert via mini-migration F022b, atau reuse `subscription_tickets` table sebagai quick win).
