# WCH Platform — VPS Deploy Guide

Setup Nginx + SSL untuk staging dan production di VPS baru.

---

## Prasyarat

1. User `deploy` sudah ada di VPS dengan akses sudo
2. SSH key `~/.ssh/github_actions_wch` sudah terdaftar di VPS (`~/.ssh/authorized_keys`)
3. DNS domain sudah pointing ke IP VPS sebelum request SSL

---

## Setup VPS (Jalankan Sekali)

Script `setup-vps.sh` dijalankan dari **local machine**, bukan dari VPS.

### Satu VPS untuk Staging + Production (shared)

```bash
bash infra/deploy/setup-vps.sh shared
```

### Dedicated VPS Staging saja

```bash
bash infra/deploy/setup-vps.sh staging
```

### Dedicated VPS Production saja

```bash
bash infra/deploy/setup-vps.sh production
```

> Script akan tanya VPS IP dan port jika belum di-set via env var `VPS_HOST` / `VPS_PORT`.

Script otomatis akan:
- Buat folder `/opt/wch-staging` dan/atau `/opt/wch-production`
- Setup passwordless sudo untuk docker & nginx
- Install Nginx + Certbot
- Deploy nginx config yang sesuai
- Tampilkan perintah certbot dan cara copy `.env`

---

## Setelah Setup Script Selesai

### 1. Copy file `.env` ke VPS

```bash
# Staging
scp -i ~/.ssh/github_actions_wch -P 3209 .env.staging deploy@<VPS_IP>:/opt/wch-staging/.env.staging

# Production
scp -i ~/.ssh/github_actions_wch -P 3209 .env.production deploy@<VPS_IP>:/opt/wch-production/.env.production
```

### 2. Request SSL via Certbot

Pastikan DNS sudah pointing ke VPS terlebih dahulu, lalu jalankan:

```bash
# Staging (semua domain dalam satu cert SAN)
ssh -i ~/.ssh/github_actions_wch -p 3209 deploy@<VPS_IP> \
  "sudo certbot --nginx --non-interactive --agree-tos -m admin@umkmai.id \
    -d stg-api.umkmai.id \
    -d stg-grf.umkmai.id \
    -d stg-n8n.umkmai.id \
    -d stg.umkmai.id \
    -d stg-spadmin.umkmai.id"

# Production
ssh -i ~/.ssh/github_actions_wch -p 3209 deploy@<VPS_IP> \
  "sudo certbot --nginx --non-interactive --agree-tos -m admin@umkmai.id \
    -d api.umkmai.id \
    -d grf.umkmai.id \
    -d n8n.umkmai.id \
    -d umkmai.id \
    -d spadmin.umkmai.id"
```

### 3. Trigger Deploy via GitHub Actions

```bash
# Staging
git tag stg-be-v1.0.0 && git push origin stg-be-v1.0.0

# Production
git tag prod-be-v1.0.0 && git push origin prod-be-v1.0.0
```

---

## Domain & Port Mapping

### Staging

| Domain | Port | Service |
|:-------|:-----|:--------|
| `stg-api.umkmai.id` | `21000` | API Gateway |
| `stg.umkmai.id` | `22001` | UMKM Web |
| `stg-spadmin.umkmai.id` | `22002` | Superadmin Web |
| `stg-grf.umkmai.id` | `23001` | Grafana |
| `stg-n8n.umkmai.id` | `23678` | N8N |

### Production

| Domain | Port | Service |
|:-------|:-----|:--------|
| `api.umkmai.id` | `8000` | API Gateway |
| `umkmai.id` | `8101` | UMKM Web |
| `spadmin.umkmai.id` | `8102` | Superadmin Web |
| `grf.umkmai.id` | `3000` | Grafana |
| `n8n.umkmai.id` | `5678` | N8N |

---

## GitHub Secrets yang Dibutuhkan

Set di **GitHub repo → Settings → Secrets → Environments**.

| Secret | Nilai | Environment |
|:-------|:------|:------------|
| `VPS_HOST` | IP VPS staging | `staging` |
| `VPS_USERNAME` | `deploy` | `staging` |
| `VPS_SSH_KEY` | Isi dari `~/.ssh/github_actions_wch` | `staging` |
| `GHCR_TOKEN` | GitHub Personal Access Token (scope: `read:packages`) | `staging` |
| `GHCR_USERNAME` | GitHub username (`nzf210`) | `staging` |
| `VPS_HOST` | IP VPS production | `production` |
| `VPS_USERNAME` | `deploy` | `production` |
| `VPS_SSH_KEY` | Isi dari `~/.ssh/github_actions_wch` (atau key berbeda) | `production` |
| `GHCR_TOKEN` | GitHub Personal Access Token (scope: `read:packages`) | `production` |
| `GHCR_USERNAME` | GitHub username (`nzf210`) | `production` |

---

## Manual Deploy (Tanpa GitHub Actions)

Jika perlu deploy manual langsung dari VPS:

```bash
# 1. SSH ke VPS
ssh -i ~/.ssh/github_actions_wch -p 3209 deploy@<VPS_IP>

# 2. Login ke GHCR
echo "<GHCR_TOKEN>" | docker login ghcr.io -u nzf210 --password-stdin

# 3. Masuk ke direktori staging/production
cd /opt/wch-staging          # atau /opt/wch-production

# 4. Pull image terbaru
IMAGE_TAG="stg-be-v1.0.0" docker compose \
  --env-file .env.staging \
  -f docker-compose.yml \
  -f docker-compose.staging.yml \
  pull

# 5. Jalankan semua service
COMPOSE_PROJECT_NAME=wch-stg \
IMAGE_TAG="stg-be-v1.0.0" docker compose \
  --env-file .env.staging \
  -f docker-compose.yml \
  -f docker-compose.staging.yml \
  up -d
```

> Ganti `stg` → `prod`, `staging` → `production`, dan `wch-stg` → `wch-prod` untuk production.

> `docker-compose.yml` dan `docker-compose.staging.yml` di-copy otomatis oleh GitHub Actions via SCP saat deploy. Jika belum ada, copy manual:
> ```bash
> scp -i ~/.ssh/github_actions_wch -P 3209 \
>   docker-compose.yml docker-compose.staging.yml \
>   deploy@<VPS_IP>:/opt/wch-staging/
> ```

---

## Struktur File

```
infra/deploy/
├── setup-vps.sh        ← Script setup VPS baru (jalankan sekali)
├── nginx-staging.conf  ← Nginx config untuk staging
├── nginx-prod.conf     ← Nginx config untuk production
└── README.md           ← Dokumen ini
```
