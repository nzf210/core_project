# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** UMKM & Accounting Web App (`frontend/umkm-web`)
- **Target Audience:** Pemilik UMKM (Usaha Mikro Kecil Menengah).
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Antarmuka pengguna (Frontend) untuk aplikasi UMKM. Aplikasi ini menyediakan dashboard kesehatan bisnis (laba rugi) dan antarmuka konfigurasi chatbot (Omni-channel). Didesain agar sangat mudah dipahami (non-akuntan).

## 3. Goals & Objectives
- **Business Goals:** Membuat UMKM merasa terbantu tanpa terintimidasi oleh kompleksitas akuntansi tradisional.
- **User Goals:** Melihat omzet hari ini, upload nota pengeluaran, dan mengatur balasan otomatis chatbot.
- **Non-Goals:** Bukan UI untuk kasir (POS) fisik.

## 4. Key Features & Requirements

### Feature 1: Financial Dashboard (Simplified)
- **Description:** Menampilkan visualisasi kesehatan bisnis.
- **Acceptance Criteria:** Menampilkan Kartu Omzet Hari Ini, Pengeluaran Bulan Ini, dan Grafik Arus Kas sederhana (Hijau/Merah).
- **Priority:** P1 (High)

### Feature 2: Receipt OCR Upload
- **Description:** Antarmuka unggah foto struk.
- **Acceptance Criteria:** Pengguna dapat menekan tombol upload gambar dari HP, UI loading memunculkan "Sedang dianalisis AI", lalu form draft jurnal muncul otomatis.
- **Priority:** P1 (High)

### Feature 3: Chatbot Setting & Inbox
- **Description:** Pengaturan asisten AI (WhatsApp/Telegram).
- **Acceptance Criteria:** Form pengaturan "Karakteristik Bot" (Ramah/Formal) dan Inbox untuk melihat obrolan AI dengan pelanggan (Live takeover).
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** React / Next.js, TailwindCSS.
- **Integrations:** Berkomunikasi dengan `apps/umkm` via API Gateway. 

## 6. User Experience & Design
- **Platform:** Mobile-first Web App (PWA). Karena banyak UMKM mengakses via HP.
- **Vibe:** "Terang, Ramah, Mudah, Ceria". Hindari jargon akuntansi ("Debit/Kredit" diganti "Uang Masuk/Keluar").

## 7. Metrics & Analytics
- **Success Metrics:** Jumlah unggahan nota per hari, PWA install rate.

## 8. Release Phases
- **Phase 1 (MVP):** Dashboard Keuangan, Input Manual Uang Masuk/Keluar.
- **Phase 2:** OCR Upload Nota, Setting Bot AI, Live Chat Inbox.
