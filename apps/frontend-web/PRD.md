# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** Main Web & Landing Pages (`apps/frontend-web`)
- **Target Audience:** Pengunjung publik (Public Audience) dan Tenant/Global Admin.
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Repositori `apps/frontend-web` difungsikan sebagai rumah untuk Landing Page WCH Platform (Marketing site) sekaligus Global Admin Dashboard. Di sinilah calon pelanggan mendaftar untuk pertama kalinya sebelum dialihkan ke produk spesifik (Crypto, UMKM, atau Campaign).

## 3. Goals & Objectives
- **Business Goals:** Menarik prospek (lead generation) dan mengonversinya menjadi pengguna trial/berbayar WCH Platform.
- **User Goals:** Memahami apa itu WCH Platform, membandingkan harga paket langganan (Pricing), dan mengelola profil akun (My Account).
- **Non-Goals:** Bukan tempat untuk mengelola fungsi spesifik (seperti trading atau input nota akuntansi).

## 4. Key Features & Requirements

### Feature 1: Landing Page & Pricing
- **Description:** Halaman marketing yang memamerkan tiga lini produk (Crypto, UMKM, Campaign).
- **Acceptance Criteria:** Terdapat halaman Pricing, Features, Testimonial, dan tombol "Start Free Trial".
- **Priority:** P1 (High)

### Feature 2: Unified Login & Registration Portal
- **Description:** Gerbang masuk utama untuk seluruh platform.
- **Acceptance Criteria:** Terintegrasi dengan Auth Service (Google OAuth, Email/Password), lalu mengarahkan ke sub-domain atau halaman aplikasi yang tepat.
- **Priority:** P1 (High)

### Feature 3: Global Admin / Tenant Management Dashboard
- **Description:** (Untuk Super Admin) Mengelola tenant, kuota, dan langganan secara terpusat.
- **Acceptance Criteria:** Admin dapat men-suspend tenant, melihat total tagihan seluruh produk, dan mengatur permission (bekerja sama dengan `billing-service` & `tenant-service`).
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** Next.js (SSG/SSR untuk SEO Landing Page), Tailwind CSS.
- **Integrations:** `auth-service` dan `billing-service`.

## 6. User Experience & Design
- **Platform:** Responsive Web (Desktop & Mobile).
- **Vibe:** "Modern, Clean, Trustworthy". Harus secepat kilat (Optimized LCP & Core Web Vitals) untuk SEO.

## 7. Metrics & Analytics
- **Success Metrics:** Conversion Rate (Visitor to Registered User), Bounce Rate, SEO Ranking.

## 8. Release Phases
- **Phase 1 (MVP):** Landing page statis, Login Page, Halaman Pricing.
- **Phase 2:** Tenant Admin Dashboard untuk mengelola langganan (Billing Portal).
