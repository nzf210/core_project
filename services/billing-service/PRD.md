# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** Billing & Subscription Service (`services/billing-service`)
- **Target Audience:** Pelanggan UMKM, Trader Kripto, dan Tim Kampanye (B2C & B2B).
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Layanan monetisasi global untuk memfasilitasi integrasi pembayaran dan mengelola siklus langganan SaaS (Subscription Lifecycle). Servis ini memungkinkan WCH Platform memotong biaya langganan bulanan atau berdasarkan pemakaian (pay-as-you-go).

## 3. Goals & Objectives
- **Business Goals:** Memastikan aliran kas masuk (cashflow) lancar, otomatisasi tagihan, dan pencegahan churn.
- **User Goals:** Pengalaman checkout yang aman, transparan melihat riwayat tagihan, dan mudah upgrade/downgrade paket.
- **Non-Goals:** Bukan untuk mengelola pembayaran dari pelanggan-ke-UMKM, melainkan dari Tenant-ke-WCH Platform.

## 4. Key Features & Requirements

### Feature 1: Payment Gateway Integration
- **Description:** Integrasi dengan Xendit (lokal) dan Stripe (global).
- **Acceptance Criteria:** Dapat memproses metode pembayaran Virtual Account, QRIS, Kartu Kredit, dan E-Wallet.
- **Priority:** P1 (High)

### Feature 2: Subscription & Tier Management
- **Description:** Menangani siklus paket bulanan/tahunan (Basic, Pro, Enterprise).
- **Acceptance Criteria:** Sistem secara otomatis mendowngrade atau memblokir akses pengguna jika tagihan langganan gagal diperpanjang (Auto-Renewal).
- **Priority:** P1 (High)

### Feature 3: Usage-Based Billing (Pay-as-you-go)
- **Description:** Penagihan tambahan berdasarkan konsumsi token LLM berlebih (di luar kuota paket).
- **Acceptance Criteria:** Sinkronisasi konsumsi dari `ai-gateway` ke keranjang tagihan bulanan pelanggan.
- **Priority:** P2 (Medium)

### Feature 4: Invoicing & Webhook
- **Description:** Mencetak invoice dan menerima konfirmasi pembayaran real-time.
- **Acceptance Criteria:** Menghasilkan PDF invoice yang dikirim otomatis via Email.
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** API Payment Gateway (Stripe/Xendit), `notification-service` untuk pengiriman Invoice.
- **Database:** PostgreSQL (pencatatan transaksi/invoice yang kuat/ACID compliant).
- **Security & Performance:** Webhook dari payment gateway HARUS divalidasi secret signature-nya untuk mencegah modifikasi status pembayaran fiktif.

## 6. Metrics & Analytics
- **Success Metrics:** Payment Success Rate, MRR Analytics (Net Revenue Retention), Webhook Processing Latency.

## 7. Release Phases
- **Phase 1 (MVP):** Integrasi QRIS & VA Xendit, Paket Langganan Manual, Webhook Handler.
- **Phase 2:** Stripe Subscription, Usage-based Billing, Auto-renewal Credit Card.
