# 🗺️ WCH Platform — Feature Map & Specification

> **Dokumen utama untuk AI governance.** Setiap fitur baru/wubah WAJIB ada SPEC di sini.
> User approve SPEC duluan, baru AI implement.

---

## 🔄 Spec-First Workflow

```
USER menulis SPEC      →       AI review & clarify      →       USER approve
     ↓                         ↓                                  ↓
 FEATURE_MAP.md         AI tanya clarifications           USER comment/approve
                              ↓                                  ↓
                      AI wait for approval          AI implement dari SPEC
                                                            ↓
                                                    USER review diff\n                                                            ↓\n                                                    JALANKAN TESTING
```

### Aturan untuk AI:
1. Baca FEATURE_MAP.md sebelum coding
2. Kalau ada feature baru/wubah, tanya USER dulu:
   - "Ada SPEC untuk fitur ini?" → kalau belum, buat draft SPEC
   - "SPEC ini sudah diapprove?" → kalau belum, jangan implement
3. Kalau ada ambiguitas di SPEC, tanya clarification
4. Setelah implement, update kolom `Implementation` di tabel
n5. **Testing Wajib** — Setiap kali ada *perubahan*, *tambah fungsi*, atau *hapus fungsi*, JALANKAN TEST sebelum menyelesaikan task:
   - `make check` (untuk menjalankan linter, build, dan semua test)
   - Atau `go test ./apps/umkm/... -v` (untuk test spesifik)

---

## 📋 Feature Specifications

Format per feature:
```markdown
### FXXX: [Nama Feature]

**Spec Status:** ⏳ Draft | 🔍 In Review | ✅ Approved | ❌ Rejected
**Implementation:** ⏳ Pending | 🔨 In Progress | ✅ Done | ❌ Cancelled

**Deskripsi:** Apa yang fitur ini lakukan

**Spec:**
- Bullet point spesifikasi detail
- Include business rules
- Include validasi yang perlu

**Acceptance Criteria (AC):**
- [ ] AC-1: Kriteria yang bisa diverifikasi
- [ ] AC-2: User bisa test apakah fitur jalan

**Files yang perlu diubah:**
- `path/to/file.go` — deskripsi perubahan

**Notes:**
- Catatan implementasi jika ada
```

---

## 📊 Feature Registry

| ID | Feature | Spec Status | Implementation | Last Updated |
|:---|:--------|:------------|:---------------|:-------------|
| F001 | Multi-Store Quota | ✅ Approved | ✅ Done | 2026-06-12 |
| F002 | Voucher Link Subscription | ✅ Approved | ✅ Done | 2026-06-12 |
| F003 | Subscription Freeze Worker | ✅ Approved | ✅ Done | 2026-06-12 |
| F004 | Read-only Enforcement (Frozen) | ✅ Approved | ✅ Done | 2026-06-12 |
| F005 | Superadmin Dashboard | ✅ Approved | ✅ Done | 2026-06-12 |
| F006 | Multi-Tenant WA Session Pool | ✅ Approved | ✅ Done | 2026-06-01 |
| F007 | Chatbot with RAG | ✅ Approved | ✅ Done | 2026-06-01 |
| F008 | Escalation to Chatwoot | ✅ Approved | ✅ Done | 2026-06-01 |
| F009 | N8N Queue Mode Automation | ✅ Approved | ✅ Done | 2026-06-01 |
| F010 | Campaign Volunteer Management | ✅ Approved | ✅ Done | 2026-06-12 |
| F011 | Campaign Voter Onboarding | ✅ Approved | ✅ Done | 2026-06-12 |
| F012 | Sidebar Navigation UI | ✅ Approved | ✅ Done | 2026-06-12 |
| F013 | N8N Integration via Super Admin | ❌ Removed | — | — |
| F014 | Flexible LLM Model System | ✅ Approved | ✅ Done | 2026-06-12 |
| F015 | Onboarding Activation Flow | ✅ Approved | ✅ Done | 2026-06-13 (UI: 2026-06-14) |
| F016 | Hybrid WhatsApp (Cloud API + whatsmeow) | ✅ Approved | ✅ Done | 2026-06-13 |
| F017 | OTP 1-Hour Reuse Window | ✅ Approved | ✅ Done | 2026-06-13 |
| F018 | Telegram Auth (Register & Login) | ✅ Approved | ✅ Done | 2026-06-13 |
| F019 | Onboarding Sync via /me (Fix Tier 1) | ✅ Approved | ✅ Done | 2026-06-14 |
| F020 | AI CS Setup Wizard (Per-Tenant Config UI) | ✅ Approved | ✅ Done | 2026-06-14 |
| F021 | Cash Flow PDF Export | ✅ Approved | ✅ Done | 2026-06-14 |
| F022 | Excel/Google Sheet Import & Export | ✅ Approved | ✅ Done | 2026-06-14 |

---

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

---

## F021: Cash Flow PDF Export

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Selesaikan Fase 5 PRD — generate PDF Laporan Arus Kas (Cash Flow Statement) untuk periode yang dipilih. Data sudah tersedia via endpoint JSON `GET /reports/cash-flow`; fitur ini tinggal membungkus output jadi file PDF yang siap download/cetak/share ke akuntan atau bank untuk pengajuan kredit. PDF patuh pada layout SAK-EMKM: header (identitas toko + periode), 3 section aktivitas (Operasional, Investasi, Pendanaan), summary net cash flow, footer (tanggal generate + signature line).

**Spec:**

### Backend (apps/umkm/accounting)

Endpoint baru:
- `GET /reports/cash-flow/pdf?from=YYYY-MM-DD&to=YYYY-MM-DD` — return `application/pdf` blob.

**PDF Layout (A4 portrait, margins 2cm):**

```
┌────────────────────────────────────────────────────────┐
│  [NAMA TOKO]                                           │
│  Laporan Arus Kas                                      │
│  Periode: 1 Januari 2026 – 31 Januari 2026            │
│  Dicetak: 14 Juni 2026 01:44 WITA                      │
├────────────────────────────────────────────────────────┤
│  I. ARUS KAS DARI AKTIVITAS OPERASIONAL                │
│    Kas Masuk:                                          │
│      Penjualan Tunai        Rp    5.000.000            │
│      Piutang Tertagih       Rp    2.000.000            │
│      Pendapatan Lain         Rp      500.000           │
│    Total Kas Masuk          Rp    7.500.000            │
│    Kas Keluar:                                          │
│      Beban Gaji              Rp   2.000.000            │
│      Beban Listrik           Rp     300.000            │
│      Beban Bahan Baku        Rp   1.500.000            │
│    Total Kas Keluar          Rp   3.800.000            │
│    Arus Kas Operasional     Rp   3.700.000            │
├────────────────────────────────────────────────────────┤
│  II. ARUS KAS DARI AKTIVITAS INVESTASI                │
│    Pembelian Aset           Rp (1.000.000)             │
│    Arus Kas Investasi       Rp (1.000.000)             │
├────────────────────────────────────────────────────────┤
│  III. ARUS KAS DARI AKTIVITAS PENDANAAN               │
│    Setor Modal               Rp 5.000.000              │
│    Arus Kas Pendanaan        Rp 5.000.000              │
├────────────────────────────────────────────────────────┤
│  KENAIKAN/(PENURUNAN) BERSIH KAS   Rp 7.700.000        │
│  Kas Awal Periode              Rp   X.XXX.XXX          │
│  Kas Akhir Periode             Rp X.XXX.XXX + net     │
├────────────────────────────────────────────────────────┤
│  Halaman 1 dari 1                                      │
│  Generated by WCH Platform • core_project              │
└────────────────────────────────────────────────────────┘
```

**Activity classification logic (berdasarkan account_type + account_code):**

| Category | Rule |
|:---------|:-----|
| Operating Inflow | debit ke cash account (100/101) DAN line counterpart = revenue/piutang (400/120) |
| Operating Outflow | credit ke cash account (100/101) DAN line counterpart = expense/beban (500-599) atau persediaan (130) |
| Investing | counterpart account = fixed asset (150-199) |
| Financing | counterpart account = modal (300), hutang (200-299), prive (310) |

Aturan disederhanakan: kalau counterpart account code di range tertentu → masuk kategori tsb. Sisanya → operating.

**Currency formatting:** IDR dengan format `Rp 1.234.567` (tanpa desimal, pakai titik sebagai thousand separator, tanpa sen — UMKM style).

**Library:** `github.com/jung-kurt/gofpdf` v1.16.2 (UTF-8 ready, lightweight, no native deps).

**Query enhancement:** Extend `handleCashFlow` untuk return per-line breakdown by counterpart account, lalu handler PDF build sectioned report.

### Frontend

**Journal.vue** (Laporan Keuangan section):
- Tambah tombol "📄 Download PDF" di samping tombol "Filter" untuk tab Arus Kas
- Klik → set `window.location = API_BASE + '/api/umkm/reports/cash-flow/pdf?from=...&to=...'`
- Loading state saat generate (PDF bisa 1-2 detik untuk data besar)

### Acceptance Criteria (AC):
- [x] AC-1: GET `/reports/cash-flow/pdf?from&to` return PDF binary (Content-Type: application/pdf)
- [x] AC-2: PDF berisi header (nama toko, periode, tanggal cetak)
- [x] AC-3: PDF punya 3 section (Operasional, Investasi, Pendanaan) + net cash flow + kas awal/akhir
- [x] AC-4: Currency di-format `Rp X.XXX.XXX`
- [x] AC-5: Query extend: return per-counterpart breakdown
- [x] AC-6: Frontend Journal.vue tab Arus Kas punya tombol Download PDF
- [x] AC-7: `go build`, `go vet`, `go test`, `vue-tsc` clean

### Files Changed:
- `apps/umkm/accounting/main.go` — handler `handleCashFlowPDF` + extend `handleCashFlow` (return per-line counterpart data)
- `frontend/umkm-web/src/components/Journal.vue` — tombol Download PDF di tab Arus Kas

### Notes:
- PDF generation synchronous & on-the-fly (tidak ada background job). Untuk data <500 lines latency <500ms. Jika nanti jadi lambat, bisa di-caching.
- `gofpdf` dipakai juga di F022 (Excel tidak, pakai `xuri/excelize/v2`).
- Library `gofpdf` sudah di-pull di F021.
- Indonesian UMKM style: pakai kata "Beban", "Kas Masuk", "Arus Kas", bukan "Expense", "Cash Inflow", "Cash Flow".

---

## F020: AI CS Setup Wizard (Per-Tenant Chatbot Config UI)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Wizard UI untuk owner UMKM setup AI Customer Service mereka sendiri — tanpa coding. Owner bisa atur kepribadian bot, bahasa, jam operasional, kapan bot eskalasi ke admin, dan kalimat sapa/pesan di luar jam. Data disimpan di tabel `tenant_chatbot_configs` (sudah ada di migration 000029) dan di-load oleh chatbot service saat melayani customer. Skenario: Owner daftar → onboarding selesai → masuk wizard ini → 3 langkah simpel → AI CS langsung aktif dengan kepribadian sesuai toko.

**Spec:**

### Backend (apps/umkm/accounting)

Endpoint baru di `apps/umkm/accounting/main.go` (sesuai FEATURE_MAP → lokasi UMKM Accounting):

| Method | Path | Deskripsi |
|:-------|:-----|:----------|
| `GET /api/umkm/chatbot/config` | Ambil konfigurasi chatbot tenant. Auto-create default row jika belum ada (idempotent). |
| `PUT /api/umkm/chatbot/config` | Update konfigurasi. Partial update — hanya field yang dikirim yang di-update. |
| `POST /api/umkm/chatbot/config/test` | Kirim pesan test pakai konfigurasi saat ini (preview — panggil AI Gateway dengan system_prompt yang sudah di-render). Body: `{ "message": "..." }`. Return: `{ "reply": "...", "would_escalate": bool }`. |

**Validation rules:**
- `language` ∈ {`id`, `en`}
- `tone` ∈ {`friendly`, `formal`, `casual`, `professional`}
- `temperature` ∈ [0.0, 1.0]
- `max_tokens` ∈ [64, 4096]
- `max_context_messages` ∈ [1, 50]
- `rag_top_k` ∈ [1, 20]
- `rag_similarity_threshold` ∈ [0.0, 1.0]
- `business_hours_start` < `business_hours_end` (jika sama → reject)
- `business_days` ⊆ {0,1,2,3,4,5,6}
- `escalation_keywords` non-empty jika `escalation_enabled = true`
- `channels_enabled` non-empty (minimal 1 channel)

**Response shape (GET):**
```json
{
  "success": true,
  "data": {
    "llm_provider": "minimax",
    "llm_model": "MiniMax-M2.7",
    "temperature": 0.7,
    "max_tokens": 1024,
    "tone": "friendly",
    "language": "id",
    "max_context_messages": 10,
    "welcome_message": "Halo! Ada yang bisa saya bantu?",
    "fallback_message": "Maaf, saya belum bisa menjawab...",
    "outside_hours_message": "Terima kasih telah menghubungi...",
    "business_hours_start": "08:00",
    "business_hours_end": "22:00",
    "business_days": [1,2,3,4,5,6],
    "escalation_enabled": true,
    "escalation_keywords": ["bicara cs","hubungi admin","operator"],
    "auto_escalate_after_minutes": 5,
    "rag_enabled": true,
    "rag_top_k": 5,
    "rag_similarity_threshold": 0.7,
    "channels_enabled": ["whatsapp"],
    "is_active": true
  }
}
```

### Chatbot integration (apps/umkm/chatbot)

Update `buildSystemPrompt()` di `apps/umkm/chatbot/main.go`:
- Tambah HTTP call ke `accountingURL + "/api/umkm/chatbot/config"` (header `X-Tenant-ID`).
- Cache hasil di Redis dengan key `chatbot:config:{tenant_id}` TTL 5 menit — supaya tidak hit DB tiap chat.
- Jika config `is_active = false` → return `outside_hours_message` regardless jam.
- Honor `business_hours_start/end` + `business_days` — di luar jam → return `outside_hours_message` tanpa panggil LLM (hemat cost).
- Honor `language` → tambahkan instruksi bahasa di system prompt.
- Honor `tone` → tambahkan instruksi nada bicara.
- Honor `max_context_messages` → batasi context window yang dikirim.
- Honor `escalation_keywords` → case-insensitive substring match.
- `system_prompt` custom (jika di-set owner) → pakai itu sebagai base, override default template.

Cache invalidation: saat `PUT /config` dipanggil, chatbot auto-evict cache key (POST notification atau ev langsung via shared Redis key).

### Frontend (frontend/umkm-web)

Component baru: `src/components/ChatbotConfig.vue` (~400 baris).

**Struktur 3-step wizard dengan progress indicator:**

1. **Step 1 — Identitas Bot** (Nama, Bahasa, Tone)
   - Field: Bot name (input text, default: toko), Bahasa (radio: Indonesia/English), Tone (select: friendly/formal/casual/professional)
   - Preview panel kanan: "Bot kamu akan bicara dalam [Bahasa] dengan nada [Tone]"

2. **Step 2 — Jam Operasional & Auto-Escalation**
   - Field: Jam buka-tutup (time pickers), Hari operasional (checkbox 7 hari), Toggle escalation, Escalation keywords (tag input, default suggestions)
   - Preview: "Bot aktif Senin-Sabtu 08:00-22:00. Di luar jam, customer dapat pesan: ..."

3. **Step 3 — Kalimat & Channel**
   - Textarea: Welcome message, Fallback message, Outside hours message (3 textarea)
   - Channel toggles: WhatsApp (default ON, terkunci), Telegram (jika bot token configured), Web chat
   - Tombol "Test Bot" → modal dengan chat preview

**Navigation:**
- Tombol "Lanjut" di step 1-2, "Simpan & Aktifkan" di step 3
- Tombol "Kembali" di step 2-3
- Klik step indicator boleh loncat ke step yang sudah dikunjungi
- Progress disimpan per-step (kalau user keluar, draft tersimpan di sessionStorage)

**Entry points:**
- Setelah onboarding modal activation sukses (F015 flow) → redirect ke `/chatbot-config?first_run=1` (banner "Selamat, lengkapi setup CS AI Anda")
- Sidebar menu: Operasional → "AI CS" (icon: 🤖)
- Settings → bagian "Customer Service AI" → link "Setup/Edit"

**UX detail:**
- Toast success/error pakai pola yang sudah ada
- Loading state pakai skeleton atau spinner
- Empty state untuk fresh tenant: ilustration + CTA "Mulai Setup"
- Responsive: 1 kolom di mobile, 2 kolom (form + preview) di desktop

### Acceptance Criteria (AC):
- [x] AC-1: `GET /api/umkm/chatbot/config` return default config untuk tenant baru (auto-create)
- [x] AC-2: `PUT /api/umkm/chatbot/config` update partial fields, validasi semua constraints
- [x] AC-3: `POST /api/umkm/chatbot/config/test` panggil AI Gateway dengan system_prompt yang sudah di-render
- [x] AC-4: Chatbot `buildSystemPrompt()` baca config dari DB via accounting service, cache 5 menit
- [x] AC-5: Chatbot honor `language`, `tone`, `business_hours_*`, `escalation_keywords`, `max_context_messages`
- [x] AC-6: Di luar jam operasional → return `outside_hours_message` (skip LLM call, hemat cost)
- [x] AC-7: `is_active = false` → chatbot return `outside_hours_message` regardless jam
- [x] AC-8: Frontend `ChatbotConfig.vue` 3-step wizard, progress indicator, form validation
- [x] AC-9: Frontend panggil API real, toast feedback, simpan draft di sessionStorage
- [x] AC-10: Sidebar & Settings entry point berfungsi, banner first_run setelah onboarding
- [x] AC-11: `go build ./...`, `go vet`, `go test ./...`, `vue-tsc --noEmit` clean

### Files Changed:
- `apps/umkm/accounting/main.go` — handler `handleChatbotConfig` (GET/PUT/POST test), SQL helpers
- `apps/umkm/chatbot/main.go` — update `buildSystemPrompt()`, Redis cache integration, business hours + escalation logic
- `frontend/umkm-web/src/components/ChatbotConfig.vue` — wizard component baru
- `frontend/umkm-web/src/api.ts` — method `api.getChatbotConfig`, `api.updateChatbotConfig`, `api.testChatbotConfig`
- `frontend/umkm-web/src/router/index.ts` — route `/chatbot-config`
- `frontend/umkm-web/src/components/AppSidebar.vue` — menu "AI CS"
- `frontend/umkm-web/src/components/Settings.vue` — link "Setup/Edit" CS AI
- `frontend/umkm-web/src/components/Onboarding.vue` — redirect ke `/chatbot-config?first_run=1` setelah activation

### Notes:
- Tabel `tenant_chatbot_configs` sudah ada lengkap dari migration 000029 (F007). Tidak perlu migration baru.
- Backend taruh di `apps/umkm/accounting` (bukan di `chatbot`) karena: (a) accounting sudah jadi hub konfigurasi tenant, (b) chatbot jadi cukup fokus ke runtime, (c) pengurangan coupling — chatbot bisa di-rebuild tanpa ganggu config storage.
- Cache 5 menit dipilih untuk keseimbangan: konfigurasi baru bisa sampai di chatbot max 5 menit (acceptable untuk setup yang jarang berubah), tapi hemat DB call.
- Untuk 'eskalasi' yang sudah ada (mark `[FORWARD_TO_ADMIN]`), logic keyword disatukan — `escalation_keywords` config menggantikan/extend keyword hardcoded.
- Tier 2 — impact langsung ke goal "UMKM bisa bikin CS AI otomatis".

---

## F019: Onboarding Sync via `/me` Endpoint (Fix Tier 1)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sediakan endpoint `GET /me` di auth-service untuk sinkronisasi status user & tenant (`onboarding_completed`, `plan`, `role`, `is_frozen`) dari backend, dan refactor router guard frontend untuk refetch saat localStorage kosong (mis. login di device baru / cache dibersihkan). Fix redirect loop ke `/onboarding` yang sudah lama dicatat di CLAUDE.md. Sekaligus refactor hardcoded WA Gateway URL di chatbot ke config.

**Spec:**
- Endpoint baru `GET /auth/me` di auth-service (alias ringkas dari `/auth/profile` GET). Return JSON berisi `user_id`, `username`, `email`, `phone_number`, `role`, `telegram_chat_id`, `tenant_id`, `plan`, `business_type`, `is_frozen`, `onboarding_completed`.
- Route di api-gateway: `/api/me` → auth-service (dengan auth middleware + tenant rate limit).
- Field `onboarding_completed` juga ditambahkan ke response `GET /auth/profile` agar FE bisa sinkronkan via endpoint yang sudah ada.
- Frontend `router/index.ts`:
  - Tambah helper `fetchAndSyncMe()` yang cache 30s per `(tenant_id, user_id)`.
  - `beforeEach` guard: jika `token` ada tapi `onboarding_completed` missing di localStorage → panggil `fetchAndSyncMe()` untuk populate flag dari BE, baru tentukan redirect.
  - Sync `onboarding_completed`, `plan`, `role`, `subscription_status` ke localStorage/sessionStorage setelah fetch berhasil.
- Refactor `apps/umkm/chatbot/main.go`:
  - Tambah `var WAGatewayURL` + helper `waSendURL()`.
  - Ganti 3 call site hardcoded `http://wa-gateway:8202/api/wa/send` jadi `waSendURL()`.
  - Resolve order: `WA_GATEWAY_URL` env → `cfg.WhatsApp.GatewayURL` → production default.

**Acceptance Criteria (AC):**
- [x] AC-1: `GET /api/me` dengan token valid → return JSON berisi semua field yang disyaratkan
- [x] AC-2: `GET /api/me` tanpa token → 401
- [x] AC-3: `GET /api/profile` sekarang juga return `onboarding_completed`
- [x] AC-4: Reload halaman di device baru dengan localStorage kosong → FE panggil `/me` otomatis, populate flag, tidak redirect loop
- [x] AC-5: Cache `/me` 30s per (tenant_id, user_id) — tidak spam backend
- [x] AC-6: `go build ./...` & `go vet ./...` clean
- [x] AC-7: `go test ./...` all packages green
- [x] AC-8: `vue-tsc --noEmit` clean
- [x] AC-9: Chatbot `waSendURL()` honour `WA_GATEWAY_URL` env + `cfg.WhatsApp.GatewayURL`
- [x] AC-10: 0 hardcoded `wa-gateway:8202` di call site chatbot (sisanya cuma default fallback)

**Files Changed:**
- `services/auth-service/main.go` — `handleMe()` handler baru, field `onboarding_completed` di `handleProfile` GET response, route `/me`
- `services/api-gateway/main.go` — route `/api/me` → auth-service
- `frontend/umkm-web/src/api.ts` — method `api.me()`
- `frontend/umkm-web/src/router/index.ts` — `fetchAndSyncMe()` helper + updated `beforeEach` guard
- `apps/umkm/chatbot/main.go` — `WAGatewayURL` var, `waSendURL()` helper, 3 call site refactored

**Notes:**
- Tier 1 fix — menyentuh 2 service + 1 app + 2 frontend file, semua test pass.
- Branch: `fix/tier1-onboarding-loop`
- Cache 30s dipilih untuk keseimbangan antara freshness dan hemat backend call. Bisa di-tune via env nanti.
- Sinkronisasi hanya terjadi jika flag missing — happy path (user sudah onboarded + localStorage ada) tidak menambah request.

---

## F018: Telegram Auth (Register & Login via Telegram Bot)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** User bisa daftar dan login via Telegram Bot. OTP dikirim melalui Telegram (bukan hanya WhatsApp), memanfaatkan bot Telegram yang sama dengan notification-service. Reuse Redis OTP key yang sudah ada — verify OTP endpoint yang sama tetap berfungsi untuk WA maupun Telegram.

**Spec:**
- User memulai chat dengan bot Telegram WCH → bot reply dengan Chat ID
- Frontend UMKM mengirimkan `telegramChatId` bersama data pendaftaran ke `POST /auth/telegram/register`
- Auth-service generate OTP dan kirim via `sendMessage` Telegram Bot API
- OTP disimpan di Redis dengan key yang sama (`otp:{phone}`) — verifikasi via `POST /auth/verify-otp` tetap berfungsi
- Untuk login: `POST /auth/telegram/login` — verifikasi nomor WA terdaftar, kirim OTP via Telegram, update `telegram_chat_id` di DB
- Webhook bot: `POST /auth/telegram/webhook` — handle command `/start` untuk mengembalikan Chat ID
- Reuse 1-hour OTP reuse window dari F017 — baik WA maupun Telegram

**Acceptance Criteria (AC):**
- [x] AC-1: POST `/auth/telegram/register` dengan `telegramChatId` + data → OTP terkirim ke Telegram
- [x] AC-2: POST `/auth/telegram/login` dengan `telegramChatId` + `phoneNumber` → OTP login terkirim ke Telegram
- [x] AC-3: POST `/auth/verify-otp` tetap berfungsi untuk verifikasi OTP (dari WA maupun Telegram)
- [x] AC-4: POST `/auth/verify-phone-login` tetap berfungsi untuk verifikasi login (dari WA maupun Telegram)
- [x] AC-5: Webhook `/auth/telegram/webhook` handle /start command dan return Chat ID
- [x] AC-6: `telegram_chat_id` tersimpan di tabel users saat registrasi & login via Telegram
- [x] AC-7: OTP 1-hour reuse window (F017) berfungsi untuk Telegram juga

**Files Changed:**
- `services/auth-service/main.go` — Telegram request types, `handleTelegramRegister()`, `handleTelegramLogin()`, `handleTelegramWebhook()`, `sendTelegramOTP()`, updated `handleVerifyOTP()` to parse map-based reg data
- `shared/sdk/config/config.go` — Telegram Bot config struct + loading
- `shared/migrations/000031_telegram_auth.up.sql` — `telegram_chat_id` column + index
- `.env.example` — `TELEGRAM_BOT_TOKEN` documentation

**Notes:**
- Bot token dibaca dari `TELEGRAM_BOT_TOKEN` env — shared dengan notification-service (bisa pakai bot yang sama)
- Webhook URL: `POST https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://<domain>/auth/telegram/webhook`
- Chat ID Telegram user berbeda dengan WA number — mapping disimpan di `users.telegram_chat_id`
- Tidak perlu perubahan di API Gateway — `/auth/telegram/*` otomatis di-proxy oleh existing `/auth/` prefix

---

## F015: Onboarding Activation Flow

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** User baru yang baru daftar bisa lanjut ke step 1 & 2 onboarding tanpa gate. Aktivasi (beli paket / masukkan voucher) baru diminta via modal dialog setelah step 2 selesai. Sistem auto-generate kode voucher sebagai bukti langganan. Superadmin bisa generate voucher dalam jumlah dengan masa aktif day-duration.

**Spec:**

### Onboarding Page (/onboarding)

- Step 1 (Pilih Jenis Usaha) — **tanpa gate**, user boleh pilih atau skip
- Step 2 (Detail Usaha: nama, alamat, nomor WA) — **tanpa gate**, boleh lanjut tanpa harus aktifkan
- Setelah step 2 selesai (klik "Lanjut"), muncul **Modal Activation**:
  - **Opsi A: Beli Paket** — pilih paket (Lite/Pro/Ultimate) → generate Xendit invoice → status subscription = `pending`
  - **Opsi B: Masukkan Kode Voucher** — input kode → validasi → langsung aktivasi jika valid

### Subscription Status Lifecycle

```
Tenant dibuat (register OTP)     → plan=inactive, is_frozen=true
User sampai modal activation
    ├─ Beli Paket                → status=pending, expires_at=now+pending_timeout
    │                              Xendit callback CONFIRMED → activateSubscription()
    │                              Pending > 24 jam tiddk dibayar → hapus tenant + user
    └─ Masukkan Voucher          → validate → activateSubscription() + generate system voucher
```

### Auto-Generate System Voucher (setelah aktivasi via Xendit)

- Saat payment confirmed via Xendit webhook → sistem generate `voucher_codes` entry untuk tenant tersebut
- Format kode: `WCH-{short_tenant_id}-{timestamp}` (contoh: `WCH-a1b2-1750000000`)
- Jenis: `system_generated`, `is_used=true`, `plan_id` sesuai paket yang dibeli
- Kode ini dikirim via WhatsApp notification ke user sebagai "bukti langganan"

### Day-Duration Voucher System (bukan tanggal fixed)

- Kolom `validity_days` (INT) — jumlah hari aktif (bukan `valid_until` date)
- Kolom `remaining_days` — hari tersisa, dihitung saat dibaca
- Saat aktivasi voucher baru:
  - Jika tenant sudah punya voucher aktif dengan **plan yang sama** → akumulasi: `remaining_days += new_validity_days`
  - Jika plan **berbeda** → buat voucher baru secara terpisah (bukan overwrite)
- Priority plan: **Pro > Business > Lite** — sistem baca voucher dengan plan tertinggi sebagai plan aktif

### Auto-Delete Pending Tenant

- Worker `subscription-worker` atau cron di `billing-service` cek tenant dengan `status=pending` dan `created_at < now - 24 jam`
- Hapus row `tenants` + `users` terkait dari DB (CASCADE)
- Log penghapusan ke `subscription_tickets` dengan status `expired`

### Superadmin Voucher Management

- `POST /admin/vouchers/generate` — generate N voucher codes sekaligus
  - Body: `{ plan_id, validity_days, quantity, program_name, max_uses }`
  - Generate N kode acak, simpan ke `voucher_codes`
  - Response: `{ plan_id, validity_days, count, codes: [{code, days}] }`
- `GET /admin/vouchers` — list semua voucher (filter: used/unused, plan_id, program)
  - Response: `{ total, used, unused, codes: [{id, code, program_name, is_redeemed, used_by, used_at, created_at, target_plan}] }`
- `GET /admin/tenants/{id}/vouchers` — list voucher aktif per tenant (untuk melihat masa aktif)
- **UI:** Card "Voucher Billing" di `SuperAdminDashboard.vue` memiliki:
  - Tombol "Lihat Daftar" di header card → buka modal daftar voucher (tabel filterable)
  - Tombol "Generate Voucher" → buka modal generate (input: program name, paket, jumlah, masa aktif) → tampilkan semua kode yang di-generate + Download CSV + Copy per kode

### WhatsApp Notification (Activation)

- Pesan template saat aktivasi:
  ```
  🎉 Langganan WCH Platform berhasil diaktifkan!

  📋 Paket: {plan_name}
  ⏱️  Masa Aktif: {validity_days} hari
  🔑 Kode Voucher: {system_generated_voucher_code}

  Simpan kode ini sebagai bukti langganan Anda.
  ```

**Acceptance Criteria:**
- [x] AC-1: User baru daftar → sampai step 2 onboarding → tidak diblokir, modal activation muncul
- [x] AC-2: Pilih "Beli Paket" → invoice Xendit dibuat, status subscription = `pending`
- [x] AC-3: Bayar Xendit → webhook confirmed → tenant aktif, kode voucher sistem di-generate, dikirim via WA
- [x] AC-4: Pending > 24 jam tidak dibayar → tenant + user dihapus otomatis
- [x] AC-5: Pilih "Masukkan Voucher" → valid → langsung aktivasi + kode voucher sistem dikirim via WA
- [x] AC-6: Redeem voucher plan sama → hari aktif diakumulasi
- [x] AC-7: Redeem voucher plan berbeda → buat voucher baru, priority tetap plan tertinggi
- [x] AC-8: Superadmin bisa generate N voucher codes sekaligus via API
- [x] AC-9: Superadmin bisa lihat voucher aktif per tenant

**Files yang perlu diubah:**
- `frontend/umkm-web/src/components/Onboarding.vue` — hapus gate di step 1 & 2, tambah modal activation
- `frontend/umkm-web/src/components/SuperAdminDashboard.vue` — Generate Voucher modal + Voucher List modal (UI layer)
- `frontend/umkm-web/src/superadminApi.ts` — `listVouchers()` + `generateVouchers()` API methods
- `services/billing-service/main.go` — `pending` subscription status, auto-delete expired, generate system voucher, day-duration logic, `handleAdminGenerateVouchers`, `handleAdminListVouchers`
- `services/auth-service/main.go` — sync `is_frozen` dan plan cache saat activate
- `shared/migrations/` — add `validity_days` / `remaining_days` columns, `pending_timeout` di `tenant_subscriptions`
- `services/subscription-worker/main.go` — cron job auto-delete expired pending tenants
- `services/wa-gateway/` — WhatsApp notification template untuk activation
- `apps/umkm/accounting/main.go` — quota middleware baca priority plan (Pro > Lite)

**Notes:**
- Billing-service adalah source of truth untuk subscription state
- Auth-service baca dari Redis cache, di-sync saat `activateSubscription()` dipanggil
- Pending timeout default: 24 jam (bisa di-config via env `SUBSCRIPTION_PENDING_TIMEOUT_HOURS`)
- Superadmin generate voucher: server-side via `POST /admin/vouchers/generate` di billing-service
- Superadmin UI voucher: `SuperAdminDashboard.vue` memiliki modal Generate (tampilkan semua kode + CSV download) dan modal List (tabel filterable semua voucher)

---

## F016: Hybrid WhatsApp (Cloud API + whatsmeow)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Pisahkan jalur pengiriman WhatsApp: Meta Cloud API (official) untuk pesan transaksional kritis, whatsmeow (unofficial) untuk chatbot dengan rate limiter ketat. Mengurangi risiko banned nomor WA pengguna.

**Spec:**
- WhatsApp Cloud API service baru (`services/wa-cloud-api`, port 8210) wrap Meta Graph API v22.0
- Per-tenant credentials di tabel `wa_cloud_api_credentials` (phone_number_id, access_token, verify_token)
- Message routing via header `X-Message-Type` di wa-gateway:
  - `otp` → Cloud API (auth-service OTP login/register)
  - `invoice` → Cloud API (billing-service payment notifications)
  - `subscription` → Cloud API (accounting revenue digest)
  - `system` → Cloud API (notification-service system alerts)
  - _(tanpa header)_ → whatsmeow (chatbot conversational)
- Fallback: Cloud API gagal → otomatis whatsmeow (logged as WARN)
- Rate limiter whatsmeow: token bucket 5 msg/menit/tenant (mencegah spam ban)
- Reconnect backoff whatsmeow: exponential (30s → 60s → 240s → 10m), max 1 reconnect/5 menit
- Webhook Meta di `/webhooks/wa-cloud/` untuk status callback & incoming messages
- Superadmin CRUD credentials via `/admin/credentials`

**Acceptance Criteria (AC):**
- [x] AC-1: OTP terkirim via Cloud API saat auth-service kirim dengan `X-Message-Type: otp`
- [x] AC-2: Invoice/payment notification terkirim via Cloud API
- [x] AC-3: Chatbot tetap bisa kirim/terima via whatsmeow (tanpa header khusus)
- [x] AC-4: Rate limiter memblokir pesan whatsmeow ke-6+ dalam 1 menit (HTTP 429)
- [x] AC-5: Cloud API gagal → otomatis fallback ke whatsmeow (logged warning)
- [x] AC-6: Webhook Meta diterima di `/webhooks/wa-cloud/`
- [x] AC-7: Superadmin bisa CRUD credential via `POST/GET /admin/credentials`

**Files yang diubah:**
- `services/wa-cloud-api/main.go` — Service baru wrap Meta Graph API
- `services/wa-cloud-api/migrations.go` — Auto-migration runner
- `services/wa-gateway/main.go` — Message router + rate limiter + reconnect backoff
- `shared/sdk/config/config.go` — WhatsApp Cloud API config fields
- `shared/migrations/000030_wa_cloud_api_credentials.up.sql` — New credential table
- `services/api-gateway/main.go` — Webhook route `/webhooks/wa-cloud/` + health check
- `services/auth-service/main.go` — `X-Message-Type: otp` + `X-Source: auth-service`
- `services/billing-service/main.go` — `X-Message-Type: invoice` + `X-Source: billing-service`
- `services/notification-service/main.go` — `X-Message-Type: system` + `X-Source: notification-service`
- `apps/umkm/accounting/main.go` — `X-Message-Type: subscription` + `X-Source: umkm-accounting`
- `Dockerfile` + `docker-compose.yml` + `Makefile` + `.env.example` — Infrastructure

**Notes:**
- WhatsApp Cloud API pricing ~$0.005-0.08/message tergantung tipe. Lebih mahal dari whatsmeow (gratis) tapi zero ban risk.
- Perlu Meta Business App + phone_number_id + permanent access token per tenant
- whatsmeow tetap dipakai untuk chatbot karena conversational messages via Cloud API akan mahal
- Nomor whatsmeow sebaiknya nomor "tumbal" khusus, bukan nomor bisnis utama
- Lihat `CLAUDE.md` section "📱 Hybrid WhatsApp Architecture" untuk detail arsitektur

---

## F017: OTP 1-Hour Reuse Window

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** OTP (registrasi & login via WhatsApp) berlaku selama 1 jam penuh. Selama masa aktif, sistem tidak mengirim ulang OTP — mengurangi volume pesan WA keluar dan menurunkan risiko banned oleh WhatsApp.

**Spec:**
- OTP disimpan di Redis dengan TTL 1 jam (sebelumnya 5 menit)
- Saat user minta OTP baru: cek dulu apakah OTP yang masih aktif ada di Redis
  - Jika ada → return success dengan pesan "OTP sudah dikirim. Masih berlaku 1 jam." (TIDAK kirim ulang)
  - Jika tidak ada → generate OTP baru, simpan ke Redis, kirim via WA Gateway
- OTP TIDAK dihapus setelah verifikasi berhasil — tetap berlaku selama 1 jam penuh
- Redis TTL menangani auto-expiry otomatis setelah 1 jam
- Mencakup 3 endpoint OTP:
  - `POST /register` → OTP registrasi (`otp:{phone}`)
  - `POST /phone-login` → OTP login (`phone-login-otp:{phone}`)
  - `POST /forgot-password` → tidak terpengaruh (email-based, bukan WA)

**Acceptance Criteria (AC):**
- [x] AC-1: Request OTP pertama → OTP baru dikirim via WA, TTL Redis 1 jam
- [x] AC-2: Request OTP kedua dalam 1 jam → tidak kirim ulang, return "OTP sudah dikirim"
- [x] AC-3: Verifikasi OTP sukses → OTP tetap bisa dipakai ulang selama 1 jam
- [x] AC-4: Setelah 1 jam → OTP expired otomatis, request baru generate OTP baru
- [x] AC-5: Login OTP dan Register OTP punya key Redis terpisah (tidak konflik)

**Files Changed:**
- `services/auth-service/main.go` — `handleRegister()`, `handleVerifyOTP()`, `handlePhoneLogin()`, `handleVerifyPhoneLogin()`

**Notes:**
- Mengurangi jumlah pesan WA keluar drastis saat user berkali-kali minta OTP
- Kombinasi dengan F016 (Hybrid WA) + rate limiter memperkuat mitigasi ban
- Risiko keamanan rendah: OTP tetap 6-digit, brute-force dalam 1 jam tidak feasible
- Test OTP `000000` tetap berfungsi di development

---

## F014: Flexible LLM Model System

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sistem LLM yang flexible dan dynamic dengan capability-based routing, mendukung multiple providers dan per-use-case model selection.

**Spec:**
- **Model Registry**: Konfigurasi model dari environment variables dengan capability tags
- **Capability Routing**: Otomatis pilih model berdasarkan `use_case`:
  - `product` — untuk mengambil data product (murah, fast model)
  - `faq` — untuk menjawab FAQ (murah, fast model)
  - `general` — untuk tugas umum (default, full model)
- **Multi-Provider Support**: MiniMax (primary), Gemini (fallback), OpenAI (optional)
- **Fallback Chain**: Automatic fallback ke provider lain jika primary gagal
- **Per-Model Metrics**: Track usage per model (requests, tokens, cost)
- **Prometheus Endpoint**: `/metrics` untuk monitoring
- **API Endpoint**: `/v1/models` untuk list available models

**Environment Variables:**
```bash
# Single model (default)
MINIMAX_MODELS=MiniMax-M2.7
MINIMAX_CAPABILITIES=general,product,faq

# Multiple models (semicolon-separated)
MINIMAX_MODELS=MiniMax-M2.7;MiniMax-M2.7-Fast
MINIMAX_CAPABILITIES=general,product,faq;general
MINIMAX_COST_PER_1M_IN=0.30;0.10
MINIMAX_FALLBACKS=gemini:gemini-1.5-flash
```

**API Usage:**
```json
// Chat request dengan use_case routing
POST /v1/chat
{
  "message": "Apa harga produk X?",
  "use_case": "product"  // → auto-route ke model dengan capability "product"
}

// Override specific model
POST /v1/chat
{
  "message": "Explain code...",
  "provider": "openai",
  "model": "gpt-4o"
}

// List available models
GET /v1/models
```

**Acceptance Criteria:**
- [x] AC-1: Model registry loaded dari environment variables
- [x] AC-2: `use_case` field mengarahkan ke model yang sesuai capability
- [x] AC-3: Fallback chain berfungsi (MiniMax → Gemini → mock)
- [x] AC-4: Per-model metrics trackable via `/metrics`
- [x] AC-5: `/v1/models` endpoint return semua available models

**Files:**
- `shared/sdk/config/config.go` — LLMModel / LLMConfig structs + loadLLMModels()
- `services/ai-gateway/main.go` — capability-based routing + metrics
- `.env.example` — updated dengan flexible model config

**Notes:**
- Fokus saat ini: MiniMax sebagai primary model
- OpenAI/Gemini sebagai fallback/optional
- Per-tenant model override bisa ditambahkan di future (via DB config)

---

## F012: Sidebar Navigation UI

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Ganti horizontal header menu dengan sidebar kiri yang grouped dan collapsible.

**Spec:**
- Sidebar kiri dengan grouped menu items
- Groups: Operasi (Dashboard, Kasir, Katalog), Keuangan (Jurnal), Sistem (Automasi, Pengaturan, Super Admin)
- Collapsible groups — klik header untuk expand/collapse
- Active route highlighting
- User profile di bottom sidebar
- Responsive: sidebar di desktop, drawer di mobile
- Data-driven menu config (bukan hardcoded HTML)

**Acceptance Criteria:**
- [x] AC-1: Sidebar menampilkan grouped menu items
- [x] AC-2: Groups bisa collapse/expand
- [x] AC-3: Active route di-highlight
- [x] AC-4: User profile terlihat di sidebar
- [x] AC-5: Mobile: hamburger → drawer sidebar
- [x] AC-6: Smooth transition animations

**Files:**
- `frontend/umkm-web/src/components/AppSidebar.vue` — sidebar component baru
- `frontend/umkm-web/src/config/menu.ts` — menu configuration
- `frontend/umkm-web/src/App.vue` — use sidebar
- `frontend/umkm-web/src/style.css` — global sidebar styles

**Notes:** Icon menggunakan emoji untuk simplicity (bisa upgrade ke lucide-icons nanti).

---

## F013: N8N Integration via Super Admin

**Spec Status:** ❌ Removed
**Implementation:** —

**Deskripsi:** Integrate N8N ke Super Admin dashboard sebagai monitoring hub, bukan custom UI.

**Spec:**
- Super Admin dashboard → link ke N8N UI (new tab)
- N8N status indicator (connected/running/error)
- Recent executions widget (fetch from N8N API)
- Quick action: "Buka Workflow Editor"

**Acceptance Criteria:**
- [x] AC-1: N8N status visible di Super Admin
- [x] AC-2: Direct link to N8N editor
- [x] AC-3: Recent executions shown

**Files:**
- `services/billing-service/main.go` — N8N status & executions endpoints
- `frontend/superadmin-web/src/views/Dashboard.vue` — Direct link ke N8N editor
- `frontend/umkm-web/src/components/SuperAdminDashboard.vue` — Direct link ke N8N editor

**Notes:** N8N UI tetap digunakan untuk workflow editing. Super Admin hanya sebagai hub + monitoring.

---

**REMOVED (2026-06-12):** F013 dihapus karena:
- Tidak perlu dedicated `/n8n` page — N8N editor langsung diakses via `http://localhost:5678`
- Fitur sudah terpenuhi cukup dengan link di Dashboard.vue (direct ke N8N editor)

---

## F001: Multi-Store Quota Management

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** 1 owner bisa buat banyak toko dengan quota per plan.

**Spec:**
- 1 owner = banyak `stores` dengan `business_type` berbeda (restoran + cafe, dll)
- Quota di-enforce via `plan_features.feature_key='max_stores'`
- Default per tier: Lite=1, Pro=1, Ultimate=5
- Superadmin bisa ubah quota via billing-service tanpa migration

**Acceptance Criteria:**
- [x] AC-1: GET `/api/umkm/stores` return quota info (`max_stores`, `can_add`)
- [x] AC-2: POST `/api/umkm/stores` check quota sebelum create
- [x] AC-3: Superadmin bisa CRUD plan-features via `/admin/plan-features`
- [x] AC-4: Header `X-User-Role: superadmin` injected by api-gateway

**Files:**
- `apps/umkm/accounting/main.go` — stores CRUD + quota check
- `services/billing-service/main.go` — superadmin plan-features CRUD

**Notes:** Quota dibaca langsung dari `plan_features` table, tidak di-cache.

---

## F002: Voucher Link Subscription Model

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Subscription = link-based voucher (primary) + Xendit (hybrid B2B).

**Spec:**
- Superadmin generate bulk voucher links via `/admin/voucher-links/generate`
- User klik link → redeem → subscription extend/created
- Grace period 0 hari (langsung freeze saat expired)
- Freeze = read-only + banner, user masih bisa login

**Voucher Lifecycle:**
```
[Superadmin] POST /admin/voucher-links/generate
    { program_id, count, valid_days, base_url }
    → Returns: { links: [{token, url}, ...] }

[User] Klik link → POST /voucher/redeem-link { token, tenant_id }
    1. Verify JWT signature
    2. Lookup voucher_links by SHA-256(token)
    3. Validate: is_active, not redeemed, not expired
    4. Check max_uses_per_tenant
    5. Mark link redeemed
    6. Extend or create subscription
    7. Un-freeze if was frozen
```

**Acceptance Criteria:**
- [x] AC-1: Superadmin generate voucher links (bulk)
- [x] AC-2: User redeem via signed token link
- [x] AC-3: Subscription extend/created on redeem
- [x] AC-4: Tenant un-frozen on successful redeem

**Files:**
- `services/billing-service/main.go` — voucher generation + redemption
- `shared/migrations/000028_voucher_subscription.up.sql` — schema

**Notes:** Voucher token di-hash SHA-256 sebelum save ke DB.

---

## F003: Subscription Freeze Worker

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Background worker yang freeze tenant expired.

**Spec:**
- Cek `tenant_subscriptions` setiap `FREEZE_CHECK_INTERVAL` (default 1 jam)
- Subscription dengan `current_period_end < NOW()` → freeze
- Batch update: `status='frozen'`, `tenants.is_frozen=true`
- Liveness check: GET `/healthz`

**Acceptance Criteria:**
- [x] AC-1: Worker running dengan interval configurable
- [x] AC-2: Expired subscriptions frozen automatically
- [x] AC-3: `is_frozen` denormalized flag updated

**Files:**
- `services/subscription-worker/main.go` — freeze worker
- `docker-compose.yml` — worker service definition

**Notes:** GRACE_PERIOD_HOURS=0 (0-day freeze).

---

## F004: Read-only Enforcement (Frozen Tenant)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Middleware block write operations saat tenant frozen.

**Spec:**
- Middleware `auth.RequireActiveSubscription`
- Block POST/PATCH/PUT/DELETE saat frozen
- GET tetap pass (user bisa lihat data)
- Set header `X-Subscription-Status: active|frozen`

**Acceptance Criteria:**
- [x] AC-1: Write operations blocked saat frozen
- [x] AC-2: Read operations tetap jalan
- [x] AC-3: Response include subscription status header

**Files:**
- `shared/sdk/auth/subscription_guard.go` — middleware
- `apps/umkm/accounting/main.go` — applied ke router

**Notes:** Banner message untuk UI frontend dari header `X-Subscription-Status`.

---

## F005: Superadmin Dashboard

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Unified dashboard untuk superadmin (bukan per-product).

**Spec:**
- 1 unified dashboard di `frontend/superadmin-web/` (port 3401)
- Sections: Overview, Voucher Programs, Generate Links, Frozen Accounts
- Overview: tenant counts, voucher stats 30d, revenue (Xendit), subs by plan
- Frozen Accounts: list + kirim reminder WA

**Acceptance Criteria:**
- [x] AC-1: Overview dengan aggregated stats
- [x] AC-2: Voucher program CRUD
- [x] AC-3: Bulk generate + download CSV
- [x] AC-4: Frozen accounts list dengan WA reminder action

**Files:**
- `frontend/superadmin-web/` — Vue 3 frontend
- `services/billing-service/main.go` — dashboard APIs

**Notes:** API Gateway inject `X-User-Role: superadmin` dari JWT claim.

---

## F006: Multi-Tenant WA Session Pool

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Setiap tenant punya WA session sendiri untuk chatbot.

**Spec:**
- Tabel `wa_sessions` store session per tenant
- Status: `connected`, `qr_pending`, `disconnected`
- WA Gateway handle multi-device
- Session di-manage via N8N workflow

**Acceptance Criteria:**
- [x] AC-1: Tenant punya dedicated WA session
- [x] AC-2: Session status trackable
- [x] AC-3: QR code generation per tenant

**Files:**
- `services/wa-gateway/main.go` — WA session management
- `shared/migrations/000029_n8n_queue_mode.up.sql` — schema

**Notes:** Saat dev lokal, hanya satu WA device aktif.

---

## F007: Chatbot with RAG

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** AI chatbot dengan Retrieval-Augmented Generation.

**Spec:**
- FAQ dan Products di-index ke pgvector
- Chatbot retrieve relevant context sebelum LLM call
- Configurable per tenant (LLM, prompt, escalation settings)
- N8N workflow: Config → Session → RAG → LLM → Save

**Acceptance Criteria:**
- [x] AC-1: FAQ/Products indexed ke vector store
- [x] AC-2: Chatbot retrieve relevant context
- [x] AC-3: Per-tenant chatbot config
- [x] AC-4: Multi-channel session (WA, web, etc)

**Files:**
- `apps/umkm/chatbot/main.go` — chatbot API
- `services/ai-gateway/main.go` — embeddings endpoint
- `n8n/workflows/rag_indexer.json` — index workflow
- `n8n/workflows/universal_chatbot.json` — chatbot workflow

**Notes:** Embeddings via OpenAI/Anthropic melalui ai-gateway.

---

## F008: Escalation to Chatwoot

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Chatbot escalation ke human agent via Chatwoot.

**Spec:**
- Trigger escalation berdasarkan keyword atau fallback
- Buat conversation di Chatwoot
- Transfer context (conversation history, customer info)
- Log escalation history

**Acceptance Criteria:**
- [x] AC-1: Auto-escalation based on config
- [x] AC-2: Conversation created in Chatwoot
- [x] AC-3: Context transferred to agent
- [x] AC-4: Escalation history logged

**Files:**
- `n8n/workflows/escalation_handler.json` — escalation workflow
- `shared/migrations/000029_n8n_queue_mode.up.sql` — escalation_history table

**Notes:** Chatwoot running di port 3000 (docker-compose).

---

## F009: N8N Queue Mode Automation

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** N8N dengan Redis queue untuk horizontal scaling.

**Spec:**
- N8N Main: UI + Webhook Receiver + Workflow Editor
- N8N Worker: Execution Worker (scalable)
- Redis DB 2: Bull Queue untuk job distribution
- 8 workflows configured

**Workflows:**
| Workflow | Trigger | Purpose |
|:---------|:--------|:--------|
| `universal_chatbot.json` | Webhook | Multi-tenant chatbot |
| `rag_indexer.json` | Webhook | Index FAQ/Products |
| `escalation_handler.json` | Webhook | Escalation to Chatwoot |
| `master_automations.json` | Cron (1m) | Execute due automations |
| `daily_revenue_digest.json` | Cron | Revenue digest to Telegram |
| `freeze_reminder.json` | Cron | Expired subscription reminder |
| `campaign_voter_onboard.json` | Webhook | Voter onboarding |
| `voucher_wa_distribute.json` | Webhook | Voucher WA distribution |

**Acceptance Criteria:**
- [x] AC-1: N8N running dengan queue mode
- [x] AC-2: Redis queue configured
- [x] AC-3: All 8 workflows deployed

**Files:**
- `docker-compose.yml` — n8n-main, n8n-worker, redis config
- `n8n/workflows/*.json` — workflow definitions
- `infra/postgres/init.sql` — auto-create `wch_n8n` database
- `.env` / `.env.example` — `N8N_DB_*`, `N8N_ENCRYPTION_KEY` vars

**Notes:** Worker auto-configure dari shared database — scaling tinggal `docker-compose up -d --scale n8n-worker=N`. Persistence via dedicated `wch_n8n` database, backup: `pg_dump wch_n8n > n8n_backup.sql`.

---

## F010: Campaign Volunteer Management

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Manajemen volunteer untuk campaign.

**Spec:**
- CRUD volunteer dengan role (ketua, saksi, dll)
- Assign volunteer ke TPS/area
- Track volunteer activity
- Encrypted NIK storage

**Acceptance Criteria:**
- [x] AC-1: Volunteer CRUD
- [x] AC-2: Volunteer assignment to area
- [x] AC-3: NIK encrypted at rest
- [x] AC-4: Activity tracking

**Files:**
- `apps/campaign/api/handlers/volunteer.go`
- `apps/campaign/api/main.go`

**Notes:** NIK di-encrypt AES-256-GCM, key dari `cfg.EncryptionKey`.

---

## F011: Campaign Voter Onboarding

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Voter registration via webhook dari N8N.

**Spec:**
- N8N workflow trigger voter onboarding
- Voter data di-encrypt sebelum save
- Link voter ke TPS

**Acceptance Criteria:**
- [x] AC-1: Webhook endpoint untuk voter creation
- [x] AC-2: Voter data encrypted
- [x] AC-3: TPS assignment
- [x] AC-4: Bulk import support

**Files:**
- `apps/campaign/api/handlers/voter.go`
- `n8n/workflows/campaign_voter_onboard.json`

**Notes:** Bulk import via CSV dengan async processing.

---

## 📍 Lokasi Kode (Quick Reference)

### Mau Tambah Endpoint/API?

```
UMKM Accounting    ──→ apps/umkm/accounting/main.go (flat pattern)
UMKM Business      ──→ apps/umkm/business/main.go (flat pattern)
UMKM Chatbot       ──→ apps/umkm/chatbot/main.go (flat pattern)
UMKM Automation    ──→ apps/umkm/automation/main.go (worker)

Campaign API       ──→ apps/campaign/api/handlers/<nama>.go
                     + daftarkan di apps/campaign/api/main.go

Auth Service       ──→ services/auth-service/main.go
AI Gateway         ──→ services/ai-gateway/main.go
Billing Service    ──→ services/billing-service/main.go
WA Gateway         ──→ services/wa-gateway/main.go
Notification       ──→ services/notification-service/main.go
API Gateway        ──→ services/api-gateway/main.go
```

### Mau Tambah Tabel Database?

```bash
# Cek nomor terakhir:
ls shared/migrations/*.up.sql | tail -1

# Buat migration baru:
shared/migrations/NNNNNN_nama_fitur.up.sql
shared/migrations/NNNNNN_nama_fitur.down.sql
```

### Mau Tambah Config?

```
1. shared/sdk/config/config.go  ← Tambah field + LoadConfig()
2. .env.example                 ← Tambah dengan contoh nilai
3. docker-compose.yml           ← Tambah env var
```

### Mau Tambah UI Frontend?

```
UMKM      ──→ frontend/umkm-web/src/components/<Nama>.vue
Campaign  ──→ frontend/campaign-web/src/
Superadmin ──→ frontend/superadmin-web/src/
```

### Mau Tambah Service/Worker?

```
Wajib update:
☐ Makefile
☐ Dockerfile
☐ docker-compose.yml
☐ services/api-gateway/main.go (jika REST API)
☐ CLAUDE.md (Port Registry)
☐ .env.example
```

---

## 🔧 Cara Menambah Feature Baru

1. **Tambah SPEC entry** di section ini dengan format:
   ```
   ### F012: [Nama Feature]
   **Spec Status:** ⏳ Draft
   **Implementation:** ⏳ Pending
   ...
   ```

2. **User approve** — tambahkan comment atau ubah status ke "✅ Approved"

3. **AI implement** — setelah approved, AI coding berdasarkan SPEC

4. **Update implementation status** — ubah ke "✅ Done" setelah selesai

5. **Update Feature Registry table** di atas

---

*Lihat [CONTRIBUTING.md](../CONTRIBUTING.md) untuk panduan coding.*