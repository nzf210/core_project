# WCH Platform — Staging Deployment Guide

## Arsitektur Traffic Staging

```
Internet
  → Cloudflare DNS (*.umkmai.id → Tunnel CNAME)
    → Cloudflare Tunnel (cloudflared di VPS)
      → Host Nginx (port 80) ← PERLU INSTALL
        → Docker containers (port 21000, 22001–22003, 23001, 23678)
```

## Cloudflare Tunnel Config

File: `/etc/cloudflared/config.yml` di VPS staging (157.15.40.27)

### Config Aktif Saat Ini

```yaml
tunnel: 810694f9-cd2e-4732-8aac-5e16d7e9d379
credentials-file: /etc/cloudflared/810694f9-cd2e-4732-8aac-5e16d7e9d379.json

ingress:
  - hostname: stg-api.umkmai.id
    service: http://localhost:21000      # api-gateway langsung
  - hostname: stg-grf.umkmai.id
    service: http://localhost:23001      # grafana langsung
  - hostname: stg-n8n.umkmai.id
    service: http://localhost:23678      # n8n langsung
  - service: http_status:404
```

### Config Setelah Nginx Dipasang

Semua hostname diarahkan ke Nginx (port 80). Nginx yang forward ke Docker port.
File tersedia di: `infra/nginx/cloudflare-tunnel-update.yml`

```yaml
tunnel: 810694f9-cd2e-4732-8aac-5e16d7e9d379
credentials-file: /etc/cloudflared/810694f9-cd2e-4732-8aac-5e16d7e9d379.json

ingress:
  - hostname: stg-api.umkmai.id
    service: http://localhost:80
    originRequest:
      httpHostHeader: stg-api.umkmai.id
  - hostname: stg-app.umkmai.id
    service: http://localhost:80
    originRequest:
      httpHostHeader: stg-app.umkmai.id
  - hostname: stg-admin.umkmai.id
    service: http://localhost:80
    originRequest:
      httpHostHeader: stg-admin.umkmai.id
  - hostname: stg-campaign.umkmai.id
    service: http://localhost:80
    originRequest:
      httpHostHeader: stg-campaign.umkmai.id
  - hostname: stg-grf.umkmai.id
    service: http://localhost:80
    originRequest:
      httpHostHeader: stg-grf.umkmai.id
  - hostname: stg-n8n.umkmai.id
    service: http://localhost:80
    originRequest:
      httpHostHeader: stg-n8n.umkmai.id
  - service: http_status:404
```

## Cloudflare DNS Records

Semua subdomain staging sudah terdaftar di Cloudflare dashboard sebagai CNAME
ke tunnel (nilai: `810694f9-cd2e-4732-8aac-5e16d7e9d379.cfargotunnel.com`):

| Subdomain | Status | Target Docker Port |
|---|---|---|
| stg-api.umkmai.id | ✅ Aktif | 21000 (api-gateway) |
| stg-grf.umkmai.id | ✅ Aktif | 23001 (grafana) |
| stg-n8n.umkmai.id | ✅ Aktif | 23678 (n8n) |
| stg-app.umkmai.id | ✅ Aktif (DNS ada, container belum jalan) | 22001 (umkm-frontend) |
| stg-admin.umkmai.id | ✅ Aktif (DNS ada, container belum jalan) | 22002 (superadmin-frontend) |
| stg-campaign.umkmai.id | ✅ Aktif (DNS ada, container belum jalan) | 22003 (campaign-frontend) |

## Status Container di VPS (2026-08-22)

| Container | Status | Port |
|---|---|---|
| wch-stg-api-gateway | ✅ Up | 21000 |
| wch-stg-auth-service | ✅ Up | internal |
| wch-stg-ai-gateway | ✅ Up | internal |
| wch-stg-billing-service | ✅ Up | internal |
| wch-stg-umkm-accounting | ✅ Up | internal |
| wch-stg-umkm-chatbot | ✅ Up | internal |
| wch-stg-umkm-automation | ✅ Up | internal |
| wch-stg-umkm-business | ✅ Up | internal |
| wch-stg-wa-gateway | ✅ Up | internal |
| wch-stg-wa-cloud-api | ✅ Up | internal |
| wch-stg-campaign-api | ✅ Up | internal |
| wch-stg-notification-service | ✅ Up | internal |
| wch-stg-subscription-worker | ✅ Up | internal |
| wch-stg-redis | ✅ Up (healthy) | 20631 |
| wch-stg-pgbouncer | ✅ Up (healthy) | 20433 |
| wch-stg-postgres | ✅ Up (healthy) | internal |
| wch-stg-n8n-main-test | ✅ Up | 23678 |
| wch-stg-grafana | ✅ Up | 23001 |
| wch-stg-prometheus | ✅ Up | internal |
| wch-stg-loki | ✅ Up | internal |
| wch-stg-umkm-frontend | ❌ Belum jalan | 22001 |
| wch-stg-superadmin-frontend | ❌ Belum jalan | 22002 |
| wch-stg-campaign-frontend | ❌ Belum jalan | 22003 |

## Masalah Frontend: Image Perlu Build via CI/CD

Docker compose mendefinisikan frontend dengan `build: context: ./frontend/umkm-web` — VPS perlu
source code untuk build. VPS staging hanya punya file compose, bukan full repo.

**Solusi yang direkomendasikan:** Build frontend image di GitHub Actions, push ke registry,
update docker-compose agar pakai `image:` bukan `build:`.

Contoh perubahan di `docker-compose.yml`:
```yaml
umkm-frontend:
  image: ghcr.io/nzf210/core_project-umkm-frontend:latest  # ganti build dengan image
  container_name: wch-umkm-frontend
  ...
```

## Langkah Install Nginx di VPS

Perlu login sebagai `administrator` (user `deploy` hanya bisa sudo docker):

```bash
ssh administrator@157.15.40.27 -p 3209
sudo bash /opt/wch-staging/infra/nginx/install-staging.sh
sudo bash /opt/wch-staging/infra/nginx/update-cloudflare-tunnel.sh
```

## Cloudflare Tunnel Credentials

**PENTING:** File `/etc/cloudflared/810694f9-...json` sempat hilang (2026-08-21) menyebabkan
tunnel crash loop. File ini berisi credentials untuk autentikasi tunnel ke Cloudflare.

Jika hilang lagi, restore dengan:
```bash
# Login ulang ke Cloudflare
cloudflared tunnel login
# Re-generate credentials file
cloudflared tunnel token --creds-file /etc/cloudflared/810694f9-cd2e-4732-8aac-5e16d7e9d379.json \
  810694f9-cd2e-4732-8aac-5e16d7e9d379
sudo systemctl restart cloudflared
```
