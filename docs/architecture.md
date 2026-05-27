# WCH Ecosystem Architecture

WCH (Web3, Crypto, HTTP) adalah arsitektur *monorepo* modern yang menggabungkan *backend* Go untuk melayani kebutuhan Enterprise (UMKM) dan *Quantitative Trading* (Crypto). Sistem ini sepenuhnya terintegrasi dengan kecerdasan buatan melalui *AI Gateway* terpusat.

## Komponen Utama

### 1. `apps-gateway` (Port 9000-9003)
Merupakan kumpulan antarmuka REST API dan layanan utama yang menghadap pengguna:
- **`umkm/accounting` (Port 9001):** *Double-Entry Bookkeeping Engine* berbasis PostgreSQL. Memvalidasi transaksi dengan prinsip dasar akuntansi debit/kredit dan mencetak laporan Laba/Rugi.
- **`umkm/chatbot` (Port 9002):** Layanan pelanggan pintar dengan teknologi *Retrieval-Augmented Generation* (RAG). Mampu memotong kompas menyuntikkan data Laba/Rugi internal ke memori AI (*System Prompt*) saat UMKM bertanya tentang keuangan mereka.

### 2. `ai-gateway` (Port 8003)
Otak utama LLM *proxying* untuk seluruh platform.
- **Provider:** Mendukung MiniMax M2.7 (Otomatis *Fallback* ke Gemini 1.5 Flash bila gagal/MT).
- **Optimization:** Melakukan penyanggaan *Semantic Cache* menggunakan Redis untuk menekan biaya Token dan _Latency_ jika terdapat kueri berulang.
- **Billing:** Menyimpan penggunaan Token Input/Output setiap *Tenant* ke `ai_usage_logs` secara _real-time_.

### 3. Event-Driven Background Workers
Sistem mengeksekusi operasi asinkron dalam skala besar tanpa mengganggu respons *frontend API*:
- **`umkm/automation`:** Berlangganan (Subscribe) saluran `tenant_events` via Redis Pub/Sub. Memproses secara tertutup operasi berbobot berat seperti pembuatan Laporan Keuangan Bulanan bentuk PDF dan transmisi Email, dibantu generator oleh AI.
- **`crypto/worker`:** Bot algoritmik simulasi *trading* yang secara simultan menarik harga (contoh: BTCUSDT dari API Publik Binance), menyerahkannya kepada asisten pintar Kripto internal (*AI Oracle*), lalu memancarkan keputusannya ("BELI" / "JUAL" / "TAHAN").

## Teknologi Inti
*   **Bahasa Utama:** Golang 1.25.10
*   **Relational Storage:** PostgreSQL 16 (`pgx/v5`)
*   **In-Memory Storage & Pub/Sub:** Redis
*   **Authentication:** JWT Bearer Token (`golang-jwt/jwt/v5`)
*   **Routing & HTTP:** *Standard Library* `net/http` Go

## Menjalankan Aplikasi Lokal (Quick Start)

Untuk mempermudah proses pengembangan (Development) dan pengujian (Testing), proyek ini telah dilengkapi dengan `Makefile` agar semua *microservices* dapat dijalankan bersamaan.

### Menjalankan Semuanya:
```bash
make start-all
```
*Perintah ini menjalankan seluruh servis di background. Log setiap servis bisa dipantau pada file bereksistensi `.log` di root folder (misal `ai.log`, `crypto.log`).*

### Mematikan Semuanya:
```bash
make stop-all
```

### Menjalankan Servis Individual:
- `make run-auth` (Port 8000)
- `make run-ai` (Port 8003)
- `make run-accounting` (Port 9001)
- `make run-chatbot` (Port 9002)
- `make run-automation` (Background Worker)
- `make run-crypto` (Background Worker)
