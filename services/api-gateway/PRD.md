# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** API Gateway (`services/api-gateway`)
- **Target Audience:** Seluruh request dari client frontend (Web, Mobile).
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
API Gateway adalah gerbang utama yang meneruskan lalu lintas HTTP (REST/WebSocket) dari dunia luar (klien frontend) ke microservices di dalam monorepo WCH Platform. Sistem ini bertugas menangani rute, rate limiting global, SSL termination, dan cross-origin resource sharing (CORS).

## 3. Goals & Objectives
- **Business Goals:** Melindungi backend dari spam/DDoS dengan rate limiting terpusat.
- **User Goals:** Memberikan satu single entrypoint (`api.wch-platform.com`) untuk memudahkan integrasi frontend.
- **Non-Goals:** Tidak menangani logika bisnis aplikasi (seperti hitung PnL atau neraca keuangan).

## 4. Key Features & Requirements

### Feature 1: Request Routing & Reverse Proxy
- **Description:** Meneruskan request dari path tertentu ke service yang tepat.
- **Acceptance Criteria:** Request ke `/api/v1/crypto` dialihkan ke port `crypto-service`, `/api/v1/umkm` ke port `umkm-service`.
- **Priority:** P1 (High)

### Feature 2: Rate Limiting & Throttling
- **Description:** Membatasi jumlah request dari satu IP atau User untuk mencegah abuse.
- **Acceptance Criteria:** Batas request 100 req/menit per IP untuk API publik.
- **Priority:** P1 (High)

### Feature 3: Global Authentication Check (Opsional/Delegasi)
- **Description:** Memverifikasi keberadaan/keabsahan JWT Token sebelum meneruskan request (bekerja sama dengan Auth Service).
- **Acceptance Criteria:** Menolak request dengan 401 Unauthorized jika rute yang dituju adalah rute privat namun tidak menyertakan Bearer Token valid.
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** Redis (untuk rate limiting distributed cache).
- **Architecture:** Biasanya diimplementasikan dengan Nginx, Traefik, Kong, atau Go native reverse proxy.

## 6. User Experience & Design
- **Platform:** Infrastruktur / Gateway (Non-UI).

## 7. Metrics & Analytics
- **Success Metrics:** 99.99% Uptime, Request processing latency < 10ms (overhead minimum).

## 8. Release Phases
- **Phase 1 (MVP):** Routing URL statis ke service masing-masing.
- **Phase 2:** Dinamis routing, Rate Limiting per Tenant, Integrasi monitoring (Prometheus/Grafana).
