# 🤝 CONTRIBUTING — Panduan Pengembangan WCH Platform

> **Baca dokumen ini sebelum menambah atau mengubah fitur apapun.**
> Panduan ini dirancang agar semua kontributor (termasuk AI agent) bisa bekerja secara konsisten.

---

## 📑 Daftar Isi

1. [Prinsip Dasar](#-prinsip-dasar)
2. [Alur Kerja Standar](#-alur-kerja-standar)
3. [Checklist Wajib](#-checklist-wajib-setiap-menambah-fitur)
4. [Cara Menambah Endpoint Baru](#-cara-menambah-endpoint-baru)
5. [Cara Menambah Service Baru](#-cara-menambah-service-baru)
6. [Cara Menambah Tabel Database](#-cara-menambah-tabel-database)
7. [Cara Menambah Config Baru](#-cara-menambah-config--env-variable-baru)
8. [Cara Menambah Background Worker](#-cara-menambah-background-worker)
9. [Cara Menambah Halaman Frontend](#-cara-menambah-halaman-frontend)
10. [Konvensi Kode](#-konvensi-kode)
11. [Quick Reference: Di Mana Menulis Apa?](#-quick-reference-di-mana-menulis-apa)
12. [Perintah Penting](#-perintah-penting)

---

## 🧭 Prinsip Dasar

1. **Satu PR, satu fitur** — Jangan campur perubahan yang tidak berhubungan.
2. **Ikuti pola yang sudah ada** — Lihat kode terdekat sebagai referensi.
3. **Multi-tenant selalu** — Setiap query DB harus di-filter dengan `tenant_id`.
4. **Test dulu** — Pastikan `go build ./...` dan `go test ./...` pass sebelum commit.
5. **Dokumentasi** — Update `CONTRIBUTING.md` atau `docs/` jika ada perubahan arsitektur.

---

## 🔄 Alur Kerja Standar

```
1. Tentukan scope → Fitur di app mana? Service baru? Worker?
         ↓
2. Butuh tabel baru? → Buat migration SQL di shared/migrations/
         ↓
3. Butuh config baru? → Tambah di shared/sdk/config/config.go + .env + .env.example
         ↓
4. Tulis kode backend → Di apps/ atau services/ sesuai pola
         ↓
5. Daftarkan route → Di main.go app/service terkait
         ↓
6. Tulis unit test → File *_test.go di samping kode
         ↓
7. go build ./... → Cek kompilasi
         ↓
8. go test ./... → Cek semua test pass
         ↓
9. Perlu UI? → Buat komponen Vue di frontend/
         ↓
10. Service baru? → Update Makefile + Dockerfile + docker-compose.yml
         ↓
11. Commit & Push → CI/CD via .github/workflows/deploy.yml
```

---

## ✅ Checklist Wajib Setiap Menambah Fitur

Copy-paste checklist ini dan centang satu per satu:

```
☐ 1. MIGRATION   — Buat file SQL di shared/migrations/ (jika butuh tabel baru)
☐ 2. CONFIG      — Tambah env variable di config.go + .env + .env.example (jika perlu)
☐ 3. BACKEND     — Tulis handler + struct di apps/ atau services/
☐ 4. ROUTE       — Daftarkan endpoint baru di main.go
☐ 5. GATEWAY     — Tambah proxy rule di api-gateway jika perlu akses eksternal
☐ 6. TEST        — Buat unit test di file *_test.go
☐ 7. FRONTEND    — Buat/edit komponen Vue di frontend/ (jika ada UI)
☐ 8. DOCKER      — Update Dockerfile + docker-compose.yml (jika service baru)
☐ 9. MAKEFILE    — Tambah run target di Makefile (jika service baru)
☐ 10. BUILD      — Pastikan: go build ./... PASS
☐ 11. TEST       — Pastikan: go test ./... PASS
☐ 12. DOCS       — Update CONTRIBUTING.md atau docs/ jika ada perubahan arsitektur
```

---

## 🔧 Cara Menambah Endpoint Baru

### Skenario: Menambah endpoint di Campaign App

**Tujuan:** `GET /donors` dan `POST /donors`

#### Step 1 — Buat Migration SQL (jika butuh tabel baru)

```bash
# Buat file migration baru dengan nomor otomatis:
make migrate-new NAME=donors
# → Membuat: shared/migrations/000029_donors.up.sql
# → Membuat: shared/migrations/000029_donors.down.sql
```

> ⚡ **Auto-migration aktif:** Migration dijalankan otomatis saat service start.
> Tidak perlu `psql` manual. Cukup buat file `.sql`, lalu jalankan service.

**`shared/migrations/NNNNNN_donors.up.sql`:**
```sql
CREATE TABLE donors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    phone       VARCHAR(50),
    amount      BIGINT NOT NULL DEFAULT 0,  -- satuan SEN, bukan rupiah!
    campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_donors_tenant_id ON donors(tenant_id);
```

**`shared/migrations/NNNNNN_donors.down.sql`:**
```sql
DROP TABLE IF EXISTS donors;
```

#### Step 2 — Untuk Campaign App (Pola Terstruktur)

Buat file: `apps/campaign/api/handlers/donor.go`

```go
package handlers

import (
    "context"
    "encoding/json"
    "net/http"

    "core_project/apps/campaign/api/repository"
)

// Donor merepresentasikan model donor kampanye.
type Donor struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    Phone      string `json:"phone,omitempty"`
    Amount     int64  `json:"amount"` // dalam satuan SEN
    CampaignID string `json:"campaign_id,omitempty"`
}

// HandleDonors handles GET /donors dan POST /donors.
func HandleDonors(w http.ResponseWriter, r *http.Request) {
    tenantID := ExtractTenantID(r) // helper di responses.go
    if tenantID == "" {
        WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "Missing X-Tenant-ID"})
        return
    }

    switch r.Method {
    case http.MethodGet:
        handleListDonors(w, r, tenantID)
    case http.MethodPost:
        handleCreateDonor(w, r, tenantID)
    default:
        WriteJSON(w, http.StatusMethodNotAllowed, APIResponse{Message: "Method not allowed"})
    }
}

func handleListDonors(w http.ResponseWriter, r *http.Request, tenantID string) {
    rows, err := repository.DB.Query(context.Background(),
        `SELECT id, name, COALESCE(phone, ''), amount, COALESCE(campaign_id::text, '')
         FROM donors WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
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
        donors = []Donor{} // kembalikan array kosong, bukan null
    }

    WriteJSON(w, http.StatusOK, APIResponse{Success: true, Data: donors})
}

func handleCreateDonor(w http.ResponseWriter, r *http.Request, tenantID string) {
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
    if req.Name == "" {
        WriteJSON(w, http.StatusBadRequest, APIResponse{Message: "name is required"})
        return
    }

    var id string
    err := repository.DB.QueryRow(context.Background(),
        `INSERT INTO donors (tenant_id, name, phone, amount, campaign_id)
         VALUES ($1, $2, $3, $4, NULLIF($5, '')) RETURNING id`,
        tenantID, req.Name, req.Phone, req.Amount, req.CampaignID).Scan(&id)
    if err != nil {
        WriteJSON(w, http.StatusInternalServerError, APIResponse{Message: "Failed to create donor"})
        return
    }

    WriteJSON(w, http.StatusCreated, APIResponse{
        Success: true,
        Message: "Donor created",
        Data:    map[string]string{"id": id},
    })
}
```

#### Step 3 — Daftarkan Route di main.go

Edit: `apps/campaign/api/main.go`

```go
// Tambah di bagian route registration:
mux.HandleFunc("/donors", handlers.HandleDonors)
```

#### Step 4 — Buat Unit Test

Buat file: `apps/campaign/api/handlers/donor_test.go`

```go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHandleDonors_MissingTenantID(t *testing.T) {
    req, _ := http.NewRequest(http.MethodGet, "/donors", nil)
    rr := httptest.NewRecorder()
    HandleDonors(rr, req)

    if rr.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", rr.Code)
    }
}
```

#### Step 5 — Verifikasi

```bash
go test ./apps/campaign/api/handlers/...
go build ./apps/campaign/api/...
```

---

### Skenario: Menambah endpoint di UMKM Accounting (Pola Flat)

Untuk app flat (semua di `main.go` seperti `apps/umkm/accounting/`):

1. Tambah struct model di `main.go`
2. Tambah fungsi handler di `main.go` (pattern: `func handleNama(...)`)
3. Register route di fungsi `main()`:
   ```go
   mux.HandleFunc("/nama-resource", handleNama)
   ```

---

## 🔧 Cara Menambah Service Baru

**Contoh:** `services/my-service/`

### Struktur Wajib

```
services/my-service/
├── main.go         ← Entry point + router + handlers
├── db.go           ← DB connection (jika pakai DB)
└── my_service_test.go
```

### Template `main.go`

```go
package main

import (
    "log/slog"
    "net/http"
    "os"
    "time"

    "core_project/shared/sdk/config"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    cfg := config.LoadConfig(".env")
    _ = cfg

    mux := http.NewServeMux()
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"status":"ok"}`))
    })
    // Tambah route lainnya di sini

    port := "808X" // Pilih port yang belum dipakai (lihat CLAUDE.md)
    slog.Info("Service started", "port", port)
    if err := http.ListenAndServe(":"+port, loggingMiddleware(mux)); err != nil {
        slog.Error("Server failed", "error", err)
    }
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "latency_ms", time.Since(start).Milliseconds(),
        )
    })
}

func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(data)
}
```

### File yang Wajib Diupdate

| File | Perubahan |
|:-----|:----------|
| `Makefile` | Tambah `run-my-service` target |
| `Dockerfile` | Tambah `go build` + `COPY` |
| `docker-compose.yml` | Tambah service definition |
| `services/api-gateway/main.go` | Tambah proxy route (jika REST API publik) |
| `CLAUDE.md` | Update Port Registry |
| `.env.example` | Tambah env vars baru |

---

## 🗄️ Cara Menambah Tabel Database

### Cari Nomor Migration Berikutnya

```bash
ls shared/migrations/*.up.sql | grep -oP '0+\d+' | sort -n | tail -1
# Misal hasilnya 000024, maka migration baru: 000025
```

### Buat File Migration

```bash
# Gunakan make target — nomor urut otomatis:
make migrate-new NAME=nama_fitur
# → Membuat: shared/migrations/NNNNNN_nama_fitur.up.sql
# → Membuat: shared/migrations/NNNNNN_nama_fitur.down.sql

# Cek status migrations:
make migrate-status
```

> ⚡ **Auto-migration:** Setiap service jalankan migration otomatis saat startup.
> Tambah file SQL → restart service → selesai. Tidak perlu `psql` manual.

### Aturan Wajib Tabel

```sql
-- ✅ Template tabel yang benar:
CREATE TABLE nama_tabel (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- === kolom bisnis ===
    name        VARCHAR(255) NOT NULL,
    amount      BIGINT NOT NULL DEFAULT 0,  -- uang dalam SEN
    status      VARCHAR(50) NOT NULL DEFAULT 'active',
    metadata    JSONB,                        -- data fleksibel
    -- === kolom wajib ===
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Index WAJIB pada tenant_id:
CREATE INDEX idx_nama_tabel_tenant_id ON nama_tabel(tenant_id);
-- Index tambahan jika sering di-query:
CREATE INDEX idx_nama_tabel_status ON nama_tabel(status);
```

### Aturan Tipe Data

| Data | SQL | Go | Catatan |
|:-----|:----|:---|:--------|
| Primary Key | `UUID` | `string` | `gen_random_uuid()` |
| Foreign Key | `UUID` | `string` | Selalu `REFERENCES` |
| Uang/Harga | `BIGINT` | `int64` | Satuan SEN (bukan rupiah) |
| Timestamp | `TIMESTAMPTZ` | `time.Time` | Selalu timezone-aware |
| Boolean | `BOOLEAN` | `bool` | |
| JSON fleksibel | `JSONB` | `map[string]any` | Untuk data dinamis |
| Data terenkripsi | `TEXT` | `string` | Simpan ciphertext AES-GCM |
| File path | `TEXT` | `string` | Relatif ke root uploads/ |

---

## 🔧 Cara Menambah Config / Env Variable Baru

### Step 1 — Tambah di Struct Config

Edit: `shared/sdk/config/config.go`

```go
type Config struct {
    // ... existing ...

    // Tambah di nested struct yang sesuai, atau buat baru:
    MyService struct {
        APIKey  string
        BaseURL string
        Timeout int
    }
}
```

### Step 2 — Load di `LoadConfig()`

```go
// Di dalam fungsi LoadConfig():
cfg.MyService.APIKey  = getEnv("MY_SERVICE_API_KEY", "")
cfg.MyService.BaseURL = getEnv("MY_SERVICE_BASE_URL", "https://api.example.com")
cfg.MyService.Timeout = getEnvAsInt("MY_SERVICE_TIMEOUT_SECONDS", 30)
```

### Step 3 — Tambah di `.env.example`

```bash
# My Service Configuration
MY_SERVICE_API_KEY=
MY_SERVICE_BASE_URL=https://api.example.com
MY_SERVICE_TIMEOUT_SECONDS=30
```

### Step 4 — Update `docker-compose.yml`

```yaml
services:
  my-service:
    environment:
      - MY_SERVICE_API_KEY=${MY_SERVICE_API_KEY}
      - MY_SERVICE_BASE_URL=${MY_SERVICE_BASE_URL}
```

> ⚠️ JANGAN commit `.env`. SELALU update `.env.example`.

---

## 🔧 Cara Menambah Background Worker

### Template Worker

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

    cfg := config.LoadConfig(".env")
    slog.Info("Worker started")

    // Opsi 1: Ticker (interval tetap)
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        processJob(cfg)
    }
}

func processJob(cfg *config.Config) {
    slog.Info("Processing job...")
    // Logic di sini
}
```

### Atau Pola Redis Pub/Sub (event-driven)

```go
subscriber := redisClient.Subscribe(ctx, "tenant_events")
for msg := range subscriber.Channel() {
    handleEvent(msg.Payload)
}
```

---

## 🔧 Cara Menambah Halaman Frontend

### Stack: Vue 3 + TypeScript + Vite

### Struktur Komponen

```
frontend/umkm-web/src/components/NamaHalaman.vue
```

### Template Komponen

```vue
<template>
  <div class="card">
    <h2 class="text-gradient">📦 Judul Halaman</h2>
    
    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else class="grid grid-2">
      <!-- Konten halaman -->
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Item {
  id: string
  name: string
}

// Ambil tenant ID dari localStorage (disimpan saat login)
const tenantID = localStorage.getItem('tenantId') || ''
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8201'

const items = ref<Item[]>([])
const loading = ref(true)
const error = ref('')

async function fetchItems() {
  try {
    const res = await fetch(`${API_BASE}/nama-resource`, {
      headers: { 'X-Tenant-ID': tenantID }
    })
    const data = await res.json()
    if (data.success) items.value = data.data
  } catch (err) {
    error.value = 'Gagal memuat data'
    console.error('Fetch failed:', err)
  } finally {
    loading.value = false
  }
}

onMounted(fetchItems)
</script>

<style scoped>
/* Gunakan CSS class dari style.css global: card, text-gradient, grid, dll */
</style>
```

### Daftarkan di App.vue

```vue
<!-- 1. Import di <script setup> -->
<script setup lang="ts">
import NamaHalaman from './components/NamaHalaman.vue'
</script>

<!-- 2. Tambah tombol navigasi -->
<button @click="currentView = 'namaHalaman'" :class="['nav-btn', currentView === 'namaHalaman' ? 'active' : '']">
  Nama Halaman
</button>

<!-- 3. Tambah render kondisional -->
<NamaHalaman v-if="currentView === 'namaHalaman'" />
```

---

## 📏 Konvensi Kode

### Go — Aturan Wajib

```go
// ✅ Logging yang benar (structured)
slog.Info("Operation completed", "user_id", userID, "duration_ms", ms)
slog.Error("Database failed", "error", err, "query", query)

// ❌ Jangan ini:
fmt.Println("Operation completed")
log.Printf("Error: %v", err)

// ✅ Error handling yang benar
rows, err := DB.Query(ctx, query, args...)
if err != nil {
    return fmt.Errorf("query users: %w", err)
}

// ❌ Jangan ini:
rows, _ := DB.Query(ctx, query, args...) // ignore error
if err != nil { panic(err) }              // panic di luar main

// ✅ Uang dalam satuan sen
price := int64(150000) // Rp 1.500,00 (bukan 1500.00)

// ❌ Jangan ini:
price := 1500.00 // float64 untuk uang = bug potensial!
```

### SQL — Aturan Wajib

```sql
-- ✅ Selalu parameterized query:
SELECT * FROM products WHERE tenant_id = $1 AND status = $2

-- ✅ Selalu filter tenant_id:
SELECT * FROM products WHERE tenant_id = $1

-- ❌ JANGAN string concatenation:
"SELECT * FROM products WHERE tenant_id = '" + tenantID + "'"

-- ❌ JANGAN query tanpa tenant_id (data bocor antar tenant!):
SELECT * FROM products WHERE status = 'active'
```

### Penamaan File

| Tipe | Konvensi | Contoh |
|:-----|:---------|:-------|
| Go handler | `nama_resource.go` | `donor.go`, `voter.go` |
| Go test | `nama_resource_test.go` | `donor_test.go` |
| Migration up | `NNNNNN_nama.up.sql` | `000025_donors.up.sql` |
| Migration down | `NNNNNN_nama.down.sql` | `000025_donors.down.sql` |
| Vue component | `NamaHalaman.vue` (PascalCase) | `DonorList.vue` |

---

## 📋 Quick Reference: Di Mana Menulis Apa?

| Tujuan | File/Direktori |
|:-------|:---------------|
| Endpoint UMKM Accounting | `apps/umkm/accounting/main.go` (flat pattern) |
| Endpoint UMKM Business | `apps/umkm/business/main.go` (flat pattern) |
| Endpoint UMKM Chatbot | `apps/umkm/chatbot/main.go` (flat pattern) |
| Endpoint Campaign | `apps/campaign/api/handlers/nama.go` + register di `main.go` |
| Endpoint Crypto API | `apps/crypto/api/handlers.go` + `repository.go` |
| Background Worker UMKM | `apps/umkm/automation/main.go` |
| Background Worker Crypto | `apps/crypto/worker/` |
| Service baru lintas produk | `services/nama-service/main.go` |
| Tabel database baru | `shared/migrations/NNNNNN_nama.up.sql` |
| Config / env variable | `shared/sdk/config/config.go` + `.env.example` |
| JWT middleware | `shared/sdk/auth/middleware.go` |
| Standard JSON response | `shared/sdk/response/http.go` |
| DB connection helper | `shared/sdk/db/postgres.go` |
| Redis helper | `shared/sdk/cache/redis.go` |
| Frontend UMKM | `frontend/umkm-web/src/components/` |
| Frontend Crypto | `frontend/crypto-web/src/` |
| Frontend Campaign | `frontend/campaign-web/src/` |
| AI/LLM call | Via `services/ai-gateway` — `POST http://localhost:8002/v1/chat` |
| Proxy routing | `services/api-gateway/main.go` |
| Docker orchestration | `docker-compose.yml` |
| Build/deploy | `Dockerfile` |
| Dev shortcuts | `Makefile` |
| CI/CD pipeline | `.github/workflows/deploy.yml` |

---

## ⚡ Perintah Penting

```bash
# === DEVELOPMENT ===
make start-all          # Jalankan semua service di background
                        # Log  → logs/*.log
                        # PID  → run/*.pid
make stop-all           # Matikan semua service (baca PID dari run/)
make status             # Cek status semua port
make run-auth           # Auth Service (port 8001)
make run-ai             # AI Gateway (port 8002)
make run-accounting     # UMKM Accounting (port 8201)
make run-chatbot        # UMKM Chatbot (port 8202)
make run-crypto-api     # Crypto API (port 8101)
make run-campaign       # Campaign API (port 9002)
make run-frontend       # Semua frontend

# === BUILD (local binary ke bin/, BUKAN ke root!) ===
make build-all          # Build semua → bin/<service>
make build              # Compile check (go build ./...)

# === LOGS ===
make logs-auth          # tail -f logs/auth.log
make logs-accounting    # tail -f logs/accounting.log
make logs-campaign      # tail -f logs/campaign-api.log
make logs-all           # tail -f logs/*.log (semua sekaligus)

# === TESTING ===
go test ./...                                   # Test semua package
go test ./apps/campaign/api/handlers/...        # Test package tertentu
go test -v -run TestHandleDonors ./...          # Test satu fungsi
go test -race ./...                             # Test dengan race detector
make check                                      # tidy + vet + build + test

# === QUALITY ===
go build ./...    # Compile check (wajib sebelum commit)
go vet ./...      # Static analysis
go mod tidy       # Bersihkan dependencies

# === DATABASE ===
# Lihat migration terakhir:
ls shared/migrations/*.up.sql | tail -5
# Buat migration baru (nomor otomatis):
make migrate-new NAME=nama_fitur
# Cek status migrations:
make migrate-status
# ⚡ Migration jalan otomatis saat service start — tidak perlu psql manual!

# === CLEANUP ===
make clean-logs         # Hapus semua file di logs/
make clean-build        # Hapus semua binary di bin/
make clean              # clean-logs + clean-build

# === FRONTEND ===
cd frontend/umkm-web && npm install && npm run dev   # UMKM (port 3201)
cd frontend/crypto-web && npm install && npm run dev  # Crypto (port 3101)
cd frontend/campaign-web && npm install && npm run dev # Campaign (port 3301)

# === DOCKER ===
docker compose up -d              # Start infra (postgres, redis)
docker compose up -d --build      # Rebuild & start semua
docker compose down               # Stop semua
docker compose logs -f            # Lihat logs real-time
docker compose logs -f auth-service  # Logs service tertentu
```


---

> 💡 **Tips:** Jika ragu pattern mana yang harus diikuti, lihat contoh terdekat:
> - Handler terstruktur → `apps/campaign/api/handlers/voter.go`
> - App flat → `apps/umkm/accounting/main.go`
> - Service sederhana → `services/notification-service/main.go`
> - Worker background → `apps/umkm/automation/main.go`
> - Enkripsi → `apps/crypto/domain/encryption.go`
