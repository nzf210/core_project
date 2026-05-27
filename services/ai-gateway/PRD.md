# Product Requirements Document (PRD)

## 1. Meta Information
- **Project Name:** AI Gateway Service (`services/ai-gateway`)
- **Target Audience:** Internal microservices (UMKM Chatbot, Campaign OCR, Crypto Oracle).
- **Status:** Draft
- **Last Updated:** 2026-05-25

## 2. Product Overview
AI Gateway bertindak sebagai proksi terpusat (centralized proxy) untuk semua interaksi dengan Large Language Models (LLM) seperti OpenAI GPT, Gemini, Claude, dan Llama (MiniMax). Servis ini mengontrol billing, rate limiting, semantic caching, dan fallback provider sehingga setiap microservice tidak perlu mengimplementasikan logika koneksi LLM masing-masing.

## 3. Goals & Objectives
- **Business Goals:** Menghemat biaya konsumsi API LLM melalui semantic caching dan routing cerdas.
- **User Goals:** Memberikan respons AI yang cepat dan reliabel bagi pengguna akhir produk.
- **Non-Goals:** Bukan untuk melatih LLM (fine-tuning) sendiri dari nol.

## 4. Key Features & Requirements

### Feature 1: Unified LLM API Proxy
- **Description:** Satu endpoint standar untuk memanggil berbagai provider LLM.
- **Acceptance Criteria:** Developer aplikasi bisa mengganti provider dari OpenAI ke Gemini tanpa mengubah kode aplikasinya sendiri (diatur di Gateway).
- **Priority:** P1 (High)

### Feature 2: Semantic Caching
- **Description:** Menyimpan prompt dan respons yang maknanya sama di Redis.
- **Acceptance Criteria:** Jika pertanyaan "Bagaimana cara buat bot?" dan "Gimana cara bikin bot?" memiliki makna sama, kembalikan respons dari Redis (cache hit) daripada menagih API LLM.
- **Priority:** P1 (High)

### Feature 3: Provider Fallback & High Availability
- **Description:** Perpindahan provider otomatis jika terjadi timeout atau error.
- **Acceptance Criteria:** Jika MiniMax M2.7 mengalami downtime, otomatis request dilempar ke Gemini 1.5 Flash tanpa memutus layanan (*graceful fallback*).
- **Priority:** P2 (Medium)

### Feature 4: Token Usage & Billing Logger
- **Description:** Melacak penggunaan token LLM per Tenant.
- **Acceptance Criteria:** Menyimpan metrik Input/Output Token ke `ai_usage_logs` secara real-time untuk kebutuhan *pay-as-you-go billing*.
- **Priority:** P2 (Medium)

## 5. Technical Considerations (Architecture & Integrations)
- **Dependencies:** SDK LLM dari berbagai provider (OpenAI, Gemini), Redis untuk cache.
- **Database:** PostgreSQL untuk menyimpan log konsumsi Token per tenant.
- **Security & Performance:** Isolasi API Key LLM yang ketat di env variables (tidak boleh bocor ke apps). Harus sangat asinkron agar tidak memblokir HTTP request.

## 6. User Experience & Design
- **Platform:** Backend Service (REST / gRPC).

## 7. Metrics & Analytics
- **Success Metrics:** Cache Hit Ratio (Target: >30%), 99.9% Uptime, Total Cost Savings dari caching.

## 8. Release Phases
- **Phase 1 (MVP):** Proxy standar untuk Gemini & Llama, basic token logging.
- **Phase 2:** Semantic caching dengan Redis Vector (atau library sejenis), fallback routing otomatis.
