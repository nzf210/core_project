# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** Crypto Web App (`frontend/crypto-web`)
- **Target Audience:** Pengguna Crypto Trading Bot (Trader Retail).
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Antarmuka pengguna (Frontend) untuk aplikasi Crypto Trading Bot. Ini adalah SPA (Single Page Application) yang menyajikan dashboard analitik portofolio kripto pengguna secara real-time dan interface untuk mengonfigurasi bot.

## 3. Goals & Objectives
- **Business Goals:** Menyediakan UI yang premium dan modern (dark mode, glassmorphism) agar trader merasa sedang memakai institusional-grade platform.
- **User Goals:** Dapat memonitor PnL, open orders, dan setelan bot dengan sangat responsif.
- **Non-Goals:** Bukan untuk mengembangkan backend engine trading (semua dilayani via API ke `apps/crypto`).

## 4. Key Features & Requirements

### Feature 1: Portfolio & PnL Dashboard
- **Description:** Ringkasan aset dan keuntungan.
- **Acceptance Criteria:** Menampilkan grafik garis untuk histori PnL (Profit and Loss), Donut chart untuk alokasi aset.
- **Priority:** P1 (High)

### Feature 2: Bot Configuration Interface
- **Description:** Form untuk membuat bot Grid / DCA.
- **Acceptance Criteria:** Terdapat input parameter (Upper/Lower limit, jumlah grid) dengan validasi form real-time.
- **Priority:** P1 (High)

### Feature 3: Real-Time Price Ticker
- **Description:** Streaming harga pasar via WebSocket.
- **Acceptance Criteria:** Harga pasangan kripto berkedip hijau/merah saat berubah, UI chart TradingView.
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** React / Next.js, TailwindCSS (Dark Theme preferred), Lightweight Charts (TradingView).
- **Integrations:** Mengkonsumsi API dari `apps/crypto` (via API Gateway). WebSocket connection untuk order updates.

## 6. User Experience & Design
- **Platform:** Web Browser (Desktop disarankan untuk trading).
- **Vibe:** "Premium, Dark, Cybernetic, Cepat".

## 7. Metrics & Analytics
- **Success Metrics:** Time on page, Feature engagement (Klik buat bot).

## 8. Release Phases
- **Phase 1 (MVP):** Dashboard PnL Statis, Form Buat Bot, Koneksi API.
- **Phase 2:** WebSocket realtime updates, TradingView Advanced Chart integration.
