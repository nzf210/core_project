# ✅ IMPLEMENTED: Self-Hosted WhatsApp Gateway (Pengganti Fonnte)

> **Status:** ✅ **PRODUCTION READY** — wa-gateway (whatsmeow) sudah berjalan di port 8202.  
> Fonnte third-party **TIDAK LAGI DIGUNAKAN**. Semua pesan WA lewat internal gateway.

## 1. Latar Belakang & Tujuan
Saat ini, modul `apps/umkm/chatbot` menggunakan pihak ketiga (**Fonnte**) untuk mengirim dan menerima pesan WhatsApp. Meskipun mudah untuk tahap MVP, ketika platform UMKM kita berkembang pesat dan memiliki ribuan pengguna (tenant) dengan ratusan ribu pesan per hari, bergantung pada *third-party provider* akan memunculkan masalah:
- **Biaya (Cost):** Biaya per pesan / langganan bulanan per *device* akan membengkak seiring bertambahnya tenant UMKM.
- **Keterbatasan API (Rate Limits):** Pihak ketiga memiliki *rate limit* yang bisa menghambat performa di jam sibuk.
- **Privasi Data:** Seluruh pesan transaksi keuangan pelanggan melewati server pihak ketiga.

**Tujuan:** Membangun *Internal WhatsApp Gateway* mandiri yang bisa di-*host* di server sendiri, memungkinkan setiap tenant UMKM menyambungkan nomor WhatsApp mereka secara gratis (tanpa biaya per pesan), serta memiliki kontrol penuh atas infrastruktur.

**✅ HASIL:** Tujuan tercapai. `services/wa-gateway` (port 8202) sudah production-ready dengan whatsmeow.

## 2. Pilihan Teknologi (Technology Stack)

Karena seluruh arsitektur *backend* WCH Platform dibangun menggunakan **Golang**, pilihan terbaik untuk *engine* WhatsApp adalah:
**`whatsmeow` (go.mau.fi/whatsmeow)**
*   *Deskripsi*: Library native Golang yang mengimplementasikan protokol WhatsApp Web.
*   *Kelebihan*: Sangat ringan, sangat cepat (karena Go Goroutines), dan mendukung *Multi-Device* (bisa melayani ratusan sesi/nomor UMKM dalam satu *instance* server).
*   *Penyimpanan Sesi*: Mendukung penyimpanan sesi login langsung ke PostgreSQL (sejalan dengan stack database utama kita).

*Alternatif lain: Baileys (Node.js/TypeScript) - namun ini akan menambah beban karena harus setup environment Node.js tambahan di luar arsitektur Go yang sudah ada.*

## 3. Arsitektur Sistem (Self-Hosted WA Gateway)

Kita akan membuat microservice baru: `services/wa-gateway`

```mermaid
graph TD
    subgraph WCH Platform
        FE[UMKM Web Dashboard]
        CB[UMKM Chatbot Service]
        WAG[wa-gateway (whatsmeow)]
        DB[(PostgreSQL - WA Sessions)]
        Redis[(Redis Pub/Sub)]
    end

    UserHP[HP Pelanggan UMKM]
    Meta[Server WhatsApp / Meta]

    %% Alur Link Device
    FE -- "1. Request QR Code" --> WAG
    WAG -- "2. Generate & Return QR" --> FE
    WAG -- "3. Save Session" --> DB

    %% Alur Pesan
    UserHP -- "Chat WA" --> Meta
    Meta -- "WebSocket" --> WAG
    WAG -- "Publish Message" --> Redis
    Redis -- "Subscribe" --> CB
    
    CB -- "Generate AI Reply" --> WAG
    WAG -- "Send to WA" --> Meta
    Meta -- "Deliver" --> UserHP
```

## 4. Mekanisme Multi-Tenancy & Isolasi Nomor (Setiap UMKM = Nomor Masing-Masing)

Kunci utama mengapa sistem ini bisa dipakai untuk pola SaaS UMKM adalah karena library `whatsmeow` dikonfigurasi berjalan secara *Multi-Client* di dalam *memory* (RAM). Berikut adalah penjabaran mekanismenya:

1. **Client Pool di Memory:**
   Microservice `wa-gateway` akan memiliki sebuah struktur data di Golang, misalnya `map[string]*whatsmeow.Client`, di mana *key* dari map tersebut adalah `tenant_id` milik UMKM.
2. **Koneksi Independen:**
   Setiap `tenant_id` memiliki *session keys* (seperti `clientId`, `macKey`, `identityKey`) yang tersimpan secara independen di PostgreSQL (misal tabel `wa_sessions` memuat kolom `tenant_id` dan `session_data`). 
   Saat `wa-gateway` *startup*, ia akan me-looping seluruh `tenant_id` yang aktif, membuat instansi `*whatsmeow.Client` untuk masing-masing tenant, dan mengoneksikannya ke server WhatsApp (Meta) secara paralel menggunakan *goroutines*.
3. **Mencegah Pesan Tertukar (Cross-talk Prevention):**
   *   **Pesan Masuk:** Saat event `Message` terpicu (pesan datang dari pelanggan), *handler* dari `whatsmeow` otomatis mengenali `*whatsmeow.Client` mana yang sedang menerima event. Sistem akan menarik `tenant_id` yang terikat dengan *client* tersebut, dan mem-publish payload seperti: `{"tenant_id": "T-123", "from": "628xxx", "message": "Halo, menu hari ini?"}` ke Redis. Dengan demikian, `apps/umkm/chatbot` tahu pasti toko mana yang sedang dichat.
   *   **Pesan Keluar:** Saat `chatbot` AI memutuskan untuk membalas, ia menembak API `POST /api/wa/send` ke `wa-gateway` dengan menyertakan `tenant_id`. `wa-gateway` akan mencari `tenant_id` di dalam *map memory*-nya. Jika ketemu, maka pengiriman pesan hanya akan dieksekusi melalui koneksi/nomor WhatsApp milik `tenant_id` tersebut. Jika tenant A nge-chat, mustahil nomor WhatsApp tenant B yang membalas.

## 5. Fitur & Alur Kerja (Workflows)

### A. Alur Pendaftaran Nomor Baru (Device Linking)
1. UMKM masuk ke `frontend/umkm-web` dan ke menu "Hubungkan WhatsApp".
2. Frontend menembak API `GET /api/wa/qr?tenant_id=123` ke `wa-gateway`.
3. `wa-gateway` membuat instansi client baru dengan `whatsmeow`, menghasilkan QR Code, dan mengembalikannya ke layar.
4. UMKM melakukan *scan* menggunakan HP mereka.
5. `wa-gateway` menyimpan *auth session* (keys) ke dalam PostgreSQL (tabel `wa_sessions`).
6. Kapanpun server *restart*, `wa-gateway` cukup memuat *session* dari database tanpa perlu *scan* QR ulang.

### B. Alur Pesan Masuk (Inbound Message)
1. `wa-gateway` secara asinkron mendengarkan *event* `Message` dari semua client yang terkoneksi.
2. Ketika pesan masuk, `wa-gateway` mem-parsing teks dan nomor pengirim, lalu meneruskannya (via REST HTTP Call atau Redis Pub/Sub) ke `apps/umkm/chatbot`.
3. `chatbot` merespons dengan logika RAG/AI.

### C. Alur Pesan Keluar (Outbound Message)
1. `apps/umkm/chatbot` mengirim perintah kirim pesan ke `wa-gateway` via internal API: `POST /api/wa/send` (Body: `tenant_id`, `to_number`, `message`).
2. `wa-gateway` mencari *client/session* berdasarkan `tenant_id`, lalu mengeksekusi pengiriman pesan ke server Meta.

## 6. Strategi Migrasi (Migration Phases)

Untuk menghindari gangguan pada pengguna *existing*, migrasi dilakukan bertahap:

*   **Fase 1: Development `wa-gateway` & Internal Testing (Bulan 1)**
    *   Setup repo `services/wa-gateway`.
    *   Implementasi `whatsmeow` dengan PostgreSQL store.
    *   Membuat API endpoint internal (Send Message, Get QR, Logout).
*   **Fase 2: Hybrid Mode / A/B Testing (Bulan 2)**
    *   Membuat *interface* `WhatsAppProvider` di `apps/umkm/chatbot` (yang bisa *switch* antara Fonnte atau Internal Gateway).
    *   Membuka fitur Beta bagi 10 UMKM pertama untuk mencoba Internal Gateway (Scan QR di sistem kita).
*   **Fase 3: Full Cut-off (Bulan 3)**
    *   Meminta seluruh UMKM yang masih menggunakan Fonnte untuk migrasi melakukan *scan* QR di Dashboard baru.
    *   Setelah semua pindah, langganan Fonnte dihentikan.
    *   Menghapus variabel `FONNTE_TOKEN` dari `.env` dan kode lama.

## 7. Pertimbangan Infrastruktur & Keamanan
*   **RAM & CPU:** Setiap sesi `whatsmeow` cukup ringan, namun jika terdapat 1.000 tenant UMKM online bersamaan, Gateway ini membutuhkan RAM sekitar 2-4GB khusus untuk *service*-nya.
*   **Auto-Reconnect:** Sistem harus memiliki mekanisme *re-connect* otomatis jika koneksi WebSocket ke Meta terputus.
*   **Rate Limiting Internal:** Meskipun gratis, `wa-gateway` harus memiliki rate limiter agar AI Chatbot tidak melakukan "spam" ke nomor pelanggan (mencegah nomor UMKM diblokir/banned oleh WhatsApp akibat perilaku mencurigakan).
*   **Anti-Ban Guidelines:** UMKM harus diedukasi agar tidak mengirim *blast message* promosi secara brutal menggunakan nomor biasa. *Blast message* massal tetap disarankan menggunakan WhatsApp Business API Resmi. Mode QR Code (*whatsmeow*) difokuskan untuk membalas obrolan interaktif (CS/Chatbot).
