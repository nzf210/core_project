# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** SaaS Crypto Trading Bot (`apps/crypto`)
- **Target Audience:** Pengguna retail (B2C) yang ingin mengotomatisasi perdagangan kripto mereka.
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Platform otomatisasi perdagangan kripto berbasis web yang ditawarkan sebagai Software as a Service (SaaS). Produk ini menyelesaikan masalah bagi trader yang tidak memiliki waktu memantau pasar 24/7 dengan menyediakan bot cerdas (Grid, DCA, Signal) yang mengeksekusi order secara otomatis berdasarkan strategi yang ditentukan. Produk ini merupakan salah satu pilar pendapatan utama WCH Platform.

## 3. Goals & Objectives
- **Business Goals:** Menghasilkan MRR (Monthly Recurring Revenue) melalui model langganan multi-tier.
- **User Goals:** Memberikan kemudahan trading 24/7 dengan risiko terukur tanpa harus memantau layar.
- **Non-Goals:** Tidak menjadi platform exchange sendiri; hanya sebagai layer otomatisasi di atas exchange pihak ketiga (Binance, Tokocrypto, dll).

## 4. Key Features & Requirements

### Feature 1: Exchange Integration & Key Management
- **Description:** Menghubungkan akun pengguna dengan exchange pihak ketiga secara aman.
- **Acceptance Criteria:** Pengguna dapat memasukkan API Key dan Secret Key yang kemudian disimpan dengan enkripsi AES-256. Sistem dapat menarik saldo dan daftar aset menggunakan CCXT.
- **Priority:** P1 (High)

### Feature 2: Strategy Builder & Pre-set Bots
- **Description:** Menyediakan opsi bot seperti Grid Bot (beli rendah, jual tinggi dalam range), DCA Bot, dan Signal Bot.
- **Acceptance Criteria:** Pengguna dapat mengonfigurasi parameter (upper/lower price, grids, investment amount) dan menjalankan/mematikan bot.
- **Priority:** P1 (High)

### Feature 3: Execution Worker
- **Description:** Background worker (Go) untuk memantau ticker harga dan mengeksekusi order asinkron.
- **Acceptance Criteria:** Menggunakan TimescaleDB untuk riwayat harga dan Redis untuk state bot, sanggup menangani ribuan bot paralel tanpa hambatan (menggunakan Goroutines).
- **Priority:** P1 (High)

### Feature 4: Backtesting Engine
- **Description:** Memungkinkan pengguna menguji konfigurasi bot menggunakan data harga historis.
- **Acceptance Criteria:** Menampilkan simulasi profitabilitas sebelum bot diaktifkan dengan uang asli.
- **Priority:** P2 (Medium)

### Feature 5: Portfolio & PnL Analytics
- **Description:** Dashboard untuk melacak performa akun secara real-time.
- **Acceptance Criteria:** Visualisasi Win Rate, PnL, alokasi aset menggunakan chart.
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** `CCXT` (atau wrapper native) untuk API exchange, `billing-service` untuk manajemen tier langganan.
- **Database:** PostgreSQL untuk konfigurasi bot dan user, Redis untuk ticker/cache, TimescaleDB untuk histori harga (OHLCV).
- **Security & Performance:** Enkripsi ketat (AES-256) untuk API Keys pengguna. Worker harus sangat efisien meminimalkan latensi eksekusi (slippage).

## 6. User Experience & Design
- **Platform:** Web Desktop & Mobile Web.
- **Key Flows:** Register -> Beli Paket (Billing) -> Masukkan API Key Exchange -> Setup Bot -> Pantau PnL.

## 7. Metrics & Analytics
- **Success Metrics:** Jumlah Bot Aktif, Total Volume Trading, MRR, User Retention Rate.

## 8. Release Phases
- **Phase 1 (MVP):** Integrasi Binance/Tokocrypto, Grid Bot & DCA Bot Engine, PnL Dasar.
- **Phase 2:** Signal Bot (TradingView Webhook), Backtesting, Integrasi ke AI Gateway untuk rekomendasi strategi.
