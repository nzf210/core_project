# Panduan Instalasi Cloudflare Tunnel untuk VPS Staging

## Overview

Cloudflare Tunnel (cloudflared) adalah solusi untuk mengekspos service lokal ke internet **tanpa membuka port inbound** (80/443). Koneksi keluar dari VPS ke Cloudflare, ideal untuk VPS rumahan atau VPS dengan port firewall yang diblokir ISP.

**Setup ini untuk:** VPS staging WCH Platform di `157.15.40.27`

---

## Prerequisites

1. **Akun Cloudflare** dengan domain `umkmai.id` sudah ditambahkan
2. **VPS Access:** SSH ke `deploy@157.15.40.27:3209`
3. **Docker containers** sudah running (port internal: 21000, 23001, 23678)

---

## Langkah 1: Install cloudflared di VPS

SSH ke VPS staging:

```bash
ssh -i ~/.ssh/github_actions_wch -p 3209 deploy@157.15.40.27
```

Install cloudflared:

```bash
# Download cloudflared binary
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb

# Install
sudo dpkg -i cloudflared-linux-amd64.deb

# Verify installation
cloudflared --version
```

---

## Langkah 2: Login dan Buat Tunnel

Login ke Cloudflare (akan buka browser untuk autentikasi):

```bash
cloudflared tunnel login
```

File credentials akan tersimpan di `~/.cloudflared/cert.pem`

Buat tunnel baru dengan nama `wch-staging`:

```bash
cloudflared tunnel create wch-staging
```

Output akan menampilkan:
- **Tunnel ID**: `810694f9-cd2e-4732-8aac-5e16d7e9d379` (simpan ini!)
- **Credentials file**: `~/.cloudflared/810694f9-cd2e-4732-8aac-5e16d7e9d379.json`

---

## Langkah 3: Konfigurasi Tunnel

Buat file konfigurasi tunnel di VPS:

```bash
sudo mkdir -p /etc/cloudflared
sudo nano /etc/cloudflared/config.yml
```

Isi dengan config berikut:

```yaml
tunnel: 810694f9-cd2e-4732-8aac-5e16d7e9d379
credentials-file: /etc/cloudflared/810694f9-cd2e-4732-8aac-5e16d7e9d379.json

ingress:
  - hostname: stg-api.umkmai.id
    service: http://localhost:21000
  - hostname: stg-grf.umkmai.id
    service: http://localhost:23001
  - hostname: stg-n8n.umkmai.id
    service: http://localhost:23678
  - service: http_status:404
```

**Penjelasan:**
- `stg-api.umkmai.id` → API Gateway (port 21000)
- `stg-grf.umkmai.id` → Grafana (port 23001)
- `stg-n8n.umkmai.id` → N8N (port 23678)
- Catch-all → 404 untuk domain tidak terdaftar

Copy credentials file ke `/etc/cloudflared/`:

```bash
sudo cp ~/.cloudflared/810694f9-cd2e-4732-8aac-5e16d7e9d379.json /etc/cloudflared/
sudo chmod 600 /etc/cloudflared/810694f9-cd2e-4732-8aac-5e16d7e9d379.json
```

---

## Langkah 4: Route DNS via Cloudflare

Route setiap domain ke tunnel:

```bash
cloudflared tunnel route dns wch-staging stg-api.umkmai.id
cloudflared tunnel route dns wch-staging stg-grf.umkmai.id
cloudflared tunnel route dns wch-staging stg-n8n.umkmai.id
```

**Atau** bisa manual via Cloudflare Dashboard:
1. Login ke [Cloudflare Dashboard](https://dash.cloudflare.com)
2. Pilih domain `umkmai.id`
3. DNS → Add Record:
   - **Type:** CNAME
   - **Name:** `stg-api`
   - **Target:** `810694f9-cd2e-4732-8aac-5e16d7e9d379.cfargotunnel.com`
   - **Proxy status:** Proxied (orange cloud)
4. Ulangi untuk `stg-grf` dan `stg-n8n`

---

## Langkah 5: Setup Tunnel sebagai Systemd Service

Install service:

```bash
sudo cloudflared service install
```

Start dan enable service:

```bash
sudo systemctl start cloudflared
sudo systemctl enable cloudflared
```

Cek status:

```bash
sudo systemctl status cloudflared
```

Output yang benar:

```
● cloudflared.service - cloudflared
   Loaded: loaded (/etc/systemd/system/cloudflared.service; enabled)
   Active: active (running) since ...
```

---

## Langkah 6: Verifikasi Tunnel Running

Cek log tunnel:

```bash
sudo journalctl -u cloudflared -f
```

Test akses domain dari browser/curl:

```bash
curl -I https://stg-api.umkmai.id/health
# Expected: HTTP/2 200
```

Cek di Cloudflare Dashboard → Traffic → Cloudflare Tunnel:
- Tunnel `wch-staging` harus **HEALTHY** (hijau)
- Lihat traffic metrics real-time

---

## Troubleshooting

### 1. Tunnel status "Disconnected"

```bash
sudo systemctl restart cloudflared
sudo journalctl -u cloudflared -n 50
```

### 2. Domain resolve tapi 502 Bad Gateway

Cek apakah Docker container backend running:

```bash
docker ps | grep stg
curl http://localhost:21000/health  # Direct local test
```

### 3. DNS tidak resolve

```bash
# Cek DNS record
dig stg-api.umkmai.id

# Re-route DNS
cloudflared tunnel route dns wch-staging stg-api.umkmai.id
```

### 4. Restart tunnel

```bash
sudo systemctl restart cloudflared
```

---

## Menambah Domain Baru ke Tunnel

1. Edit config di VPS:

```bash
sudo nano /etc/cloudflared/config.yml
```

Tambahkan ingress baru:

```yaml
ingress:
  - hostname: stg-api.umkmai.id
    service: http://localhost:21000
  - hostname: stg-grf.umkmai.id
    service: http://localhost:23001
  - hostname: stg-n8n.umkmai.id
    service: http://localhost:23678
  - hostname: stg-chatwoot.umkmai.id    # BARU
    service: http://localhost:23000      # BARU
  - service: http_status:404
```

2. Route DNS:

```bash
cloudflared tunnel route dns wch-staging stg-chatwoot.umkmai.id
```

3. Restart tunnel:

```bash
sudo systemctl restart cloudflared
```

---

## Uninstall / Cleanup

Jika ingin hapus tunnel:

```bash
# Stop service
sudo systemctl stop cloudflared
sudo systemctl disable cloudflared

# Uninstall service
sudo cloudflared service uninstall

# Delete tunnel
cloudflared tunnel delete wch-staging

# Remove files
sudo rm -rf /etc/cloudflared
rm -rf ~/.cloudflared
```

---

## Automation Script (Optional)

File ini sudah ada di repo: `infra/deploy/setup-cloudflared.sh`

Usage dari **local machine**:

```bash
# Set VPS credentials
export VPS_HOST=157.15.40.27
export VPS_PORT=3209
export TUNNEL_NAME=wch-staging
export TUNNEL_ID=810694f9-cd2e-4732-8aac-5e16d7e9d379

# Run setup
bash infra/deploy/setup-cloudflared.sh
```

Script akan:
1. SSH ke VPS
2. Install cloudflared
3. Copy config dari `infra/deploy/cloudflared-staging.yml`
4. Setup systemd service
5. Route DNS otomatis

---

## Security Notes

1. **Credentials file** (`810694f9-cd2e-4732-8aac-5e16d7e9d379.json`) adalah **SECRET** — jangan commit ke git!
2. **SSL termination** dihandle Cloudflare — VPS hanya serve HTTP di localhost
3. **Firewall:** Tidak perlu buka port 80/443 — tunnel gunakan port outbound (443/7844)
4. **Rate limiting:** Cloudflare dashboard untuk set rate limit per domain

---

## Reference

- Cloudflare Tunnel Docs: https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/
- VPS Deploy Memory: `~/.claude/projects/-home-syahril-dev-core-project/memory/vps-deploy-setup.md`
- Config files:
  - `infra/deploy/cloudflared-staging.yml` — template config
  - `infra/deploy/nginx-staging-http.conf` — nginx HTTP-only (aktif di VPS)

---

## Status Saat Ini (2026-08-21)

✅ Tunnel `wch-staging` (ID: `810694f9-cd2e-4732-8aac-5e16d7e9d379`) sudah **AKTIF** di VPS staging  
✅ Domain routing:
- `stg-api.umkmai.id` → port 21000 ✅
- `stg-grf.umkmai.id` → port 23001 ✅  
- `stg-n8n.umkmai.id` → port 23678 ✅

Untuk cek status live:
```bash
ssh -i ~/.ssh/github_actions_wch -p 3209 deploy@157.15.40.27 "sudo systemctl status cloudflared"
```
