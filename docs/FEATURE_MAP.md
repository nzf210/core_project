# 🗺️ WCH Platform — Feature Location Map (Cheat Sheet)

> Quick reference: **Di mana saya harus menulis kode?**

---

## Mau Tambah Endpoint/API Baru?

```
┌──────────────────────────────────────────────────────────────┐
│  Ini untuk produk mana?                                      │
│                                                              │
│  UMKM Accounting ──→ apps/umkm/accounting/main.go           │
│  UMKM Chatbot    ──→ apps/umkm/chatbot/main.go              │
│  UMKM Automation ──→ apps/umkm/automation/main.go            │
│  Campaign        ──→ apps/campaign/api/handlers/<nama>.go    │
│                      + daftarkan di apps/campaign/api/main.go│
│  Crypto API      ──→ apps/crypto/api/ (buat struktur baru)  │
│  Crypto Worker   ──→ apps/crypto/worker/main.go              │
│                                                              │
│  Shared Service  ──→ services/<nama-service>/main.go         │
│  (auth, billing, AI, notification, dll)                      │
└──────────────────────────────────────────────────────────────┘
```

## Mau Tambah Tabel Database?

```
shared/migrations/<YYYYMMDD>_<nama_feature>.up.sql
shared/migrations/<YYYYMMDD>_<nama_feature>.down.sql
```

## Mau Tambah/Ubah Config?

```
1. shared/sdk/config/config.go  ← Tambah field struct + LoadConfig
2. .env                         ← Tambah key=value
3. docker-compose.yml           ← Tambah environment variable
```

## Mau Tambah UI Frontend?

```
┌──────────────────────────────────────────────────────────────┐
│  UMKM   ──→ frontend/umkm-web/src/components/<Nama>.vue     │
│              + import & daftarkan di src/App.vue              │
│  Crypto ──→ frontend/crypto-web/src/                         │
│  Campaign→ frontend/campaign-web/src/                        │
└──────────────────────────────────────────────────────────────┘
```

## Mau Tambah Service/Worker Baru?

```
WAJIB update file-file ini:
☐ Makefile                    ← Tambah run target
☐ Dockerfile                  ← Tambah go build + COPY
☐ docker-compose.yml          ← Tambah service definition
☐ services/api-gateway/main.go ← Tambah proxy route (jika REST API)
```

## Port Allocation

| Port | Service | Type | Directory |
|---|---|---|---|
| `8000` | Auth Service | REST API | `services/auth-service` |
| `8003` | AI Gateway | REST API | `services/ai-gateway` |
| `9001` | UMKM Accounting | REST API | `apps/umkm/accounting` |
| `9002` | Campaign API | REST API | `apps/campaign/api` |
| `9003` | Crypto API | REST API | `apps/crypto/api` |

### API Gateway Routing
Requests ke API Gateway diarahkan sesuai prefix URL:
- `/api/v1/auth/*` ➔ `localhost:8000`
- `/api/v1/accounting/*` ➔ `localhost:9001`
- `/api/v1/campaign/*` ➔ `localhost:9002`
- `/api/v1/crypto/*` ➔ `localhost:9003`

---

*Lihat [DEVELOPER_GUIDE.md](file:///home/syahril/Desktop/dev/core_project/docs/DEVELOPER_GUIDE.md) untuk panduan lengkap dengan contoh kode.*
