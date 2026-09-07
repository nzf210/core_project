# 🎯 WCH Platform — Project Goals & Anti-Drift Framework

> **Dokumen Sasaran Utama (North Star) & Panduan Fokus WCH Platform.**
> Dokumen ini adalah acuan tertinggi untuk menjaga konsistensi fokus seluruh pengembangan, mencegah *goal drift*, *scope creep*, dan deviasi arsitektur.

---

## 🧭 1. North Star & Identitas Proyek

**WCH Platform** adalah SaaS multi-produk berbasis **Go (Golang)** berarsitektur monorepo tunggal (`go.mod`).
Tujuan utama platform ini adalah menyediakan solusi operasional digital enterprise-grade yang terjangkau, aman, dan terotomasi untuk dua segmen pasar utama di Indonesia:

1. **UMKM (Usaha Mikro, Kecil, dan Menengah)**: AI Agent otomatisasi bisnis, akuntansi double-entry, POS, dan CS WhatsApp AI cerdas.
2. **Campaign (Manajemen Elektoral & Politik)**: Manajemen relawan berjenjang, real count suara pemilu, verifikasi pemilih terenkripsi, dan AI sentiment analysis.

Semua produk berbagi infrastruktur bersama (**Shared Core Services**) untuk autentikasi multi-tenant, billing terpadu, WhatsApp gateway, AI gateway, dan asynchronous job processing.

---

## 🌳 2. Hierarki Tujuan (Goal Tree)

```
                       ┌──────────────────────────────────────────────┐
                       │           WCH PLATFORM NORTH STAR            │
                       │ Multi-Tenant SaaS Terpadu (Go Monorepo)       │
                       └──────────────────────┬───────────────────────┘
                                              │
         ┌────────────────────────────────────┼────────────────────────────────────┐
         │                                    │                                    │
         ▼                                    ▼                                    ▼
┌──────────────────┐               ┌──────────────────┐               ┌──────────────────┐
│   PILAR 1: UMKM  │               │ PILAR 2: CAMPAIGN│               │ PILAR 3: SHARED  │
│  (apps/umkm/)    │               │ (apps/campaign/) │               │   (services/ &   │
│                  │               │                  │               │     shared/)     │
│ • Accounting &   │               │ • Koordinator &  │               │ • API Gateway    │
│   Double-Entry   │               │   Relawan Ber-   │               │ • Auth & RBAC    │
│ • POS & Klinis   │               │   jenjang        │               │ • Hybrid WA      │
│ • Hybrid WA AI   │               │ • Real Count &   │               │ • AI Gateway     │
│   Chatbot (N8N)  │               │   Audit Suara    │               │ • Xendit Billing │
│ • Auto Job Worker│               │ • NIK Encryption │               │ • RabbitMQ Queue │
└──────────────────┘               └──────────────────┘               └──────────────────┘
         │                                                                     │
         └─────────────────────────────────┬───────────────────────────────────┘
                                           │
                                           ▼
                       ┌──────────────────────────────────────┐
                       │     BATASAN NEGATIF (OUT-OF-SCOPE)   │
                       │ ❌ Crypto Bot (ARCHIVED)              │
                       │ ❌ Direct External LLM Calls         │
                       │ ❌ Single-tenant / Unscoped queries  │
                       │ ❌ Monolithic files (>450 lines BE)  │
                       └──────────────────────┘
```

---

## 📌 3. Pilar Produk Aktif & Sasaran Spesifik

### Pilar A: UMKM (`apps/umkm/`)
- **Accounting Engine (`apps/umkm/accounting`)**:
  - Sistem pembukuan double-entry berstandar akuntansi.
  - Manajemen POS, mutasi kas, invoice, dan spesialisasi klinik (`clinic`).
  - Uang/harga wajib bertipe `int64` satuan **sen** (1 rupiah = 100 sen).
- **AI Chatbot & CRM (`apps/umkm/chatbot` + `services/wa-gateway` + N8N)**:
  - Layanan CS otomatis multi-tenant 24/7.
  - Hybrid WhatsApp routing: Cloud API (Meta Official) untuk blast/transaksional; whatsmeow untuk percakapan 1-on-1 & OTP login staff.
  - Guard aktivasi WA aktif (F048 AC-8) sebelum chatbot bisa di-enable.
- **Automation & Background Worker (`apps/umkm/automation`)**:
  - Pemrosesan batch transaksi, distribusi voucher, dan digest harian via RabbitMQ.

### Pilar B: Campaign & Pemilu (`apps/campaign/`)
- **Struktur Relawan Berjenjang**:
  - Hierarki: `korprov` → `korKab` → `korKec` → `korKades` → `saksi_tps`.
- **Keamanan Data Pemilih**:
  - NIK wajib terenkripsi AES-256-GCM (`encrypted_nik`).
- **Real Count & Quick Count**:
  - Verifikasi suara TPS dengan upload C1 Plano dan audit trail.

### Pilar C: Shared Platform Core (`services/` & `shared/`)
- **API Gateway (8000)**: Single entry point, routing, reverse proxy.
- **Auth Service (8001)**: Multi-tenant JWT, login OTP via WA & Telegram, impersonasi Superadmin, 3 tier pengguna (Superadmin, Tenant Owner, Staff).
- **AI Gateway (8002)**: Proxy terpusat ke MiniMax M2.7, semantic caching Redis, kuota & billing log per-tenant.
- **Billing Service (8003)**: Integrasi Xendit per-tenant, manajemen paket langganan (Lite, Pro, Ultimate), dan siklus voucher.
- **WA Gateway (8202) & Cloud API (8210)**: Abstraksi WhatsApp hybrid dengan rate-limiting dan auto-reconnect.
- **RabbitMQ Queue (`shared/sdk/queue`)**: Asynchronous worker pipeline untuk operasi berat.

---

## 🚫 4. Boundary & Out-of-Scope (Batas Tegas)

1. **Crypto Trading Bot (`apps/crypto/`) — ARCHIVED**:
   - DILARANG menambah fitur baru, refactor, atau mengubah modul crypto kecuali ada perintah eksplisit dari Owner proyek.
2. **Larangan Direct LLM**:
   - TIDAK BOLEH memanggil OpenAI/MiniMax/Gemini langsung dari `apps/`. Wajib selalu lewat `services/ai-gateway`.
3. **Larangan Floating Point Uang**:
   - DILARANG menggunakan `float64` untuk nominal rupiah. Wajib `int64` sen.
4. **Larangan Hardcode & String Query**:
   - Wajib parameterized query SQL. Tidak ada string concatenation SQL.

---

## 🛡️ 5. Standar Eksekusi & Anti-Drift Protocol

Setiap kali AI menerima instruksi tugas, ikuti siklus 4 langkah ini:

```
1. ANCHOR GOAL          2. GRAPHIFY QUERY        3. SPEC & EXECUTE       4. VERIFY & SYNC
   Cek apakah tugas        Cari node terkait        Ikuti SPEC approved     Test (make check)
   sesuai pilar aktif      di graphify-out/         Jaga batas file size    Update FEATURE_MAP
   (UMKM / Campaign)       Petakan dependensi       BE <= 450, FE <= 500    graphify update .
```

1. **Anchor**: Pastikan pekerjaan berkontribusi pada salah satu pilar aktif (UMKM, Campaign, atau Shared Core). Tolak atau klarifikasi jika menyimpang ke modul yang di-archive.
2. **Orientasi Graphify**: Jalankan `graphify query "<konsep/fitur>"` sebelum mengubah kode untuk melihat dependensi, god nodes, dan komunitas arsitektur terkait.
3. **Patuhi Batas File**: Jangan pernah melanggar batas SonarQube: backend Go maksimal 450 baris per file, frontend Vue maksimal 500 baris per file.
4. **Validasi & Verifikasi**: Jalankan unit test, build check, perbarui `docs/FEATURE_MAP.md`, dan perbarui knowledge graph dengan `graphify update .`.
