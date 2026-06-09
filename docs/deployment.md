# WCH Platform — Deployment & Infrastructure Blueprint

Dokumen ini mendefinisikan arsitektur deployment, panduan instalasi lokal, strategi kontainerisasi, dan tata kelola infrastruktur untuk **WCH Multi-Product Platform**.

---

## 🏗️ 1. Arsitektur Infrastruktur Produksi

Untuk efisiensi biaya awal dan skalabilitas jangka menengah, kami merekomendasikan arsitektur berbasis **Docker Compose Multi-Container** pada Virtual Private Server (VPS) berspesifikasi tinggi (min. 4 vCPU, 8GB RAM) seperti DigitalOcean, AWS EC2, atau Alibaba Cloud.

```
                  ┌──────────────────────────────────────────┐
                  │            Internet (HTTPS)              │
                  └───────────────────┬──────────────────────┘
                                      │
                                      ▼
                  ┌──────────────────────────────────────────┐
                  │          Nginx Reverse Proxy             │
                  │       (SSL Let's Encrypt / Certbot)      │
                  └───────────────────┬──────────────────────┘
                                      │ (Internal Bridge)
           ┌──────────────────────────┼──────────────────────────┐
           ▼                          ▼                          ▼
┌──────────────────────┐   ┌──────────────────────┐   ┌──────────────────────┐
│  frontend-gateway    │   │  services-gateway    │   │     apps-gateway     │
│   (Port 80/443)      │   │   (Port 8000-8005)   │   │   (Port 9000-9003)   │
├──────────────────────┤   ├──────────────────────┤   ├──────────────────────┤
│ - umkm-web (3001)    │   │ - auth-service (8000)│   │ - umkm-app (9001)    │
│ - campaign-web (3002)│   │ - billing-serv (8002)│   │ - campaign-app (9002)│
│ - admin-web (3003)   │   │ - ai-gateway (8003)  │   └──────────────────────┘
└──────────────────────┘   │ - notify-serv (8004) │
                           │ - n8n workflow (5678)│
                           └──────────────────────┘
                                      │
           ┌──────────────────────────┴──────────────────────────┐
           ▼                                                     ▼
┌──────────────────────────────────────┐              ┌──────────────────────┐
│        PostgreSQL Database           │              │     Redis Cluster    │
│ (Isolated Database Schemas per App)  │              │ (Cache, Pub/Sub,     │
└──────────────────────────────────────┘              │  AI Semantic Cache)  │
                                                      └──────────────────────┘
```

---

## 💻 2. Panduan Instalasi Lokal (Local Development)

### Prasyarat
Pastikan mesin lokal Anda telah terinstal:
*   Docker & Docker Compose (v2.x+)
*   Node.js (v18.x+) & npm / pnpm
*   Git

### Langkah-Langkah Menjalankan Proyek:

1.  **Clone Repositori & Masuk ke Folder Project**:
    ```bash
    git clone <repository_url> core_project
    cd core_project
    ```

2.  **Konfigurasi Environment Variable**:
    Salin file contoh konfigurasi dan sesuaikan isinya:
    ```bash
    cp infra/deploy/.env.example .env
    ```

3.  **Jalankan Infrastruktur Dasar (DB & Redis) via Docker Compose**:
    ```bash
    docker compose -f infra/docker/docker-compose.yml up -d
    ```
    *Perintah ini akan menjalankan PostgreSQL, Redis, dan n8n.*

4.  **Inisialisasi Database (Migrasi Skema)**:
    ```bash
    npm run db:migrate
    ```

5.  **Jalankan Aplikasi dalam Mode Development**:
    ```bash
    npm run dev
    ```
    *Ini akan memicu dev server untuk semua microservices dan frontends secara paralel menggunakan Turbo/pnpm workspaces.*

---

## 🔒 3. Matriks Konfigurasi Environment Variables (`.env`)

Berikut adalah parameter konfigurasi wajib yang harus diatur dalam file `.env` sistem:

### 🌐 Core & Shared Configuration
```ini
NODE_ENV=development
JWT_SECRET=super_secret_key_minimum_32_characters_long
ENCRYPTION_KEY=aes_256_key_for_encrypting_sensitive_data
PLATFORM_DOMAIN=wchplatform.com
```

### 🗄️ Database & Cache Settings
```ini
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=wch_admin
POSTGRES_PASSWORD=secure_postgres_password_123
POSTGRES_DB=wch_platform

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=secure_redis_password_123
```

### 🤖 AI API Configurations
```ini
# API Keys untuk AI Gateway
OPENAI_API_KEY=sk-proj-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
GEMINI_API_KEY=AIzaSyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
CLAUDE_API_KEY=sk-ant-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 💳 Payment & Billing Settings
```ini
# Integrasi Payment Gateway (Xendit untuk IDR, Stripe untuk USD)
XENDIT_API_KEY=xnd_development_xxxxxxxxxxxxxxxxxxxxxxxx
STRIPE_API_KEY=sk_test_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
STRIPE_WEBHOOK_SECRET=whsec_xxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 🔔 Notification & Automation
```ini
# WhatsApp Business API / Provider Webhook
WHATSAPP_API_TOKEN=eaagxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
WHATSAPP_PHONE_NUMBER_ID=1234567890

# SMTP Config untuk Email
SMTP_HOST=smtp.mailgun.org
SMTP_PORT=587
SMTP_USER=postmaster@wchplatform.com
SMTP_PASS=smtp_secure_password_abc
```

---

## 🚀 4. Strategi CI/CD Pipeline (GitHub Actions)

Untuk menjamin kelancaran perilisan tanpa downtime (*zero-downtime deployment*), kita menerapkan pipeline CI/CD otomatis sebagai berikut:

```mermaid
flowchart TD
    A[Developer Push ke branch 'main'] --> B{CI Checks}
    B -- Linter Fail / Test Fail --> C[Kirim Notifikasi Error ke Slack/WA]
    B -- Linter & Tests Success --> D[Build Docker Images per Service]
    D --> E[Push Images ke Docker Hub / Private Container Registry]
    E --> F[Trigger CD Deployment Script di Target Server via SSH]
    F --> G[Server Melakukan docker compose pull]
    G --> H[Server Melakukan Rolling Update Tanpa Downtime]
    H --> I[Kirim Notifikasi Rilis Sukses ke Tim Developer]
```

### Script Deployment Produksi Sederhana (`infra/deploy/deploy.sh`):
```bash
#!/bin/bash
set -e

echo "🚀 Memulai proses deployment WCH Platform..."

# 1. Tarik pembaruan kode terbaru dari Git
git pull origin main

# 2. Login ke Registry Kontainer
echo $DOCKER_PASSWORD | docker login -u $DOCKER_USERNAME --password-stdin

# 3. Ambil Docker Image versi terbaru
docker compose -f infra/docker/docker-compose.prod.yml pull

# 4. Jalankan migrasi database
docker compose -f infra/docker/docker-compose.prod.yml run --rm services-gateway npm run db:migrate

# 5. Jalankan kontainer baru dengan metode rolling-restart (Nginx redirect)
docker compose -f infra/docker/docker-compose.prod.yml up -d --remove-orphans

# 6. Bersihkan docker image usang untuk menghemat penyimpanan disk
docker image prune -f

echo "🎉 Platform WCH berhasil diperbarui!"
```

---

## 🛠️ 5. Monitoring & Pemeliharaan (Maintenance)

1.  **Log Management**:
    Semua log kontainer dikirim ke file log host lokal `/var/log/docker/` dan diputar menggunakan `logrotate` untuk mencegah kehabisan ruang disk.
2.  **Uptime Monitoring**:
    Menggunakan tools pihak ketiga gratis seperti **UptimeRobot** atau **Better Uptime** untuk memantau status kesehatan endpoint `/health` pada masing-masing domain/aplikasi.
3.  **Database Backup**:
    Menjadwalkan cron job harian di server untuk mencadangkan database PostgreSQL ke cloud storage terpisah (misalnya AWS S3) secara otomatis:
    ```bash
    0 2 * * * pg_dump -U wch_admin -h localhost wch_platform | gzip > /backups/db_$(date +\%F).sql.gz
    ```
