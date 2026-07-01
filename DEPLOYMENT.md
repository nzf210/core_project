# WCH Platform - Staging Deployment Guide

## Prerequisites

- VPS dengan minimal 4GB RAM, 2 CPU cores, 40GB disk
- Ubuntu 20.04 LTS atau lebih baru
- Domain yang sudah di-point ke VPS IP
- GitHub repository access

---

## 1. VPS Initial Setup

### SSH ke VPS

```bash
ssh root@your-vps-ip
```

### Run Setup Script

```bash
# Clone repo dulu (setup repo access/SSH key sebelumnya)
mkdir -p /opt/wch-platform
cd /opt/wch-platform
git clone <your-repo-url> .

# Run setup script
sudo bash scripts/staging-setup.sh
```

Script akan install:
- Docker & Docker Compose
- Nginx
- Certbot (untuk SSL)
- Git, Make, Curl, Wget
- Setup direktori `/opt/wch-platform` dan `/backup/wch`

---

## 2. Generate Production Secrets

**PENTING:** JWT_SECRET dan ENCRYPTION_KEY harus TEPAT 32 karakter. N8N_ENCRYPTION_KEY harus 64 karakter hex.

```bash
# JWT Secret (32 karakter, trim ke 32)
openssl rand -base64 32 | head -c 32

# Encryption Key (32 karakter, trim ke 32)
openssl rand -base64 32 | head -c 32

# N8N Encryption Key (64 karakter hex)
openssl rand -hex 32

# Grafana Password
openssl rand -base64 16

# Database Password
openssl rand -base64 24

# Redis Password
openssl rand -base64 24
```

---

## 3. Configure Environment

```bash
cd /opt/wch-platform
cp .env.staging.example .env.staging
nano .env.staging
```

**WAJIB diisi (ikuti checklist di .env.staging.example):**

```bash
# Security (TEPAT 32 karakter!)
APP_ENV=production
JWT_SECRET=<generated-32-char-secret>
ENCRYPTION_KEY=<generated-32-char-key>

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=wch_admin
DB_PASSWORD=<strong-password>
DB_NAME=wch_platform

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=<strong-password>

# N8N (64 karakter hex!)
N8N_ENCRYPTION_KEY=<generated-64-char-hex>
N8N_DB_PASSWORD=<strong-password>
N8N_ADMIN_PASSWORD=<strong-password>

# Monitoring
GRAFANA_ADMIN_PASSWORD=<strong-password>

# API Keys
MINIMAX_API_KEY=<anthropic-api-key>
GEMINI_API_KEY=<google-gemini-key>
XENDIT_API_KEY=<xendit-staging-key>
TELEGRAM_BOT_TOKEN=<telegram-bot-token>
WA_CLOUD_API_TOKEN=<meta-cloud-api-token>
# ... (lengkap di .env.staging.example)
```

**Validasi sebelum deploy:**

```bash
# Cek panjang JWT_SECRET dan ENCRYPTION_KEY (harus 32)
echo -n "$JWT_SECRET" | wc -c
echo -n "$ENCRYPTION_KEY" | wc -c

# Cek panjang N8N_ENCRYPTION_KEY (harus 64)
echo -n "$N8N_ENCRYPTION_KEY" | wc -c
```

---

## 4. Setup SSL Certificate

```bash
# Setup Let's Encrypt SSL
sudo certbot --nginx -d api.yourdomain.com

# Certbot akan auto-renew. Verify:
sudo certbot renew --dry-run
```

---

## 5. Deploy via CI/CD (Recommended)

### Setup GitHub Secrets

Di GitHub repo → Settings → Secrets and variables → Actions:

```
VPS_HOST=your-vps-ip-or-domain
VPS_USERNAME=root
VPS_SSH_KEY=<your-private-ssh-key>
```

### Trigger Deploy

```bash
# Push ke main branch akan auto-deploy
git push origin main
```

Workflow akan:
1. Run tests
2. Build services
3. Deploy ke VPS
4. Verify health checks

---

## 6. Manual Deploy (Alternative)

```bash
ssh root@your-vps
cd /opt/wch-platform

# Pull latest
git pull origin main

# Deploy
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d --build

# Wait 30s for services to start
sleep 30

# Check status
docker-compose ps
curl http://localhost:8010/health
```

---

## 7. Post-Deploy Verification

### Check All Services Running

```bash
docker-compose ps
# All services should show "Up" or "Up (healthy)"
```

### Test API Gateway

```bash
curl https://api.yourdomain.com/health
# Should return: {"status":"healthy"}
```

### Check Logs

```bash
# API Gateway
docker-compose logs -f api-gateway

# Auth Service
docker-compose logs -f auth-service

# All services
docker-compose logs -f
```

### Access Grafana

```bash
# Via SSH tunnel (recommended for staging)
ssh -L 3001:localhost:3001 root@your-vps

# Then open browser: http://localhost:3001
# Login: admin / <GRAFANA_ADMIN_PASSWORD>
```

### Verify Database

```bash
docker exec -it wch-postgres psql -U wch_admin -d wch_platform
\dt  # List tables
SELECT * FROM schema_migrations ORDER BY version DESC LIMIT 5;
\q
```

### Setup WhatsApp Gateway

```bash
# Generate QR code for system tenant
curl http://localhost:8202/wa/qr?tenant_id=system

# Scan dengan WhatsApp dan tunggu "Connected to WhatsApp"
curl http://localhost:8202/wa/status
# Should return: {"status":"connected","jid":"..."}
```

---

## 8. Monitoring & Alerts

### Grafana Dashboards

1. **WCH Services Overview** - Health, request rate, error rate
2. **Database Performance** - Queries, connections, slow queries
3. **Redis Performance** - Memory, hit rate, connections
4. **System Resources** - CPU, Memory, Disk

### Prometheus Alerts

Alert rules configured di `infra/observability/prometheus/alerts.yml`:

- ServiceDown (critical)
- HighErrorRate (warning)
- PostgresDown (critical)
- RedisDown (critical)
- HighResponseTime (warning)
- LowDiskSpace (warning)

**TODO:** Setup Alertmanager untuk notifikasi (Slack/Telegram/Email).

---

## 9. Backup & Recovery

### Manual Backup

```bash
bash scripts/backup.sh /backup/wch
```

### Automated Daily Backup

Setup script sudah menambahkan cron:

```bash
# Check crontab
crontab -l
# Should show: 0 2 * * * /opt/wch-platform/scripts/backup.sh /backup/wch
```

### Restore from Backup

```bash
cd /backup/wch
tar -xzf backup_wch_YYYYMMDD_HHMM.tar.gz

# Restore main DB
docker exec -i wch-postgres pg_restore -U wch_admin -d wch_platform -c < wch_platform.dump

# Restore N8N DB
docker exec -i wch-postgres pg_restore -U wch_n8n -d wch_n8n_db -c < wch_n8n_db.dump

# Restore .env
cp .env.backup /opt/wch-platform/.env

# Restart services
docker-compose restart
```

---

## 10. Troubleshooting

### Service Won't Start

```bash
# Check logs
docker-compose logs <service-name>

# Check health
docker inspect <container-name> | grep Health -A 10

# Restart specific service
docker-compose restart <service-name>
```

### High Memory Usage

```bash
# Check container stats
docker stats

# Check what's using memory
free -h
df -h
```

### Database Connection Issues

```bash
# Check Postgres is running
docker exec wch-postgres pg_isready -U wch_admin

# Check connection from service
docker exec wch-auth-service ping -c 3 postgres

# Check logs
docker-compose logs postgres
```

### Redis Connection Issues

```bash
# Test Redis
docker exec wch-redis redis-cli ping
# Should return: PONG

# Check memory
docker exec wch-redis redis-cli INFO memory
```

### Roll Back Deployment

```bash
cd /opt/wch-platform

# Revert to previous commit
git log --oneline -5
git checkout <previous-commit-hash>

# Rebuild
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d --build
```

---

## 11. Scaling Considerations

### Horizontal Scaling

Untuk scale services tertentu:

```bash
# Scale N8N workers
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d --scale n8n-worker=3

# Scale frontend
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d --scale umkm-frontend=2
```

### Vertical Scaling

Edit `docker-compose.staging.yml` resource limits sesuai kebutuhan:

```yaml
deploy:
  resources:
    limits:
      memory: 1G  # Increase dari 512M
      cpus: '1.0' # Increase dari 0.5
```

---

## 12. Security Checklist

- [ ] JWT_SECRET & ENCRYPTION_KEY adalah random 32-byte
- [ ] Database & Redis password kuat
- [ ] SSL certificate active (HTTPS)
- [ ] Port internal services (8001-8210) hanya bind ke 127.0.0.1
- [ ] Firewall configured (ufw allow 80, 443, 22)
- [ ] SSH key-based auth (disable password login)
- [ ] Regular backups running
- [ ] Grafana admin password changed dari default
- [ ] .env tidak ter-commit ke git
- [ ] Docker images up-to-date

---

## 13. Performance Tuning

### Postgres

```sql
-- Check slow queries
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC LIMIT 10;

-- Check connection pool
SELECT count(*) FROM pg_stat_activity;
```

### Redis

```bash
# Monitor real-time
docker exec wch-redis redis-cli MONITOR

# Check hit rate
docker exec wch-redis redis-cli INFO stats | grep hit
```

---

## Support

Jika ada issue:

1. Check logs: `docker-compose logs -f <service>`
2. Check Grafana dashboards
3. Check `/backup/wch` untuk restore point
4. Refer ke `docs/` untuk troubleshooting spesifik

---

## Next Steps → Production

Sebelum production:

1. Setup Alertmanager dengan notification channel
2. Setup offsite backup (S3/GCS)
3. Load testing
4. Security audit
5. Setup CDN untuk frontend
6. Multi-region database replication (jika perlu)
