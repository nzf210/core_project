# Product Requirements Document (PRD) — AI Agent UMKM & Pembukuan

## 1. Meta Information
- **Project Name:** AI Agent UMKM & Pembukuan (`apps/umkm`)
- **Target Audience:** Pemilik Usaha Mikro, Kecil, dan Menengah (UMKM) yang ingin mengotomatiskan pembukuan akuntansi, memiliki asisten kasir/operasional pintar, serta sistem Point of Sale (POS) cerdas.
- **Status:** **Production Ready (MVP & Advanced Features Active)**
- **Last Updated:** 2026-05-28

---

## 2. Product Overview
Sistem SaaS All-in-One cerdas berbasis multi-tenant yang dirancang khusus untuk mempermudah operasional dan pembukuan keuangan UMKM secara otomatis. Platform ini menggabungkan:
1. **Double-Entry Accounting Core** yang patuh pada standar SAK-EMKM/GAAP.
2. **AI Omni-Channel Chatbot** (WhatsApp & Web UI) yang terintegrasi dengan Redis Queue (100 concurrent workers) dan LLM MiniMax M2.7 untuk melayani pelanggan secara cerdas serta mendukung *Conversational Accounting*.
3. **Point of Sale (POS) & Checkout** dengan opsi Cash dan QRIS Dinamis (dilengkapi kalkulasi otomatis CRC16-CCITT dan webhook Xendit).
4. **Multi-Tenant Onboarding & Subscriptions** terbagi berdasarkan 7 tipe bisnis khusus dengan tampilan dashboard widget dinamis dan batasan kuota layanan (*Quota Middleware*).

---

## 3. Architecture & Services Blueprint
Sistem dideploy dalam 3 layanan mikro (*microservices*) utama di bawah namespace `apps/umkm`:

```
┌────────────────────────────────────────────────────────────────────────┐
│                          UMKM SaaS Monorepo                            │
├───────────────────┬────────────────────────────┬───────────────────────┤
│ Business Service  │     Accounting Engine      │    Chatbot Service    │
│    (Port 9001)    │        (Port 8201)         │      (Port 8202)      │
├───────────────────┼────────────────────────────┼───────────────────────┤
│ • Tenant Onboard  │ • Jurnal, COA, & Akuntansi │ • Web Chat UI         │
│ • Business Types  │ • Catalog & Stock Products │ • WA Webhook (Fonnte) │
│ • Custom Widgets  │ • Checkout (Cash/QRIS)     │ • Redis Job Queue     │
│ • Subscription/   │ • Webhook Xendit QRIS      │ • Dynamic RAG Prompt  │
│   Quota Middleware│ • OCR Receipt, FAQs, &     │ • Auto JSON Parsing   │
│                   │   Admin Forwarders         │ • Forward to Admin    │
└───────────────────┴────────────────────────────┴───────────────────────┘
```

---

## 4. Key Features & Technical Specifications

### Feature 1: Multi-Tenant Onboarding & Dynamic Dashboard Widget
- **Deskripsi:** Proses onboarding instan yang menyesuaikan modul aktif dan tata letak dashboard berdasarkan jenis usaha pengguna.
- **Kategori Bisnis Didukung:**
  - **Warung** (Fokus pada: Bagan penjualan hari ini, item terlaris, notifikasi stok menipis, pendapatan hari ini).
  - **Laundry** (Fokus pada: Lacak pesanan aktif, pendapatan harian, chart jenis paket, total pelanggan).
  - **Toko Online** (Fokus pada: Volume pesanan 7 hari, analisis kanal penjualan, tren pendapatan 30 hari, pengiriman tertunda).
  - **Restoran** (Fokus pada: Pendapatan harian, menu terpopuler, rasio biaya, jam sibuk, table turnover).
  - **Jasa** (Fokus pada: Daftar janji hari ini, pendapatan layanan, layanan terpopuler, retensi pelanggan, utilisasi staff).
  - **Industri Kreatif** (Fokus pada: Kanban proyek aktif, margin proyek, spend bahan, status invoice).
  - **Umum** (Laba rugi bulanan, ringkasan pengeluaran, aktivitas transaksi terbaru).
- **Sistem Langganan:** Tiering akun (Free, Starter, Pro, Enterprise) yang dikontrol ketat melalui `QuotaMiddleware` untuk membatasi jumlah transaksi, produk, atau akses asisten AI.

### Feature 2: Double-Entry Accounting (SAK-EMKM)
- **Deskripsi:** Mesin akuntansi dasar otomatis yang mematuhi standar SAK-EMKM.
- **Spesifikasi Teknis:**
  - **Seeding COA Otomatis:** Akun standar EMKM langsung dibuat saat tenant terdaftar (Kas, Bank/QRIS, Piutang, Persediaan, Hutang, Modal, Pendapatan Usaha, Beban Operasional).
  - **Jurnal Validasi:** Transaksi wajib memiliki nilai seimbang (`debit == credit`) dan tidak boleh bernilai 0 rupiah.
  - **Laporan Keuangan Real-Time:** 
    - Laporan Laba Rugi (`/reports/income-statement`)
    - Neraca (`/reports/balance-sheet`)
    - Laporan Arus Kas (`/reports/cash-flow`)

### Feature 3: Point of Sale (POS) & Dynamic QRIS Checkout
- **Deskripsi:** Antarmuka kasir mini untuk pencatatan transaksi langsung dengan dukungan checkout multi-metode pembayaran.
- **Spesifikasi Teknis:**
  - **Manajemen Produk:** Pengelolaan stok, harga (dalam satuan sen/int64 untuk akurasi finansial), kategori, foto produk utama, serta galeri foto tambahan.
  - **Catalog Import/Export:** Kemudahan migrasi data katalog menggunakan format file CSV.
  - **Dynamic QRIS Generator:** Menghasilkan QRIS Dinamis secara aman dengan melakukan kalkulasi **CRC16-CCITT** di backend menggunakan QRIS Statis merchant & nominal belanja.
  - **Payment Webhook:** Integrasi penuh dengan webhook Xendit untuk memverifikasi pembayaran secara real-time dan otomatis memposting jurnal transaksi ke COA `101 (Bank / QRIS)` setelah sukses.

### Feature 4: AI Conversational Accounting & WhatsApp Agent
- **Deskripsi:** Chatbot cerdas yang bertindak sebagai customer service untuk pembeli sekaligus asisten CFO untuk pemilik toko, dapat diakses via WhatsApp (Fonnte) & Web UI.
- **Spesifikasi Teknis & RAG:**
  - **High Scalability Queue:** Webhook WA diantrekan secara asinkron menggunakan Redis Queue (`chatbot:queue`) dengan pool **100 Concurrent Workers** guna menghindari timeout dari penyedia API WA.
  - **Dynamic RAG Prompting:** Sistem menyuntikkan katalog produk, Chart of Accounts (COA), dan data FAQ internal toko secara dinamis ke system prompt AI MiniMax M2.7 di setiap giliran chat.
  - **Role-Based Security:** Prompt AI disesuaikan berdasarkan peran pengguna (Kasir/Staff dibatasi untuk tidak bisa melihat laba/rugi, modal, atau neraca toko; Customer Service dibatasi hanya menjawab katalog produk/FAQ).
  - **Conversational Accounting Jurnal:** Pemilik dapat mencatat keuangan hanya lewat teks biasa (contoh: *"Beli minyak goreng 50rb pakai uang laci"*). AI akan merumuskan format transaksi JSON di balik layar, memposting ke jurnal akuntansi, dan mengonfirmasi keberhasilan pencatatan.
  - **Eskalasi Otomatis:** Deteksi keluhan emosional pelanggan atau pertanyaan di luar cakupan FAQ menggunakan penanda `[FORWARD_TO_ADMIN]`. Pesan eskalasi akan otomatis diteruskan ke nomor WhatsApp Admin/Staff yang terdaftar di `tenant_forwarders`.

### Feature 5: Automation & Background Insights
- **Deskripsi:** Latar belakang worker yang memproses analisis data berkala.
- **Spesifikasi Teknis:**
  - **Redis Pub/Sub Events:** Worker (`apps/umkm/automation`) berlangganan ke channel `tenant_events`.
  - **Automated AI Report:** Ketika menerima event `monthly_report`, sistem otomatis memicu LLM AI untuk menganalisis performa keuangan tenant dalam sebulan, merumuskan PDF laporan ringkasan eksekutif, dan menyimulasikan pengiriman email ke pemilik.

---

## 5. Technical Stack & Dependencies
- **Backend:** Go (Golang) menggunakan `net/http` dan router bawaan untuk kesederhanaan & efisiensi performa.
- **Database:** PostgreSQL dengan driver `github.com/jackc/pgx/v5` (tanpa ORM GORM untuk menjamin ACID transaksi finansial & kecepatan).
- **Caching & Queue:** Redis Cluster (Queue WA, Pub/Sub Automation).
- **AI Integration:** Panggilan terpusat ke `services/ai-gateway` (Port 8003) untuk memicu model `MiniMax-M2.7`.
- **Payment Gateway:** Xendit Go SDK (`github.com/xendit/xendit-go/v6`).

---

## 6. Release Phases (Actual Evolution)

| Fase | Rincian Kesiapan Fitur | Status |
| :--- | :--- | :--- |
| **Fase 1 (Core)** | Onboarding tenant, COA EMKM, Jurnal Ledger, Laba Rugi & Neraca. | **100% Selesai & Stabil** |
| **Fase 2 (AI & WA)** | Integrasi AI Gateway, Redis Async Queue (100 workers), RAG Katalog & FAQs, Eskalasi `[FORWARD_TO_ADMIN]`. | **100% Selesai & Stabil** |
| **Fase 3 (POS & FinTech)**| Manajemen produk (import/export CSV), POS Checkout, Kalkulasi Dynamic QRIS CRC16, Webhook Xendit. | **100% Selesai & Stabil** |
| **Fase 4 (Automation)** | Background worker Pub/Sub, email laporan bulanan otomatis oleh AI. | **100% Selesai & Stabil** |
| **Fase 5 (Refinement)** | Implementasi detail Laporan Arus Kas (`/reports/cash-flow`) dan perbaikan unit test. | **Dalam Pengerjaan 🚧** |
