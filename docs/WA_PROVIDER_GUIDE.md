# WhatsApp Provider Guide

> Panduan developer untuk mengoperasikan dan menggunakan WhatsApp provider di WCH Platform.
> Fokus utama: **Self-hosted wa-gateway (whatsmeow)** —上手 sebelum subscribe ke Meta Cloud API.

---

## Ringkasan: Dua WA Provider

WCH Platform memiliki dua WA provider yang bekerja bersama:

| Provider | Library | Port | Use Case | Biaya |
|:---------|:--------|:-----|:---------|:------|
| **wa-gateway** | whatsmeow (Web-based) | 8202 | Chatbot interaktif, broadcast, voucher | Tanpa biaya per pesan (sesuai Meta Cloud API) |
| **wa-cloud-api** | Meta Graph API (Official) | 8210 | OTP, invoice, notifikasi transaksional | Per-pesan |

**Alur routing default** (`services/wa-gateway/main.go`):

```
X-Message-Type: otp          → wa-cloud-api (auth-service OTP)
X-Message-Type: invoice      → wa-cloud-api (billing-service invoice)
X-Message-Type: payment       → wa-cloud-api (billing-service payment)
X-Message-Type: subscription → wa-cloud-api (accounting revenue digest)
X-Message-Type: system        → wa-cloud-api (notification-service alert)
(tanpa header)                → whatsmeow (chatbot interaktif)
```

wa-gateway otomatis **fallback ke whatsmeow** jika wa-cloud-api gagal.

---

## 1. wa-gateway (whatsmeow) — Self-Hosted

### 1.1 Arsitektur

```
┌─────────────────────────────────────────────────────────┐
│                    wa-gateway (port 8202)               │
│                                                          │
│  clientMap: map[tenant_id]*whatsmeow.Client             │
│                                                          │
│  ┌──────────────┐    ┌──────────────┐                  │
│  │ tenant_a     │    │ tenant_b     │                  │
│  │ @628123...   │    │ @628456...   │  (1 client/tenant)│
│  └──────────────┘    └──────────────┘                  │
│         ↓                    ↓                           │
│  ┌──────────────────────────────────┐                  │
│  │  PostgreSQL (whatsmeow sqlstore)  │  ← Device keys  │
│  │  wa_tenant_sessions (tenant→jid)   │  ← Our mapping  │
│  └──────────────────────────────────┘                  │
│                                                          │
│  ┌──────────────────────────────────┐                  │
│  │  Redis DB 9 (distributed lock)   │  ← Multi-instance│
│  │  wa:lock:{tenant} / wa:owner:{t}  │                  │
│  └──────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────┘
         ↓ WebSocket (WhatsApp Web protocol)
    Meta / WhatsApp Servers
```

### 1.2 Memulai dari Nol (Fresh Setup)

**Prasyarat:**
- PostgreSQL accessible (tabel `wa_tenant_sessions` dibuat otomatis)
- Redis accessible (untuk distributed lock — opsional tapi sangat direkomendasikan)
- Nomor HP yang belum terhubung ke WhatsApp Web lain (atau siap disconnect dulu)

**Langkah 1: Jalankan wa-gateway**

```bash
# Atau dengan Makefile
make run-wagateway

# Atau langsung
cd services/wa-gateway && go run main.go
```

**Langkah 2: Generate QR Code**

Dari browser atau frontend, buka:

```
GET http://localhost:8202/api/wa/qr?tenant_id=<TENANT_UUID>
```

Response:
```json
{
  "qr": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",  // QR code base64 PNG
  "tenant_id": "..."
}
```

**Langkah 3: Scan QR dengan HP**

1. Buka WhatsApp di HP
2. iOS: Settings → Linked Devices → Link a Device
3. Android: ⋮ (三点) → Linked Devices → Link a Device
4. Scan QR yang muncul di dashboard

**Langkah 4: Verifikasi Koneksi**

```
GET http://localhost:8202/api/wa/status?tenant_id=<TENANT_UUID>
```

Response saat terhubung:
```json
{
  "connected": true,
  "jid": "6281234567890@s.whatsapp.net",
  "device_name": "WA Gateway"
}
```

Response saat belum terhubung:
```json
{
  "connected": false,
  "jid": "",
  "device_name": ""
}
```

**Langkah 5: Kirim Pesan Test**

```
POST http://localhost:8202/api/wa/send
Content-Type: application/x-www-form-urlencoded

tenant_id=<TENANT_UUID>&target=6281234567890&message=Hello dari WA Gateway!
```

Response sukses:
```json
{"success": true, "message_id": "wamid.xxx"}
```

---

### 1.3 API Reference (wa-gateway)

#### `GET /api/wa/qr?tenant_id=<id>`

Generate QR code untuk linking perangkat baru.

| Parameter | Lokasi | Wajib | Deskripsi |
|:----------|:-------|:------|:----------|
| `tenant_id` | query | Ya | UUID tenant |

**Response:**
```json
{
  "qr": "<base64 PNG>",
  "expires_in": 60
}
```

QR code expired dalam 60 detik. Refresh dengan memukul endpoint lagi.

**Catatan:** Jika tenant sudah memiliki sesi tersimpan (dari scan sebelumnya), endpoint ini akan **mengembalikan sesi yang ada** — tidak perlu scan ulang. QR hanya di-generate untuk sesi baru.

---

#### `GET /api/wa/status?tenant_id=<id>`

Cek status koneksi WhatsApp untuk satu tenant.

**Response (terhubung):**
```json
{
  "connected": true,
  "jid": "6281234567890@s.whatsapp.net",
  "device_name": "iPhone 15 Pro"
}
```

**Response (terputus):**
```json
{
  "connected": false,
  "jid": "",
  "device_name": ""
}
```

---

#### `POST /api/wa/send`

Kirim pesan WhatsApp keluar.

**Request (form-urlencoded atau JSON):**
```
tenant_id=<id>&target=<phone>&message=<text>
```

Atau JSON:
```json
POST /api/wa/send
Content-Type: application/json
{
  "tenant_id": "uuid",
  "target": "6281234567890",
  "message": "Pesan ini"
}
```

| Parameter | Wajib | Deskripsi |
|:----------|:------|:----------|
| `tenant_id` | Ya | UUID tenant pengirim |
| `target` | Ya | Nomor tujuan (format: 628xxxxxxxxx, tanpa +) |
| `message` | Ya | Isi pesan (max ~4096 chars) |

**Response sukses:**
```json
{"success": true, "message_id": "wamid.HBgLMTI4..."}
```

**Response rate limited (429):**
```json
{"error": "Rate limit exceeded. Max 5 messages/minute."}
```

**Response tidak terhubung:**
```json
{"error": "Client not connected for tenant xxx"}
```

---

#### `POST /api/wa/logout?tenant_id=<id>`

Putuskan koneksi dan hapus sesi. Tenant harus scan QR ulang untuk reconnect.

---

#### `GET /health`

Health check dengan detail koneksi.

```json
{
  "status": "ok",
  "instance_id": "hostname-pid",
  "connected_sessions": 3,
  "redis": "connected",
  "database": "connected"
}
```

---

#### `GET /metrics`

Metrik Prometheus untuk monitoring.

```
# HELP wa_gateway_connected_sessions Current connected WhatsApp sessions
# TYPE wa_gateway_connected_sessions gauge
wa_gateway_connected_sessions{instance="hostname-pid"} 3

# HELP wa_gateway_messages_sent_total Total messages sent
# TYPE wa_gateway_messages_sent_total counter
wa_gateway_messages_sent_total{instance="hostname-pid"} 1247
```

---

### 1.4 Tenant ID Resolution (Multi-Source)

wa-gateway mencari `tenant_id` dari request dengan urutan:

1. **URL query:** `?tenant_id=X`
2. **Form value:** `tenant_id=X`
3. **HTTP header:** `X-Tenant-ID: X`
4. **JSON body:** `{"tenant_id": "X"}`

Ini memudahkan integrasi dari berbagai sumber (frontend, chatbot, atau service lain).

---

### 1.5 Rate Limiting

Setiap tenant dibatasi **5 pesan per menit** via token bucket algorithm.

```
Tenant A: ████░ (4/5 tokens) → allowed
Tenant B: █████ (5/5 tokens) → blocked (HTTP 429)
```

Token refill setiap menit. Tidak ada burst — 1 pesan/menit adalah rate dasar.

**Mengapa 5/menit?** WhatsApp memblokir nomor yang mengirim terlalu cepat. 5/menit cukup untuk chatbot interaktif normal. Untuk blast promosi massal, gunakan Meta Cloud API.

---

### 1.6 Reconnect & Resilience

Jika koneksi WhatsApp terputus (internet mati, WhatsApp update):

1. **Auto-reconnect** dengan exponential backoff: `30s → 60s → 120s → 240s → 480s → 960s`
2. Maksimum **1 reconnect attempt per 5 menit** per tenant
3. Maksimum **5 attempt** sebelum berhenti (manual intervention required)
4. Session data **tetap tersimpan** di PostgreSQL — tidak perlu scan QR lagi setelah koneksi pulih

Cek status reconnect:
```bash
# Lihat log reconnect
tail -f logs/wa-gateway.log | grep -i reconnect
```

---

### 1.7 Multi-Instance Deployment

Jika menjalankan **beberapa instance wa-gateway** (scale out):

```
Instance A (hostname-pid1)     Instance B (hostname-pid2)
       ↓                               ↓
   ┌─────────────────────────────────────────────┐
   │  Redis DB 9 (distributed lock)              │
   │  wa:lock:{tenant}  = "hostname-pid1"       │
   │  wa:owner:{tenant} = "hostname-pid1"       │
   └─────────────────────────────────────────────┘
```

- Instance yang **mendapat lock** yang mengelola sesi tersebut
- Instance lain **memblokir** operasi untuk tenant itu
- Heartbeat setiap 30 detik menjaga lock tetap hidup
- Jika instance mati → lock expire otomatis setelah 5 menit, instance lain ambil alih

**Startup race prevention:** Saat startup, ada jitter 1-4 detik sebelum restore sesi — ini mencegah semua instance bersamaan menarik sesi dari DB replica yang belum sync.

---

## 2. wa-cloud-api (Meta Cloud API) — Official

### 2.1 Kapan Menggunakan

| Gunakan | Jangan Gunakan |
|:--------|:---------------|
| OTP login/registrasi | Chatbot interaktif (banyak pesan/menit) |
| Invoice notifikasi | Blast promosi massal (wajib template approval Meta) |
| Payment confirmation | Pesan personal one-on-one |
| Subscription alerts | CS/chatbot (bisa kena ban jika abused) |

### 2.2 Setup Credential (Superadmin)

Superadmin menambah credential via dashboard atau API:

```
POST http://localhost:8210/admin/credentials
Content-Type: application/json
{
  "tenant_id": "uuid",
  "phone_number_id": "xxx",
  "waba_id": "yyy",
  "access_token": "EAAC...",
  "verify_token": "my-secret-token"
}
```

**Cara mendapatkan credential dari Meta:**

1. Buat [Meta Business Account](https://business.facebook.com)
2. Verifikasi nomor WA Business
3. Buat WhatsApp Business App di [Meta for Developers](https://developers.facebook.com)
4. Add WhatsApp Product → pilih nomor telepon bisnis
5. Copy **Phone Number ID** dan **WhatsApp Business Account ID**
6. Generate **Permanent Access Token** (Graph API Explorer → User Access Token → permissions: `whatsapp_business_management`, `whatsapp_messaging`)
7. Set **Verify Token** (string acak untuk webhook verification)

### 2.3 Webhook Setup

```bash
# Set webhook URL ke wa-cloud-api
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://your-domain.com/webhooks/wa-cloud/webhook"
```

---

## 3. Chatbot Integration

### 3.1 Inbound (Pesan Masuk dari Pelanggan)

```
Pelanggan chat HP → WhatsApp Server → wa-gateway (whatsmeow)
                                            ↓
                                    eventHandler()
                                            ↓
                                    POST umkm-chatbot:8202/webhook/wa
                                            ↓
                                    Redis queue (chatbot:queue)
                                            ↓
                                    Worker pool (100 goroutines)
                                            ↓
                                    AI Processing (RAG + LLM)
                                            ↓
                                    POST wa-gateway:8202/api/wa/send
                                            ↓
                                    WhatsApp Server → HP Pelanggan
```

### 3.2 Outbound (Balasan Chatbot)

```go
// Di apps/umkm/chatbot/main.go
func sendWA(tenantID, target, message string) error {
    resp, err := http.Post(
        "http://wa-gateway:8202/api/wa/send",
        "application/x-www-form-urlencoded",
        strings.NewReader(fmt.Sprintf("tenant_id=%s&target=%s&message=%s",
            tenantID, target, message)),
    )
    // ...
}
```

---

## 4. Anti-Ban Guidelines

> ⚠️ **KRITIS:** WhatsApp bisa memblokir nomor yang berperilaku mencurigakan. self-hosted wa-gateway (whatsmeow) memiliki **risiko ban lebih tinggi** daripada Meta Cloud API Official.

### 4.1 Yang Bikin Nomor Diblokir

| Perilaku | Risiko | Solusi |
|:---------|:-------|:-------|
| Kirim pesan ke nomor yang belum menyimpan kontak kita | 🔴 Tinggi | Pastikan pelanggan sudah chat duluan |
| Kirim pesan massal dalam waktu singkat | 🔴 Tinggi | Gunakan rate limit 5/menit, atau Meta Cloud API |
| Kirim pesan yang mengandung link mencurigakan | 🟡 Sedang | Hindari bit.ly, t.co, atau link pendek lainnya |
| Kirim pesan yang sama berulang kali | 🟡 Sedang | Gunakan template berbeda untuk broadcast |
| Sering logout/login ulang | 🟡 Sedang | Biarkan sesi tetap aktif terus-menerus |
| Login dari banyak IP berbeda | 🟡 Sedang | Gunakan server dengan IP tetap |

### 4.2 Best Practice

1. **Dua arah dulu:** Pastikan pelanggan mengirim pesan pertama sebelum chatbot membalas
2. **Rate limit ketat:** 5 pesan/menit sudah sangat cukup untuk chatbot normal
3. **Jangan blast:** Untuk broadcast massal, gunakan Meta Cloud API (wajib template approval)
4. **Session longevity:** Sekali di-scan, JANGAN logout unless perlu. Biarkan sesi aktif.
5. **Monitoring:** Cek `/api/wa/status` secara berkala. Jika `connected: false` dan tidak bisa reconnect setelah 5 attempt → nomor kemungkinan sudah diblokir.
6. **Nomor cadangan:** Sediakan nomor cadangan. Jika diblokir, migrasi ke nomor baru dengan scan QR ulang.

### 4.3 Tanda-Tanda Akan Dibanned

```
⚠️ Pesan terkirim tapi tidak delivered (hanya satu centang)
⚠️ Terkirim lalu hilang setelah beberapa menit
⚠️ WhatsApp Web otomatis logout terus-menerus
⚠️ Mendapat pesan "Nomor ini tidak bisa digunakan dengan WhatsApp Web"
```

---

## 5. Troubleshooting

### QR Code tidak muncul / expired terus

**Penyebab:** wa-gateway sudah memiliki sesi tersimpan tapi sesi tersebut invalid/expired.

**Solusi:**
```bash
# Logout sesi lama, generate QR baru
curl -X POST "http://localhost:8202/api/wa/logout?tenant_id=<id>"

# Kemudian generate QR baru
curl "http://localhost:8202/api/wa/qr?tenant_id=<id>"
```

### Status connected: false

**Penyebab 1:** WhatsApp update menyebabkan sesi invalid.

**Solusi:** Logout + scan QR ulang.

**Penyebab 2:** Server internet terputus.

**Solusi:** Tunggu auto-reconnect. Cek log:
```bash
tail -f logs/wa-gateway.log | grep -i "disconnected\|reconnect"
```

### Rate limit exceeded (429)

**Penyebab:** Tenant mengirim >5 pesan/menit.

**Solusi:** Tunggu 1 menit, atau alihkan pesan non-kritis ke queue.

### Session owned by another instance

**Penyebab:** Dua instance wa-gateway bersamaan, lock belum release.

**Solusi:** Lock expire dalam 5 menit. Atau restart instance yang seharusnya ownership.

### "Client not connected for tenant"

**Penyebab:** Tenant belum scan QR, atau sesi terputus.

**Solusi:** Cek `/api/wa/status`, scan QR jika perlu.

---

## 6. Schema Reference

### `wa_tenant_sessions` (created in-code, not migration)

```sql
CREATE TABLE IF NOT EXISTS wa_tenant_sessions (
    tenant_id TEXT PRIMARY KEY,
    jid       VARCHAR NOT NULL
);
```

| Kolom | Tipe | Deskripsi |
|:------|:-----|:---------|
| `tenant_id` | TEXT | UUID tenant (PK) |
| `jid` | VARCHAR | WhatsApp JID (contoh: `628123@s.whatsapp.net`) |

> **Catatan:** Tabel ini dibuat langsung di `services/wa-gateway/main.go` saat startup, bukan via migration file. whatsmeow store (device keys, session data) disimpan di tabel-tabel internal whatsmeow (`device_store`, `app_state`, dll) yang juga di PostgreSQL.

### `wa_cloud_api_credentials` (migration 000030)

```sql
CREATE TABLE wa_cloud_api_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    phone_number_id VARCHAR NOT NULL,
    waba_id         VARCHAR NOT NULL,
    access_token    TEXT NOT NULL,
    verify_token    VARCHAR NOT NULL,
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_cloud_phone ON wa_cloud_api_credentials(phone_number_id);
```

---

## 7. Environment Variables

| Variable | Default | Deskripsi |
|:---------|:--------|:----------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5433` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | `postgres` | PostgreSQL password |
| `DB_NAME` | `wch_core` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | _(empty)_ | Redis password |
| `REDIS_DB` | `9` | Redis DB untuk WA coordination |
| `APP_ENV` | _(empty)_ | Set `production` untuk gunakan `wa-cloud-api:8210` |
| `WA_CLOUD_API_URL_PORT` | _(unused)_ | Legacy, diabaikan |

---

## 8. Quick Reference Card

```
# Jalankan wa-gateway
make run-wagateway

# Cek status semua tenant
curl "http://localhost:8202/health" | jq

# Generate QR untuk tenant baru
curl "http://localhost:8202/api/wa/qr?tenant_id=<UUID>" | jq '.qr' | base64 -d > qr.png && open qr.png

# Cek koneksi satu tenant
curl "http://localhost:8202/api/wa/status?tenant_id=<UUID>" | jq

# Kirim pesan test
curl -X POST "http://localhost:8202/api/wa/send" \
  -d "tenant_id=<UUID>&target=6281234567890&message=Test from WA Gateway"

# Logout & reset sesi
curl -X POST "http://localhost:8202/api/wa/logout?tenant_id=<UUID>"

# Monitor metrics
curl "http://localhost:8202/metrics"

# Lihat log reconnect
tail -f logs/wa-gateway.log | grep -i reconnect
```
