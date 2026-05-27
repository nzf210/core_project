# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** AI Agent UMKM & Pembukuan (`apps/umkm`)
- **Target Audience:** Pemilik Usaha Mikro, Kecil, dan Menengah (UMKM) yang kesulitan mengelola pencatatan keuangan dan pelayanan pelanggan.
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Sistem cerdas all-in-one untuk UMKM yang menggabungkan asisten operasional (Omni-channel chatbot) dan pembukuan keuangan otomatis (Double-entry accounting). Produk ini mengatasi beban operasional UMKM dengan mengizinkan AI untuk melayani chat pelanggan sekaligus membebaskan UMKM dari entri manual pembukuan melalui pencatatan suara, teks, dan pindai nota otomatis.

## 3. Goals & Objectives
- **Business Goals:** Menjadi platform SaaS dominan untuk UMKM lokal melalui kemudahan penggunaan berbasis AI.
- **User Goals:** Memiliki asisten 24/7 yang bisa menjawab pelanggan sekaligus mencatat keuangan dengan bahasa manusia biasa tanpa perlu ilmu akuntansi.
- **Non-Goals:** Bukan sistem ERP yang kompleks; difokuskan untuk kemudahan dan otomatisasi (low touch, high impact).

## 4. Key Features & Requirements

### Feature 1: Double-Entry Accounting Core
- **Description:** Mesin akuntansi dasar yang sesuai dengan standar akuntansi (GAAP/SAK-EMKM).
- **Acceptance Criteria:** Mampu memproses debit/kredit, memiliki Chart of Accounts (COA) template untuk UMKM, dan menghasilkan Laba Rugi, Arus Kas, dan Neraca secara otomatis.
- **Priority:** P1 (High)

### Feature 2: AI Omni-Channel Chatbot (RAG)
- **Description:** Asisten pintar di WhatsApp/Telegram/IG untuk UMKM dan Pelanggan.
- **Acceptance Criteria:** Pelanggan UMKM dapat bertanya produk/stok dan memesan. Pemilik UMKM dapat menanyakan omzet hari ini (System Prompt membaca data pembukuan internal).
- **Priority:** P1 (High)

### Feature 3: Conversational Accounting
- **Description:** Pencatatan transaksi via NLP (suara/teks).
- **Acceptance Criteria:** Pemilik dapat mengetik "Beli bahan baku tepung 100 ribu" dan AI otomatis mengonversinya ke jurnal debit/kredit yang tepat di sistem akuntansi.
- **Priority:** P1 (High)

### Feature 4: AI OCR Receipt & Invoice Scanner
- **Description:** Ekstraksi data otomatis dari foto nota/struk belanja.
- **Acceptance Criteria:** AI dapat mendeteksi nominal, tanggal, kategori, dan vendor dari gambar lalu membuat draft jurnal pembukuan.
- **Priority:** P2 (Medium)

### Feature 5: Automation & AI Insights
- **Description:** Background worker untuk tugas otomatisasi dan analisis kesehatan kas.
- **Acceptance Criteria:** Sinkronisasi n8n untuk notifikasi tagihan/resi kurir, dan AI alert jika cash flow negatif.
- **Priority:** P3 (Low)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** `ai-gateway` (untuk RAG dan NLP), n8n (workflow service), WhatsApp API.
- **Database:** PostgreSQL (multi-tenant schema) untuk data transaksi finansial (ACID compliance sangat krusial).
- **Security & Performance:** Isolasi data tenant yang sangat kuat untuk mencegah bocornya data finansial antar toko.

## 6. User Experience & Design
- **Platform:** Web Dashboard & Chat-based Interface (WhatsApp/Telegram).
- **Key Flows:** Setup Toko -> Mulai Chat dengan Bot (Catat pengeluaran) -> Dashboard Finansial update real-time.

## 7. Metrics & Analytics
- **Success Metrics:** Jumlah tenant aktif, Rata-rata transaksi tercatat per pengguna/hari, Persentase keberhasilan parsing NLP transaksi.

## 8. Release Phases
- **Phase 1 (MVP):** Pencatatan manual, Laba Rugi dasar, Chatbot WA FAQ dasar.
- **Phase 2:** Conversational accounting (teks to journal), OCR Scanner, Integrasi otomatis n8n.
