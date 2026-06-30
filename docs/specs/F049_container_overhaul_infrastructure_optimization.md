# F049: Container Overhaul & Infrastructure Optimization


### 📊 Backend: New Endpoint

#### `GET /api/umkm/reports/sales-chart?period=week|month|year`

**Response shape:**
```json
{
  "success": true,
  "data": {
    "period": "month",
    "labels": ["1 Juni", "2 Juni", ...],
    "revenue": [500000, 750000, 620000, ...],
    "expense": [200000, 300000, 250000, ...],
    "profit": [300000, 450000, 370000, ...]
  }
}
```

**SQL logic:**
```sql
-- Period = 'week' (7 hari terakhir, group by date)
SELECT DATE(created_at) as day,
       SUM(CASE WHEN type = 'credit' AND account_code ~ '^4' THEN amount ELSE 0 END) as revenue,
       SUM(CASE WHEN type = 'debit' AND account_code ~ '^5' THEN amount ELSE 0 END) as expense
FROM journal_entries
WHERE tenant_id = $1
  AND created_at >= NOW() - INTERVAL '7 days'
GROUP BY DATE(created_at)
ORDER BY day

-- Period = 'month' (30 hari)
-- Period = 'year' (12 bulan, GROUP BY month)
```

Atau, reuse data dari `income-statement` yang sudah ada — hitung per-hari dari data transaksi yang sudah di-aggregate.

#### `GET /api/umkm/reports/top-products?limit=5&from=&to=`

**Response:**
```json
{
  "success": true,
  "data": [
    { "name": "Nasi Goreng", "quantity": 42, "revenue_cents": 840000 },
    { "name": "Es Teh", "quantity": 38, "revenue_cents": 190000 }
  ]
}
```

**SQL logic:**
```sql
-- Dari journal_entries, filter debit ke account pendapatan (4xx)
-- Join dengan description / metadata untuk extract product name
-- Atau dari transaction_items jika ada
```

Jika tabel `transaction_items` tidak ada, gunakan pattern matching dari `journal_entries.description` (approximate).

#### `GET /api/umkm/reports/recent-transactions?limit=5`

**Response:**
```json
{
  "success": true,
  "data": [
    { "id": "...", "date": "2026-06-22", "description": "Penjualan Tunai", "amount_cents": 50000, "type": "income" }
  ]
}
```

**SQL:**
```sql
SELECT id, created_at, description, amount_cents, entry_type
FROM journal_entries
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2
```

### 🖥️ Frontend Changes

#### DynamicDashboard.vue — 3 perubahan

**1. Chart widget (type: 'chart') — render Chart.js real:**
```
<template>
  <div class="widget-chart">
    <div class="chart-header">
      <span class="widget-title">{{ widget.title }}</span>
      <div class="period-switcher" v-if="widget.id === 'daily_sales' || widget.id === 'order_volume'">
        <button :class="{ active: period === 'week' }" @click="setPeriod('week')">7H</button>
        <button :class="{ active: period === 'month' }" @click="setPeriod('month')">30H</button>
        <button :class="{ active: period === 'year' }" @click="setPeriod('year')">12B</button>
      </div>
    </div>
    <div style="height: 200px; width: 100%;">
      <Line v-if="chartReady" :data="chartData" :options="chartOptions" />
      <div v-else class="chart-loading">Memuat data...</div>
    </div>
  </div>
</template>
```

**2. List widget (type: 'list', id: 'recent_transactions') — fetch real data:**
```
GET /api/umkm/reports/recent-transactions?limit=5
→ render: description | amount | relative time
```

**3. List widget (id: 'best_selling', 'top_products') — fetch real data:**
```
GET /api/umkm/reports/top-products?limit=5
→ render: product name | qty sold | total revenue
```

**4. Metric widget enhancement:**
- Tambah loading state (skeleton)
- Tambah tooltip "vs bulan lalu" pada perubahan persen

#### Dashboard.vue (Classic) — period switcher

Classic Dashboard sudah punya Bar chart `handleCashFlow` data. Tambah period switcher:
- Tombol "Minggu Ini" / "Bulan Ini" / "Tahun Ini"
- Refetch chart data saat period berubah

### ✅ Acceptance Criteria (AC)

- [x] AC-1: `GET /reports/sales-chart?period=week` → return 7 data points (per-hari)
- [x] AC-2: `GET /reports/sales-chart?period=month` → return 30 data points
- [x] AC-3: `GET /reports/sales-chart?period=year` → return 12 data points (per-bulan)
- [x] AC-4: Dashboard widget `daily_sales` menampilkan Chart.js bar chart real
- [x] AC-5: Period switcher (7H/30H/12B) berfungsi — ganti data chart
- [x] AC-6: Widget `recent_transactions` menampilkan 5 transaksi terakhir real
- [x] AC-7: Widget `top_products` menampilkan produk terlaris real
- [x] AC-8: Loading state (spinner) selama fetch
- [x] AC-9: Empty state jika belum ada transaksi
- [x] AC-10: `go build ./...`, `go vet`, `vue-tsc` clean ✅

### 📁 Files Changed

**Backend:**
- `apps/umkm/accounting/main.go` — **NEW** handler `handleSalesChart` (GET /reports/sales-chart), `handleTopProducts`, `handleRecentTransactions` + 3 routes

**Frontend:**
- `frontend/umkm-web/src/components/DynamicDashboard.vue` — Chart.js real data, period switcher (7H/30H/12B), loading states, real list data
- `frontend/umkm-web/src/api.ts` — `reportsApi` with `getSalesChart`, `getTopProducts`, `getRecentTransactions`

### Notes:

- `vue-chartjs` + `chart.js` sudah terinstall ✅ — tidak perlu npm install baru
- Endpoint `sales-chart` bisa reuse aggregasi dari `income-statement` tetapi di-breakdown per-day
- Untuk produk terlaris: jika tabel `transaction_items` tidak ada, gunakan journal entries description matching sebagai aproximasi
- Period switcher state disimpan di `ref()` lokal, tidak perlu localStorage
- Dark/light mode: Chart.js label colors harus adjust — gunakan CSS variable atau computed property yang read theme


## F049: Container Overhaul & Infrastructure Optimization
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Standarisasi penamaan container (`wch-` prefix), pengurangan duplikasi replika yang tidak perlu di environment dev, serta penambahan koneksi pool database (pgBouncer) untuk mencegah exhaustion koneksi pada arsitektur monorepo shared.

**Acceptance Criteria (AC):**
- [x] Semua container menggunakan `wch-` prefix secara eksplisit di `docker-compose.yml`.
- [x] Duplikasi replika container `wa-gateway` dan `umkm-chatbot` diturunkan menjadi 1.
- [x] Container pgbouncer ditambahkan di port 6432.
- [x] Environment variable koneksi PostgreSQL dari seluruh service diubah dari host `postgres:5432` menjadi `pgbouncer:6432`.

## F051: AI Quota Per-Modalitas (Text/Vision/Image)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Membedakan quota rate-limit untuk penggunaan AI berdasarkan modalitasnya. Saat ini seluruh request ke `ai-gateway` menghabiskan 1 pool quota yang sama, menyebabkan risiko perebutan resource antara chatbot (text) dan fitur OCR Campaign (vision).

**Tujuan:**
- Mencegah starvation quota dari layanan yang murah (text chatbot) terhadap layanan yang mahal (vision OCR, image generation).
- Menerapkan pembatasan `plan_features` secara lebih granular: `ai_text`, `ai_vision`, `ai_image`.

**Acceptance Criteria (AC):**
- [x] Database migration untuk menambahkan feature keys baru di `plan_features`.
- [x] Implementasi routing key di `ai-gateway` middleware sesuai dengan modalitas endpoint.
- [x] Redis keys quota dihitung per modalitas: `quota_counter:{tenant}:{period}:{feature}` (feature = `ai_text` / `ai_vision` / `ai_audio_stt` / `ai_audio_tts` / `image_gen` / `ai_image`).

**Files:**
- `shared/migrations/000072_ai_image_modality.up.sql` — seed `ai_image` plan_feature key + per-tier numeric limits
- `services/ai-gateway/main.go` — per-modality quota routing (`ai_text`, `ai_vision`, `ai_audio_*`, `image_gen`)
- `services/ai-gateway/image.go` — increments `ai_image` counter on image generation
- `shared/sdk/auth/quota_counter.go` — `ai_image` → `MaxImageGen` mapping
- `services/ai-gateway/f050_modality_test.go` — modality routing key assertions
- `shared/sdk/auth/quota_counter_test.go` — `ai_image` limit coverage

## F050: WCH E2E MCP Server (UI Testing & Browser Automation)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Server MCP kustom untuk otomatisasi UI browser, enabling Hermes untuk melakukan testing end-to-end (E2E) dan pengecekan UI secara langsung di environment dev (localhost).

**Tujuan:**
- Memberikan Hermes kemampuan untuk melihat/berinteraksi dengan UI.
- Mempercepat testing alur pendaftaran, konfigurasi chatbot, dan dashboard.

**Acceptance Criteria (AC):**
- [x] AC-1: Server MCP berjalan (`node infra/mcp/wch-e2e-server.js`) dan terintegrasi ke Hermes CLI.
- [x] AC-2: Implementasi tools: `e2e_navigate`, `e2e_click`, `e2e_fill`, `e2e_screenshot`, `e2e_expect_selector`.
- [x] AC-3: Flow testing: Login → Navigate `/chatbot-config` → Fill settings → Verify (contoh di README.md).

**Files:**
- `infra/mcp/wch-e2e-server.js` — Core MCP Server logic (stdio transport, single-page Playwright session).
- `infra/mcp/package.json` — Playwright + MCP SDK dependencies, `start` & `test` scripts.
- `infra/mcp/test-server.js` — Unit test (tool registry validation tanpa launch browser).
- `infra/mcp/README.md` — Quick start + AC-3 example flow.

## F033: Campaign Logistics Tracking

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Sistem anti-bocor logistik kampanye (kaos, sembako, baliho) dari gudang pusat hingga ke rumah warga/TPS, dipantau via WhatsApp Bot dengan validasi lokasi.

**Tujuan:**
- Memastikan dana kampanye yang dibakar untuk barang fisik benar-benar sampai ke target.
- Deteksi dini jika ada koordinator wilayah yang menahan logistik.

**Acceptance Criteria (AC):**
- [x] AC-1: `HandleDistributeLogistics` (logistics.go line 129) INSERT ke `logistic_distributions` dengan item, qty, receiver, lokasi ✅
- [x] AC-2: Status `received` hanya diset jika `proof_image_url` non-empty (logistics.go line 134-136) — selfie+location proof required ✅
- [x] AC-3: `HandleStalledDistributions` (logistics.go line 165) query distribusi dengan `last_status_change` > 2 hari ✅

## F034: Cost-per-Vote (Campaign Accounting)

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Integrasi *Accounting Engine* (yang sudah ada di UMKM) ke modul *Campaign* untuk melacak setiap Rupiah yang keluar dan membaginya dengan jumlah dukungan valid.

**Tujuan:**
- Menghitung *Cost-per-Vote* di setiap desa/kecamatan secara real-time.
- Mencegah pengeluaran kampanye di daerah yang sudah over-target (hijau).

**Spec:**
- **Database:** Tabel `campaign_expenses` (tenant_id, campaign_id, expense_category, amount, target_region_type, target_region_id, description) — migration `000074`
- **API Endpoint (`POST /campaign/expenses`):** Catat pengeluaran kampanye dengan region targeting opsional. Auto-sync ke UMKM Accounting engine.
- **Cost-per-Vote Calculation:** `total_expense / total_valid_endorsements` — dihitung real-time di `GET /campaign/finance`
- **Alert System:** `checkAndAlertCPV()` — jika CPV > Rp 200.000, tulis ke tabel `notifications` dengan type `cpv_alert`

**Acceptance Criteria (AC):**
- [x] AC-1: Aplikasi Campaign dapat memanggil API Accounting internal untuk mencatat pengeluaran kampanye.
- [x] AC-2: Perhitungan *Cost-per-Vote* = (Total Pengeluaran Daerah X) / (Total Endorsement Valid Daerah X).
- [x] AC-3: Jika Cost-per-Vote di suatu desa melampaui batas wajar (misal Rp 200.000/suara), sistem mengirimkan alert notifikasi.

**Files:**
- `shared/migrations/000074_f034_campaign_expenses.up.sql` — tabel `campaign_expenses`
- `apps/campaign/api/handlers/finance.go` — `HandleCampaignFinance` (GET CPV + POST expense), `checkAndAlertCPV()`, `syncExpenseToAccounting()`

## F035: Auto-Scan KTP (AI OCR Vision)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Relawan kirim foto KTP via WA. N8N kirim ke AI Gateway `/v1/vision` -> Ekstrak NIK, Nama, Alamat jadi JSON -> Masuk otomatis ke tabel `citizens`.
**Target:** Menghilangkan salah ketik NIK (Typo) oleh relawan.

## F036: Dashboard Sentimen Isu Harian (AI NLP)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Chat relawan dari lapangan diproses AI untuk mengekstrak kata kunci keluhan warga. Diagregrasi ke tabel `village_issues` (Desa, Isu, Sentimen -1 s/d +1).
**Target:** Bahan pidato spesifik per-desa untuk Kandidat.

## F037: Wargame & Simulasi Kemenangan (Predictive AI)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** UI Slider di Dashboard. Kalkulasi algoritma menggabungkan data `campaign_expenses` (uang dibakar) dengan rasio konversi `endorsements`. 
**Target:** Memprediksi probabilitas menang vs Cost-per-vote jika anggaran digeser ke daerah lain.

## F038: Peta Kerawanan & Pelaporan Pelanggaran
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Tabel `fraud_reports`. Relawan kirim "Share Loc" + Foto pelanggaran (Spanduk dirusak / serangan fajar lawan). Tampil sebagai titik MERAH di Heatmap UI.
**Target:** Bukti hukum siap lapor Bawaslu.

## F039: Pemilih Siluman & Anomali Detektor
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Job otomatis yang mem-flag `endorsements`. Syarat siluman: Usia > 100 thn, 1 relawan setor 500 KTP dalam 1 jam (indikasi bot/joki), kode wilayah NIK tidak cocok dengan TPS.
**Target:** Cleansing data agar kandidat tidak tertipu "Data Sampah" timses.

## F040: WA Blast Bertarget (Micro-targeting)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Filter query di Frontend (misal: "Wanita, Desa A, Pekerjaan Petani") -> Lempar payload ke N8N / WA Gateway untuk *bulk send*.
**Target:** Efisiensi kuota WA, pesan kampanye super personal.

**Acceptance Criteria:**
- [x] AC-1: `POST /blast/target` dengan filter `village_id`, `gender` (L/P), `age_range` (e.g. "18-25", "60+").
- [x] AC-2: Exclude anomaly-flagged endorsements (`is_anomaly = TRUE`).
- [x] AC-3: Return filtered phone list + `target_count`.

**Files:**
- `apps/campaign/api/handlers/blast.go` — `HandleBlastTarget` + `parseAgeRange`.
- `apps/campaign/api/handlers/blast_test.go` — Unit test for age range parser.

## F041: Gamification & Leaderboard Relawan
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Query agregat `COUNT(endorsements) GROUP BY recruiter_id`. Tampil di UI. Bot WA otomatis kirim ranking ke relawan tiap minggu.
**Target:** Memacu kompetisi antar relawan lapangan.

## F042: WA Bot FAQ Panduan Kampanye (RAG)
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Dokumen visi-misi paslon di-vectorize (pgvector `embeddings`). Jika warga/relawan tanya via WA, AI Gateway cari jawaban berbasis dokumen (RAG).
**Target:** Relawan lapangan selalu punya contekan cerdas.

**Acceptance Criteria:**
- [x] AC-1: `POST /bot/faq` dengan `question` → embed via AI Gateway `/v1/embeddings` → cosine similarity search di `vector_embeddings` (top-K=3).
- [x] AC-2: pgvector HNSW index untuk fast similarity.
- [x] AC-3: Fallback ke ILIKE keyword search kalau AI Gateway unavailable.
- [x] AC-4: Return `sources` (content + similarity) untuk transparansi.

**Files:**
- `shared/migrations/000071_campaign_rag_documents.{up,down}.sql` — `campaign_documents` table.
- `apps/campaign/api/handlers/faq.go` — `HandleBotFAQ` + `embedQuestion` + `vectorSearch` + `keywordSearch` + `synthesizeFallbackAnswer`.

## F043: Multi-Level Election & Sainte-Laguë Simulator
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Upgrade Campaign Engine agar mendukung pemilihan Legislatif (Pileg DPR/DPRD) dan DPD dengan kalkulasi perolehan kursi Sainte-Laguë yang realistik dan penanganan multi-dapil dalam satu dashboard.
**Target:** Menghitung probabilitas perolehan kursi real-time berdasarkan sisa suara & divisor Sainte-Laguë.

**Acceptance Criteria:**
- [x] AC-1: Sainte-Laguë divisor sequence (1, 3, 5, 7, ...) untuk seat allocation.
- [x] AC-2: Parliamentary threshold 4% per UU Pileg — parties di bawah threshold excluded.
- [x] AC-3: Multi-dapil dashboard: `GET /wargame/sainte-lague` tanpa `dapil_id` → list all tenant dapils.
- [x] AC-4: Single dapil detail: `GET /wargame/sainte-lague?dapil_id=...` → seat allocations + party standings (vote_share %, above_threshold bool).
- [x] AC-5: Final standings sorted by seats DESC, lalu votes DESC.

**Files:**
- `apps/campaign/api/handlers/pileg.go` — `HandleSainteLague` + `simulateAllDapils` + `simulateSingleDapil`.

## F044: Campaign Modular License & Payment System
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Monetisasi fitur Campaign via kombinasi Self-Service Payment Gateway (Xendit) untuk pembelian instan dan Manual License Key (Superadmin-generated) untuk transaksi B2B custom pricing.

**Acceptance Criteria:**
- [x] AC-1: `POST /billing/checkout` — generate mock Xendit invoice untuk `wargame_token` atau `intelligence_pack`.
- [x] AC-2: `POST /billing/webhook` — Xendit callback, idempotent (race-safe via `SELECT FOR UPDATE`), credit tokens / addons to campaign.
- [x] AC-3: Affiliate commission on PAID (referral config-driven rate, default 10%).
- [x] AC-4: `POST /superadmin/licenses/generate` — manual B2B license key.
- [x] AC-5: `GET /superadmin/licenses?used=&limit=` — list all licenses with usage status.
- [x] AC-6: `POST /licenses/redeem` — tenant burns license, atomic via tx.
- [x] AC-7: `GET /licenses/active?campaign_id=...` — return election_type, max_voters, wargame_tokens, active_addons per campaign.

**Files:**
- `apps/campaign/api/handlers/billing.go` — `HandleBillingCheckout` + `HandleBillingWebhook`.
- `apps/campaign/api/handlers/license.go` — `HandleSuperadminGenerateLicense` + `HandleRedeemLicense` + `HandleListLicenses` + `HandleTenantActiveAddons`.
- `apps/campaign/api/main.go` — routes registered.

## F045: UMKM Healthcare Clinic Queue System
**Spec Status:** ✅ Approved
**Implementation:** ✅ Done
**Deskripsi:** Modul reservasi antrian buat klinik UMKM. Sistem memberikan nomor antrian otomatis dan melakukan reminder via N8N WA Gateway.
**Fitur:** Backend (settings, book, cancel, queue, call) + N8N Workflows (booking_bot, reminder).

**Acceptance Criteria (AC):**
- [x] AC-1: Reservasi antrian (Book) dengan slot dinamis & validasi tipe antrian (Sequential/Timeslot).
- [x] AC-2: Pembatalan antrian via WA Bot.
- [x] AC-3: Notifikasi otomatis 1 jam sebelum jadwal via N8N WA Gateway.
- [x] AC-4: Call nomor antrian (Atomic increment counter).

**Files:**
- `apps/umkm/accounting/main.go` — handlers: `handleClinicBook`, `handleClinicCancel`, `handleClinicQueue`, `handleClinicCall`.
- `apps/umkm/accounting/clinic_middleware.go` — `requireClinicType` check.
- `apps/umkm/accounting/clinic_test.go` — Unit tests.


## F001: Webhook Subscription

## F027: Core Business Flow Fixes & Optimizations

**Spec Status:** ✅ Approved
**Implementation:** ✅ Done

**Deskripsi:** Perbaikan logika bisnis utama hasil audit keamanan dan flow transaksi untuk mencegah kerugian perusahaan dan tenant.

### Sub-Tasks:
1. **Accounting Hard-Delete Fix**:
   - `DELETE /api/umkm/accounts/{id}` harus memblokir penghapusan jika akun masih memiliki `journal_entries` atau balance `!= 0`. Kembalikan HTTP 400.
2. **Chatbot Instant Escalation**:
   - Deteksi fallback response di `apps/umkm/chatbot/main.go`. Jika AI mengeluarkan pesan fallback, langsung *trigger* escalation ke *owner* secara instan, bypass `AutoEscalateAfterMinutes`.
3. **AI Quota Race-Condition Fix**:
   - Modifikasi `QuotaMiddlewareFeature` di `shared/sdk/auth/quota_mw.go`. Lakukan check/increment kuota **SEBELUM** meneruskan ke handler API. Jika kuota habis, kembalikan `402 Payment Required` tanpa memanggil handler (vendor API).
4. **Billing Proration (Sisa Hari)**:
   - Modifikasi `activateSubscription` di `services/billing-service/main.go`. Hitung sisa hari dari subscription sebelumnya yang masih aktif. Berikan kompensasi (tambahan waktu atau konversi) pada subscription yang baru, atau setidaknya tidak menghilangkan masa aktif paket lama jika tidak prorata (contoh: masa aktif paket baru = hari ini + 30 + sisa hari paket lama yang sepadan/di-scale down, atau cukup tambahkan secara kasar).

**Acceptance Criteria (AC):**
- [x] AC-1: Accounting hard-delete block jika akun punya journal entries atau balance != 0
- [x] AC-2: Chatbot instant escalation on fallback via goroutine (bypass AutoEscalateAfterMinutes)
- [x] AC-3: AI Quota check/increment BEFORE handler via QuotaMiddlewareFeature middleware
- [x] AC-4: Billing proration — remaining days dari subscription lama ditambahkan ke voucher baru
