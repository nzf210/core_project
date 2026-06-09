# 📘 WCH Platform — Panduan Lengkap Menambah Feature Baru

> Dokumen ini menjelaskan **langkah demi langkah** cara menambahkan feature baru di project WCH Platform.
> Untuk setiap skenario, dijelaskan **file mana yang perlu dibuat/diubah** dan **contoh kode nyata** dari project ini.
>
> **💡 Rekomendasi:** Lihat [CONTRIBUTING.md](../CONTRIBUTING.md) untuk panduan yang lebih ringkas dan up-to-date.

---

## 📑 Daftar Isi

1. [Arsitektur Ringkas](#-arsitektur-ringkas)
2. [Skenario 1: Menambah Endpoint Baru di App yang Sudah Ada](#-skenario-1-menambah-endpoint-baru-di-app-yang-sudah-ada)
3. [Skenario 2: Menambah Service Baru di `services/`](#-skenario-2-menambah-service-baru-di-services)
4. [Skenario 3: Menambah Modul/Sub-App Baru di `apps/`](#-skenario-3-menambah-modulsub-app-baru-di-apps)
5. [Skenario 4: Menambah Tabel Database (Migration)](#-skenario-4-menambah-tabel-database-migration)
6. [Skenario 5: Menambah Halaman/Komponen Frontend](#-skenario-5-menambah-halamankomponen-frontend)
7. [Skenario 6: Menambah Background Worker](#-skenario-6-menambah-background-worker)
8. [Skenario 7: Menambah Config/Environment Variable Baru](#-skenario-7-menambah-configenvironment-variable-baru)
9. [Checklist Lengkap Setiap Kali Tambah Feature](#-checklist-lengkap-setiap-kali-tambah-feature)
10. [Quick Reference: Di Mana Harus Menulis Apa?](#-quick-reference-di-mana-harus-menulis-apa)
11. [Konvensi Kode Wajib](#-konvensi-kode-wajib)
12. [Perintah Penting](#-perintah-penting)

---

## 🏗️ Arsitektur Ringkas

```
core_project/                         ← Root monorepo (satu go.mod)
├── apps/                             ← Produk/Aplikasi bisnis
│   ├── umkm/                         ← UMKM SaaS
│   │   ├── accounting/main.go        ← Accounting engine (Port 8201)
│   │   ├── chatbot/main.go           ← AI Chatbot via WhatsApp (Port 8202)
│   │   ├── business/main.go          ← Business management API (Port 9001)
│   │   └── automation/main.go        ← Background automation worker
│   └── campaign/                     ← Campaign management
│       └── api/                      ← REST API (Port 9002)
│           ├── main.go               ← Entry point & router
│           ├── handlers/             ← HTTP handlers (per-fitur)
│           │   ├── campaign.go
│           │   ├── volunteer.go
│           │   ├── voter.go
│           │   ├── task.go
│           │   ├── event.go
│           │   ├── survey.go
│           │   ├── notification.go
│           │   ├── access.go
│           │   ├── report.go
│           │   ├── region.go
│           │   └── responses.go      ← Helper WriteJSON & ExtractTenantID
│           └── repository/
│               └── db.go             ← Database connection pool
│
├── services/                         ← Shared services (lintas produk)
│   ├── api-gateway/                  ← Reverse proxy + routing (Port 8000)
│   │   └── main.go
│   ├── auth-service/                 ← JWT login/register (Port 8001)
│   │   ├── main.go                   ← Semua handler + router
│   │   ├── jwt.go                    ← Token generate & validate
│   │   ├── db.go                     ← DB connection
│   │   └── auth_test.go              ← Unit tests
│   ├── ai-gateway/                   ← LLM proxy + semantic cache (Port 8002)
│   │   ├── main.go
│   │   ├── db.go
│   │   └── gateway_test.go
│   ├── billing-service/              ← Subscription & payment (Port 8003)
│   │   ├── main.go
│   │   └── schema.sql
│   ├── wa-gateway/                   ← WhatsApp via whatsmeow (Port 8202)
│   │   └── main.go
│   ├── subscription-worker/          ← Freeze expired tenants (Port 8006)
│   │   └── main.go
│   └── notification-service/         ← Notifikasi Telegram/Email (Port 8005)
│       └── main.go
│
├── shared/                           ← Kode yang dipakai bersama
│   ├── sdk/
│   │   ├── config/config.go          ← SATU-SATUNYA cara baca konfigurasi
│   │   ├── auth/middleware.go        ← JWT middleware untuk protect routes
│   │   ├── db/postgres.go            ← PostgreSQL connection pool helper
│   │   ├── cache/redis.go            ← Redis connection helper
│   │   ├── migrate/migrate.go        ← Auto-migration runner (dipakai tiap startup)
│   │   └── response/http.go          ← Standard JSON response helper
│   └── migrations/                   ← Database migration files (000001 — 000028)
│       ├── 000001_init_schema.up.sql
│       ├── 000002_accounting_schema.up.sql
│       ├── ...                       ← (000003 — 000027)
│       └── 000028_campaign_features_2.up.sql
│
├── frontend/                         ← Frontend apps
│   ├── umkm-web/                     ← Vue 3 + Vite (UMKM dashboard)
│   └── campaign-web/                 ← Vue 3 + Vite (Campaign dashboard)
│
├── infra/                            ← Infrastructure & deployment
│   ├── docker/
│   ├── deploy/
│   └── n8n/
│
├── .github/workflows/deploy.yml      ← CI/CD pipeline
├── docker-compose.yml                ← Docker orchestration
├── Dockerfile                        ← Multi-stage build
├── Makefile                          ← Shortcut commands
├── go.mod                            ← Go module (satu untuk semua)
└── .env                              ← Environment variables
```

---

## 🔧 Skenario 1: Menambah Endpoint Baru di App yang Sudah Ada

**Contoh:** Menambah endpoint `GET /donors` dan `POST /donors` di Campaign app.

### Langkah-langkah:

#### Step 1️⃣ — Buat Migration SQL (jika butuh tabel baru)

Buat file baru di `shared/migrations/`:

```
shared/migrations/YYYYMMDD_donor_schema.up.sql
shared/migrations/YYYYMMDD_donor_schema.down.sql
```

**Isi `up.sql`:**
```sql
CREATE TABLE donors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    amount BIGINT NOT NULL DEFAULT 0,  -- dalam satuan SEN (int64)
    campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_donors_tenant_id ON donors(tenant_id);
```

**Isi `down.sql`:**
```sql
DROP TABLE IF EXISTS donors;
```

#### Step 2️⃣ — Buat Handler File

Buat file: `apps/campaign/api/handlers/donor.go`

```go
package handlers

import (
    "context"
    "encoding/json"
    "net/http"

    "core_project/apps/campaign/api/repository"
)

type Donor struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    Phone      string `json:"phone"`
    Amount     int64  `json:"amount"`
    CampaignID string `json:"campaign_id"`
}

func HandleDonors(w http.ResponseWriter, r *http.Request) {
    tenantID := ExtractTenantID(r)
    if tenantID == "" {
        WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
        return
    }

    if r.Method == http.MethodGet {
        rows, err := repository.DB.Query(context.Background(),
            "SELECT id, name, COALESCE(phone, ''), amount, COALESCE(campaign_id::text, '') FROM donors WHERE tenant_id = $1", tenantID)
        if err != nil {
            WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Database error"})
            return
        }
        defer rows.Close()

        var donors []Donor
        for rows.Next() {
            var d Donor
            if err := rows.Scan(&d.ID, &d.Name, &d.Phone, &d.Amount, &d.CampaignID); err == nil {
                donors = append(donors, d)
            }
        }

        if donors == nil {
            donors = []Donor{}
        }

        WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: donors})
        return
    }

    if r.Method == http.MethodPost {
        var req struct {
            Name       string `json:"name"`
            Phone      string `json:"phone"`
            Amount     int64  `json:"amount"`
            CampaignID string `json:"campaign_id"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Invalid request payload"})
            return
        }

        var id string
        err := repository.DB.QueryRow(context.Background(),
            "INSERT INTO donors (tenant_id, name, phone, amount, campaign_id) VALUES ($1, $2, $3, $4, $5) RETURNING id",
            tenantID, req.Name, req.Phone, req.Amount, req.CampaignID).Scan(&id)

        if err != nil {
            WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create donor"})
            return
        }

        WriteJSON(w, http.StatusOK, APIResponse{Success: true, Message: "Donor created", Data: map[string]string{"id": id}})
        return
    }

    WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
}
```

#### Step 3️⃣ — Daftarkan Route di `main.go`

Edit: `apps/campaign/api/main.go`

```go
// Tambahkan di bagian route registration:
mux.HandleFunc("/donors", handlers.HandleDonors)
```

#### Step 4️⃣ — Tambahkan ke API Gateway (jika perlu akses dari luar)

Edit: `services/api-gateway/main.go` — jika endpoint ini harus bisa diakses melalui gateway utama, pastikan sudah ada proxy ke campaign-api.

#### Step 5️⃣ — Buat Unit Test

Buat file: `apps/campaign/api/handlers/donor_test.go`

```go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHandleDonors_MissingTenantID(t *testing.T) {
    req, _ := http.NewRequest("GET", "/donors", nil)
    rr := httptest.NewRecorder()
    HandleDonors(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", rr.Code)
    }
}
```

#### Step 6️⃣ — Jalankan & Verifikasi

```bash
# Test
go test ./apps/campaign/api/handlers/...

# Build check
go build ./apps/campaign/api/...

# Run
go run ./apps/campaign/api
```

### 📁 Ringkasan File yang Diubah/Dibuat:

| Aksi | File |
|:-----|:-----|
| **BUAT** | `shared/migrations/YYYYMMDD_donor_schema.up.sql` |
| **BUAT** | `shared/migrations/YYYYMMDD_donor_schema.down.sql` |
| **BUAT** | `apps/campaign/api/handlers/donor.go` |
| **BUAT** | `apps/campaign/api/handlers/donor_test.go` |
| **EDIT** | `apps/campaign/api/main.go` (tambah route) |

---

## 🔧 Skenario 2: Menambah Service Baru di `services/`

**Contoh:** Menambah `notification-service` untuk kirim email & WA.

### Struktur Folder:

```
services/notification-service/
├── main.go            ← Entry point, router, handlers
├── db.go              ← Database connection (opsional)
├── sender.go          ← Business logic (email, WA)
└── notification_test.go
```

### Langkah-langkah:

#### Step 1️⃣ — Buat Folder & main.go

```go
// services/notification-service/main.go
package main

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "os"
    "time"

    "core_project/shared/sdk/config"
)

type Response struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    cfg := config.LoadConfig(".env")
    _ = cfg  // Gunakan cfg untuk WhatsApp.FonnteToken, SMTP, dll

    mux := http.NewServeMux()
    mux.HandleFunc("/send/email", handleSendEmail)
    mux.HandleFunc("/send/wa", handleSendWA)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, http.StatusOK, Response{Success: true, Message: "Notification service healthy"})
    })

    port := "8005"
    slog.Info("📬 Notification Service listening", "port", port)
    if err := http.ListenAndServe(":"+port, loggingMiddleware(mux)); err != nil {
        slog.Error("Server failed", "error", err)
    }
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        slog.Info("request", "method", r.Method, "path", r.URL.Path, "latency_ms", time.Since(start).Milliseconds())
    })
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(data)
}

func handleSendEmail(w http.ResponseWriter, r *http.Request) {
    // implementasi kirim email via SMTP
    writeJSON(w, http.StatusOK, Response{Success: true, Message: "Email sent"})
}

func handleSendWA(w http.ResponseWriter, r *http.Request) {
    // implementasi kirim WA via Fonnte
    writeJSON(w, http.StatusOK, Response{Success: true, Message: "WA sent"})
}
```

#### Step 2️⃣ — Update Makefile

```makefile
# Tambahkan di Makefile:
run-notification:
	@echo "Starting Notification Service on port 8005..."
	@go run ./services/notification-service
```

#### Step 3️⃣ — Update Dockerfile

Tambahkan build & copy di `Dockerfile`:
```dockerfile
# Di build stage:
RUN go build -o /bin/notification-service ./services/notification-service

# Di final stage:
COPY --from=builder /bin/notification-service /usr/local/bin/
```

#### Step 4️⃣ — Update docker-compose.yml

```yaml
  notification-service:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: wch-notification
    command: ["notification-service"]
    environment:
      - FONNTE_TOKEN=${FONNTE_TOKEN}
      - SMTP_HOST=${SMTP_HOST}
      - SMTP_PORT=${SMTP_PORT}
      - SMTP_USER=${SMTP_USER}
      - SMTP_PASS=${SMTP_PASS}
    ports:
      - "8005:8005"
```

#### Step 5️⃣ — Update API Gateway (jika perlu)

```go
// services/api-gateway/main.go — tambahkan route:
mux.Handle("/api/notification/", auth.Middleware(
    http.StripPrefix("/api/notification", newTenantProxy("http://notification-service:8005")),
))
```

#### Step 6️⃣ — Update start-all di Makefile

```makefile
# Di target start-all, tambahkan:
@nohup go run ./services/notification-service > notification.log 2>&1 & echo $$! > notification.pid
```

### 📁 Ringkasan File yang Diubah/Dibuat:

| Aksi | File |
|:-----|:-----|
| **BUAT** | `services/notification-service/main.go` |
| **BUAT** | `services/notification-service/notification_test.go` |
| **EDIT** | `Makefile` (tambah run target) |
| **EDIT** | `Dockerfile` (tambah build & copy) |
| **EDIT** | `docker-compose.yml` (tambah service) |
| **EDIT** | `services/api-gateway/main.go` (tambah proxy route) |

---

## 🔧 Skenario 3: Menambah Modul/Sub-App Baru di `apps/`

**Contoh:** Menambah modul `apps/umkm/inventory` untuk manajemen inventori.

### Ada 2 Pola di Project Ini:

#### Pola A: Flat (semua di main.go) — *untuk modul kecil*
Digunakan di: `umkm/accounting`, `umkm/chatbot`, `services/auth-service`

```
apps/umkm/inventory/
├── main.go          ← Semua: router + handlers + struct
├── db.go            ← DB connection
└── inventory_test.go
```

#### Pola B: Terstruktur (handlers + repository) — *untuk modul besar*
Digunakan di: `apps/campaign/api`

```
apps/umkm/inventory/
├── main.go                ← Entry point & router saja
├── handlers/
│   ├── product.go         ← Handler per resource
│   ├── stock.go
│   └── responses.go       ← Helper functions
├── repository/
│   └── db.go              ← DB connection pool
└── inventory_test.go
```

> **Rekomendasi:** Untuk modul baru, gunakan **Pola B** (terstruktur) agar lebih mudah di-maintain.

### Langkah-langkah (Pola B):

1. Buat migration di `shared/migrations/YYYYMMDD_inventory_schema.up.sql`
2. Buat folder `apps/umkm/inventory/` dengan struktur di atas
3. Buat `repository/db.go` (copy pattern dari `apps/campaign/api/repository/db.go`)
4. Buat handler files di `handlers/`
5. Buat `main.go` dengan router
6. Update `Makefile`, `Dockerfile`, `docker-compose.yml`
7. Buat test files

---

## 🔧 Skenario 4: Menambah Tabel Database (Migration)

### Lokasi: `shared/migrations/`

### Konvensi Penamaan:

```
# Untuk migrasi baru dengan nomor urut:
000005_feature_name.up.sql
000005_feature_name.down.sql

# Atau format tanggal:
20260523_feature_name.up.sql
20260523_feature_name.down.sql
```

### Aturan Wajib Tabel:

```sql
-- ✅ BENAR:
CREATE TABLE nama_tabel (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),      -- UUID selalu
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,  -- Multi-tenant
    -- kolom-kolom bisnis...
    created_at TIMESTAMPTZ DEFAULT NOW(),                -- Wajib
    updated_at TIMESTAMPTZ DEFAULT NOW()                 -- Wajib
);
CREATE INDEX idx_nama_tabel_tenant_id ON nama_tabel(tenant_id);  -- Index tenant

-- ❌ SALAH:
CREATE TABLE nama_tabel (
    id SERIAL PRIMARY KEY,           -- JANGAN pakai SERIAL
    name VARCHAR(255)                -- Tanpa tenant_id = data bocor antar tenant!
);
```

### Aturan Tipe Data:

| Data | Tipe SQL | Tipe Go |
|:-----|:---------|:--------|
| ID | `UUID` | `string` |
| Uang/Harga | `BIGINT` (satuan sen) | `int64` |
| Timestamp | `TIMESTAMPTZ` | `time.Time` |
| Boolean | `BOOLEAN` | `bool` |
| JSON Fleksibel | `JSONB` | `map[string]interface{}` atau custom struct |
| Data Terenkripsi | `VARCHAR(255)` | `string` (terenkripsi) |

---

## 🔧 Skenario 5: Menambah Halaman/Komponen Frontend

### Stack Frontend:
- **Framework:** Vue 3 + TypeScript
- **Build Tool:** Vite
- **Charting:** Chart.js + vue-chartjs
- **Styling:** Vanilla CSS dengan CSS Variables (glassmorphism dark theme)

### Lokasi: `frontend/umkm-web/`

### Struktur:
```
frontend/umkm-web/
├── src/
│   ├── App.vue              ← Root component + navigation
│   ├── main.ts              ← Vue app entry point
│   ├── style.css            ← Global styles & CSS variables
│   └── components/          ← Semua komponen UI
│       ├── Dashboard.vue
│       ├── Journal.vue
│       ├── AdminDashboard.vue
│       ├── Chatbot.vue
│       └── HelloWorld.vue
├── package.json
└── vite.config.ts
```

### Langkah Tambah Halaman Baru:

#### Step 1️⃣ — Buat Komponen Baru

Buat file: `frontend/umkm-web/src/components/Inventory.vue`

```vue
<template>
  <div class="card">
    <h2 class="text-gradient">📦 Inventori</h2>
    <div class="grid grid-2">
      <!-- Konten inventori -->
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const items = ref<any[]>([])

const API_BASE = 'http://localhost:9003'  // inventory service port

async function fetchItems() {
  try {
    const res = await fetch(`${API_BASE}/products`, {
      headers: { 'X-Tenant-ID': 'your-tenant-id' }
    })
    const data = await res.json()
    if (data.success) items.value = data.data
  } catch (err) {
    console.error('Failed to fetch items:', err)
  }
}

onMounted(fetchItems)
</script>

<style scoped>
/* Gunakan CSS class dari style.css global: card, text-gradient, grid, dll */
</style>
```

#### Step 2️⃣ — Daftarkan di App.vue

Edit: `frontend/umkm-web/src/App.vue`

```vue
<!-- Tambah tombol navigasi -->
<button 
  @click="currentView = 'inventory'" 
  :class="['nav-btn', currentView === 'inventory' ? 'active' : '']"
>
  Inventori
</button>

<!-- Tambah render view -->
<Inventory v-if="currentView === 'inventory'" />

<!-- Tambah import -->
<script setup lang="ts">
import Inventory from './components/Inventory.vue'
</script>
```

#### Step 3️⃣ — Jalankan

```bash
cd frontend/umkm-web
npm run dev
```

---

## 🔧 Skenario 6: Menambah Background Worker

**Contoh:** Menambah worker untuk auto-generate laporan bulanan.

### Lokasi: `apps/umkm/automation/` atau buat baru.

### Pola Worker di Project Ini:

```go
package main

import (
    "log/slog"
    "os"
    "time"

    "core_project/shared/sdk/config"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    _ = config.LoadConfig(".env")

    slog.Info("Worker started")

    ticker := time.NewTicker(10 * time.Second) // Interval sesuai kebutuhan
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            processJob()
        }
    }
}

func processJob() {
    slog.Info("Processing job...")
    // Logic di sini
}
```

### Atau Pola Redis Pub/Sub:

```go
// Subscribe ke channel Redis untuk event-driven processing
subscriber := redisClient.Subscribe(ctx, "tenant_events")
for msg := range subscriber.Channel() {
    handleEvent(msg.Payload)
}
```

### File yang Perlu Diubah/Dibuat:

| Aksi | File |
|:-----|:-----|
| **BUAT** | `apps/<product>/<worker-name>/main.go` |
| **EDIT** | `Makefile` (tambah run target) |
| **EDIT** | `Dockerfile` (tambah build) |
| **EDIT** | `docker-compose.yml` (tambah service) |

---

## 🔧 Skenario 7: Menambah Config/Environment Variable Baru

### Step 1️⃣ — Tambah di struct Config

Edit: `shared/sdk/config/config.go`

```go
type Config struct {
    // ... existing fields ...

    // Tambahkan field baru di struct yang sesuai:
    MyNewService struct {
        APIKey  string
        BaseURL string
    }
}
```

### Step 2️⃣ — Load di LoadConfig()

```go
// Di dalam fungsi LoadConfig():
cfg.MyNewService.APIKey = getEnv("MY_NEW_SERVICE_API_KEY", "")
cfg.MyNewService.BaseURL = getEnv("MY_NEW_SERVICE_BASE_URL", "https://api.example.com")
```

### Step 3️⃣ — Tambah di .env

```env
# My New Service
MY_NEW_SERVICE_API_KEY=your_api_key_here
MY_NEW_SERVICE_BASE_URL=https://api.example.com
```

### Step 4️⃣ — Tambah di docker-compose.yml

```yaml
environment:
  - MY_NEW_SERVICE_API_KEY=${MY_NEW_SERVICE_API_KEY}
```

> ⚠️ **JANGAN** commit `.env` ke git! Tambahkan variabel baru juga ke `.env.example` jika ada.

---

## ✅ Checklist Lengkap Setiap Kali Tambah Feature

Gunakan checklist ini setiap kali menambah feature baru:

```
☐ 1. MIGRATION — Buat file SQL di shared/migrations/ (jika butuh tabel baru)
☐ 2. BACKEND  — Buat/edit handler + struct di apps/ atau services/
☐ 3. ROUTE    — Daftarkan endpoint baru di main.go
☐ 4. GATEWAY  — Tambah proxy di api-gateway jika perlu akses external
☐ 5. TEST     — Buat unit test di file *_test.go
☐ 6. CONFIG   — Tambah env variable di config.go + .env (jika perlu)
☐ 7. FRONTEND — Buat/edit komponen Vue di frontend/
☐ 8. DOCKER   — Update Dockerfile + docker-compose.yml (jika service baru)
☐ 9. MAKEFILE — Tambah run target di Makefile (jika service baru)
☐ 10. BUILD   — Pastikan go build ./... PASS
☐ 11. TEST    — Pastikan go test ./... PASS
☐ 12. DOCS    — Update dokumentasi di docs/ jika perlu
```

---

## 📋 Quick Reference: Di Mana Harus Menulis Apa?

| Mau Apa? | Di Mana? |
|:----------|:---------|
| Tambah endpoint di UMKM accounting | `apps/umkm/accounting/main.go` |
| Tambah endpoint di UMKM chatbot | `apps/umkm/chatbot/main.go` |
| Tambah endpoint di Campaign | `apps/campaign/api/handlers/` (buat file baru) + daftar di `apps/campaign/api/main.go` |
| Tambah endpoint di Crypto | `apps/crypto/api/` (buat struktur baru) |
| Tambah service baru (auth, billing, dll) | `services/<nama-service>/main.go` |
| Tambah background worker | `apps/<product>/<worker>/main.go` |
| Tambah tabel database | `shared/migrations/<nomor>_<nama>.up.sql` |
| Ubah config/env variable | `shared/sdk/config/config.go` + `.env` |
| Tambah auth middleware | `shared/sdk/auth/middleware.go` |
| Tambah response helper | `shared/sdk/response/http.go` |
| Tambah DB helper | `shared/sdk/db/postgres.go` |
| Tambah Redis helper | `shared/sdk/cache/redis.go` |
| Tambah halaman frontend UMKM | `frontend/umkm-web/src/components/` + update `App.vue` |
| Tambah halaman frontend Crypto | `frontend/crypto-web/src/` |
| Tambah halaman frontend Campaign | `frontend/campaign-web/src/` |
| Deploy baru / CI-CD | `.github/workflows/deploy.yml` |
| Ubah Docker setup | `docker-compose.yml` + `Dockerfile` |
| Tambah make command | `Makefile` |
| Panggil AI/LLM | Selalu lewat `services/ai-gateway` (POST http://localhost:8003/v1/chat) |
| Dokumentasi | `docs/` |

---

## 🚨 Konvensi Kode Wajib

### Go Backend — HARUS Diikuti:

| Aturan | Benar ✅ | Salah ❌ |
|:-------|:---------|:---------|
| HTTP Framework | `net/http` standar | Gin, Echo, dll (kecuali sudah ada) |
| Database | `pgx/v5` (parameterized query) | GORM, string concatenation |
| JWT | `golang-jwt/jwt/v5` | Library lain |
| Password | `bcrypt` (cost=12) | MD5, SHA, plain text |
| Logging | `log/slog` (structured JSON) | `fmt.Println`, `log.Println` |
| Config | `config.LoadConfig()` dari shared | `os.Getenv()` langsung |
| Uang/Harga | `int64` dalam satuan **sen** | `float64` |
| UUID | `github.com/google/uuid` | Auto-increment integer |
| Error | `return error` eksplisit | `panic()` di luar main |
| AI/LLM | Via `ai-gateway` service | Direct API call dari apps |

### Database — HARUS Diikuti:

```sql
-- SETIAP tabel harus punya:
id UUID PRIMARY KEY DEFAULT gen_random_uuid()
tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE  -- multi-tenant!
created_at TIMESTAMPTZ DEFAULT NOW()
updated_at TIMESTAMPTZ DEFAULT NOW()

-- SETIAP tabel harus punya INDEX pada tenant_id
CREATE INDEX idx_<table>_tenant_id ON <table>(tenant_id);

-- SEMUA query pakai parameterized:
-- ✅ DB.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", userID)
-- ❌ DB.QueryRow(ctx, "SELECT * FROM users WHERE id = '" + userID + "'")
```

### Keamanan — KRITIS:

| Data Sensitif | Cara Simpan |
|:-------------|:------------|
| API Key Crypto Exchange | AES-256-GCM → kolom `encrypted_api_key` |
| NIK Pemilih (Campaign) | AES-256-GCM → kolom `encrypted_nik` |
| Refresh Token | SHA-256 hash → DB + Redis |
| Password User | bcrypt (cost=12) → kolom `password_hash` |

---

## ⚡ Perintah Penting

```bash
# === DEVELOPMENT ===
make run-auth          # Jalankan auth service (port 8001)
make run-ai            # Jalankan AI gateway (port 8002)
make run-accounting    # Jalankan UMKM accounting (port 8201)
make run-chatbot       # Jalankan UMKM chatbot (port 8202)
make run-campaign      # Jalankan Campaign API (port 9002)
make start-all         # Jalankan semua di background
make stop-all          # Matikan semua
make status            # Cek status semua port

# === TESTING ===
go test ./...                              # Test semua
go test ./apps/campaign/api/handlers/...   # Test spesifik
go test -v -run TestHandleDonors ./...     # Test satu fungsi

# === BUILD ===
go build ./...         # Build semua (cek compile errors)
go vet ./...           # Cek kode quality
go mod tidy            # Bersihkan dependencies

# === DOCKER ===
docker compose up -d --build    # Start semua via Docker
docker compose down             # Stop semua
docker compose logs -f          # Lihat logs

# === FRONTEND ===
cd frontend/umkm-web && npm run dev       # Dev server UMKM
cd frontend/umkm-web && npm run build     # Production build
```

---

## 🔄 Alur Kerja Tipikal (Workflow)

```
1. Tentukan Feature → Masuk di app mana? (umkm/crypto/campaign/service?)
         ↓
2. Butuh tabel baru? → Buat migration SQL di shared/migrations/
         ↓
3. Buat Handler Go → Di apps/ atau services/ sesuai pola yang sudah ada
         ↓
4. Daftarkan Route → Di main.go app/service terkait
         ↓
5. Buat Unit Test → File *_test.go di samping kode yang ditest
         ↓
6. go build ./... → Pastikan kompilasi sukses
         ↓
7. go test ./... → Pastikan semua test pass
         ↓
8. Perlu UI? → Buat komponen Vue di frontend/
         ↓
9. Service baru? → Update Makefile, Dockerfile, docker-compose.yml
         ↓
10. Commit & Push → CI/CD otomatis via .github/workflows/deploy.yml
```

---

> **💡 Tips:** Jika ragu pattern mana yang harus diikuti, **lihat contoh terdekat** di project ini:
> - Service kecil → Lihat `services/auth-service/main.go`
> - App dengan handlers terstruktur → Lihat `apps/campaign/api/`
> - App flat (semua di main) → Lihat `apps/umkm/accounting/main.go`
> - Worker background → Lihat `apps/crypto/worker/main.go`
> - Frontend komponen → Lihat `frontend/umkm-web/src/components/Dashboard.vue`
