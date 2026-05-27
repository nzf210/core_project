# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** Auth Service (`services/auth-service`)
- **Target Audience:** Pengguna sistem (UMKM, Caleg, Trader Kripto, Admin) dan internal services.
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
Auth Service adalah layanan terpusat untuk Manajemen Identitas dan Akses (IAM) di seluruh ekosistem WCH Platform. Menangani registrasi, login, penerbitan token JWT, Single Sign-On (SSO), serta sinkronisasi hak akses berbasis peran (Role-Based Access Control).

## 3. Goals & Objectives
- **Business Goals:** Menyediakan pengalaman login tanpa batas (seamless) di seluruh lini produk WCH (Satu Akun untuk Semua).
- **User Goals:** Dapat mendaftar, mengamankan akun dengan MFA, dan masuk (login) dengan aman dan cepat (Google, Email/Password).
- **Non-Goals:** Tidak mengelola profil bisnis spesifik produk (misal: setting margin kripto) — hanya mengelola keamanan identitas inti.

## 4. Key Features & Requirements

### Feature 1: JWT Access & Refresh Token Management
- **Description:** Mekanisme login standar.
- **Acceptance Criteria:** Mengeluarkan short-lived Access Token (15m) dan long-lived Refresh Token (7d).
- **Priority:** P1 (High)

### Feature 2: OAuth2 Social Login
- **Description:** Login menggunakan akun pihak ketiga.
- **Acceptance Criteria:** Pengguna dapat login melalui "Continue with Google".
- **Priority:** P2 (Medium)

### Feature 3: Role-Based Access Control (RBAC) & Tenant Awareness
- **Description:** Otorisasi spesifik produk (misal: Admin UMKM vs Kasir UMKM).
- **Acceptance Criteria:** Token JWT memuat `role` dan `tenant_id` untuk mempercepat validasi di sisi aplikasi.
- **Priority:** P1 (High)

### Feature 4: Multi-Factor Authentication (MFA)
- **Description:** Pengamanan lapis kedua (khususnya untuk Crypto App).
- **Acceptance Criteria:** Integrasi dengan Google Authenticator (TOTP) atau OTP Email.
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** PostgreSQL (tabel `users`, `roles`), Redis (menyimpan token ter-revoke/blacklist token saat logout).
- **Security & Performance:** Password di-hash dengan Bcrypt/Argon2. Keamanan terhadap serangan Brute-Force melalui rate limiting di level Nginx/API Gateway.

## 6. Metrics & Analytics
- **Success Metrics:** Kecepatan token verification (<5ms), Zero security breach.

## 7. Release Phases
- **Phase 1 (MVP):** Email/Password Login, JWT Generation, RBAC Dasar.
- **Phase 2:** OAuth Google Login, MFA dengan TOTP, Integrasi Notifikasi (Email OTP).
