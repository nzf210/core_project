# WCH Ecosystem Architecture

WCH Platform adalah arsitektur *monorepo* modern yang menggabungkan *backend* Go untuk melayani dua produk aktif: **AI Agent UMKM & Pembukuan** dan **Manajemen Pemenangan Pemilu**. Sistem ini sepenuhnya terintegrasi dengan kecerdasan buatan melalui *AI Gateway* terpusat.

> **Catatan:** Produk SaaS Crypto Trading Bot (`apps/crypto`) telah **DIARSIPKAN** dan tidak aktif dikembangkan.

## Komponen Utama

### 1. Core Apps & Services

**UMKM (Port 8201–8202, 9001):**
- **`umkm/accounting` (Port 8201):** *Double-Entry Bookkeeping Engine* berbasis PostgreSQL. Memvalidasi transaksi dengan prinsip dasar akuntansi debit/kredit dan mencetak laporan Laba/Rugi. Termasuk POS, klinik queue, dan RAG handlers.
- **`umkm/chatbot` (Port 8202):** Layanan pelanggan pintar dengan teknologi *Retrieval-Augmented Generation* (RAG). Menyuntikkan data Laba/Rugi internal ke memori AI (*System Prompt*) saat UMKM bertanya tentang keuangan mereka.
- **`umkm/business` (Port 9001):** Business management API.

**Campaign (Port 9002):**
- **`campaign/api` (Port 9002):** REST API manajemen relawan, real count, dan sentiment analysis.

**Shared Services (Port 8000–8006, 8210):**
- **`api-gateway` (Port 8000):** Reverse proxy dan routing ke semua service.
- **`auth-service` (Port 8001):** JWT, Login, Register, RBAC.
- **`ai-gateway` (Port 8002):** Proksi tunggal LLM dengan semantic cache dan billing.
- **`billing-service` (Port 8003):** Xendit subscription, wallet, voucher.
- **`notification-service` (Port 8005):** Telegram/Email notifikasi.
- **`subscription-worker` (Port 8006):** Hold expired tenants.
- **`wa-gateway` (Port 8202):** WhatsApp via whatsmeow (unofficial).
- **`wa-cloud-api` (Port 8210):** WhatsApp Cloud API (Meta Official).

### 2. `ai-gateway` (Port 8002)
Otak utama LLM *proxying* untuk seluruh platform.
- **Provider:** Mendukung MiniMax M2.7 (otomatis *fallback* ke Gemini 1.5 Flash bila gagal).
- **Optimization:** *Semantic Cache* menggunakan Redis untuk menekan biaya Token dan *Latency* jika terdapat kueri berulang.
- **Billing:** Menyimpan penggunaan Token Input/Output setiap *Tenant* ke `ai_usage_logs` secara _real-time_.

### 3. Event-Driven Background Workers
Sistem mengeksekusi operasi asinkron tanpa mengganggu respons *frontend API*:
- **`umkm/automation`:** Berlangganan saluran `tenant_events` via Redis Pub/Sub. Memproses operasi berat seperti pembuatan Laporan Keuangan Bulanan PDF dan transmisi Email, dibantu AI.

## Teknologi Inti
*   **Bahasa Utama:** Golang 1.25.10
*   **Relational Storage:** PostgreSQL 16 (`pgx/v5`)
*   **In-Memory Storage & Pub/Sub:** Redis
*   **Authentication:** JWT Bearer Token (`golang-jwt/jwt/v5`)
*   **Routing & HTTP:** *Standard Library* `net/http` Go

## Menjalankan Aplikasi Lokal (Quick Start)

Proyek dilengkapi `Makefile` agar semua *microservices* dapat dijalankan bersamaan.

### Menjalankan Semuanya:
```bash
make start-all
```
*Log setiap servis tersimpan di `logs/*.log`. PID tersimpan di `run/*.pid`.*

### Hot-Reload Development:
```bash
make dev-all        # Semua BE (air) + FE (Vite) hot-reload
```

### Mematikan Semuanya:
```bash
make stop-all
```

### Menjalankan Servis Individual:
- `make dev-gateway`    (Port 8000)
- `make dev-auth`       (Port 8001)
- `make dev-accounting` (Port 8201)
- `make dev-chatbot`    (Port 8202)
- `make run-automation` (Background Worker)
