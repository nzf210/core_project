# Deployment Runbook — WCH Platform

Panduan deployment production untuk WCH Platform.

## Prerequisites

### Server Requirements
- **OS:** Ubuntu 22.04 LTS atau lebih baru
- **RAM:** Minimum 8GB (recommended 16GB untuk production)
- **CPU:** 4 cores minimum
- **Storage:** 100GB SSD
- **Network:** Port 80, 443 (public), 5433, 6381, 8000-8210 (internal)

### Software Requirements
```bash
# Docker & Docker Compose
docker --version  # 24.0+
docker compose version  # 2.20+

# PostgreSQL Client (untuk management)
psql --version  # 16+

# Redis CLI (untuk debugging)
redis-cli --version

# Git
git --version
```

## Initial Server Setup

### 1. Clone Repository

```bash
# Clone ke server production
git clone https://github.com/your-org/wch-platform.git /opt/wch-platform
cd /opt/wch-platform

# Checkout tag production (JANGAN gunakan branch main!)
git fetch --tags
git checkout tags/v1.0.0  # ganti dengan versi stable terbaru
```

### 2. Environment Configuration

```bash
# Copy template staging (untuk production/VPS)
cp .env.staging.example .env.staging

# Edit dengan nilai production
nano .env.staging
```

**Critical Environment Variables:**

```bash
# Database (nama key sesuai .env.staging.example)
DB_HOST=postgres
DB_PORT=5432
DB_USER=wch_admin
DB_PASSWORD=STRONG_PASSWORD
DB_NAME=wch_platform

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=STRONG_REDIS_PASSWORD

# JWT Secret (WAJIB 32+ karakter)
JWT_SECRET=$(openssl rand -base64 32)

# Encryption Key (WAJIB 32 bytes untuk AES-256)
ENCRYPTION_KEY=$(openssl rand -base64 32)

# Xendit (Global Fallback)
XENDIT_API_KEY=xnd_production_...
XENDIT_WEBHOOK_TOKEN=$(openssl rand -hex 32)

# N8N Encryption Key
N8N_ENCRYPTION_KEY=$(openssl rand -base64 32)

# App Environment
APP_ENV=production

# Domain (untuk Nginx/SSL config)
DOMAIN=app.wch-platform.com
```

**⚠️ Security Checklist:**
- [ ] `JWT_SECRET` minimal 32 karakter random
- [ ] `ENCRYPTION_KEY` tepat 32 bytes (base64)
- [ ] PostgreSQL password kompleks (min 20 karakter)
- [ ] Redis password set
- [ ] `.env` file permission 600 (`chmod 600 .env`)
- [ ] TIDAK commit `.env` ke git

### 3. SSL/TLS Setup

```bash
# Install Certbot (Let's Encrypt)
sudo apt update
sudo apt install certbot python3-certbot-nginx

# Generate certificate
sudo certbot certonly --nginx -d app.wch-platform.com

# Certificate path
# /etc/letsencrypt/live/app.wch-platform.com/fullchain.pem
# /etc/letsencrypt/live/app.wch-platform.com/privkey.pem
```

**Update Nginx Config:**
```nginx
server {
    listen 443 ssl http2;
    server_name app.wch-platform.com;

    ssl_certificate /etc/letsencrypt/live/app.wch-platform.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/app.wch-platform.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 4. Database Initialization

```bash
# Start PostgreSQL container
docker compose up -d postgres

# Wait untuk ready
sleep 10

# Verify connection
docker compose exec postgres psql -U wch_admin -d wch_platform -c "SELECT version();"

# Migrations run automatically saat service start
# Atau manual:
# psql -h localhost -p 5433 -U wch_admin -d wch_platform < shared/migrations/000001_initial.up.sql
```

### 5. Redis Setup

```bash
# Start Redis
docker compose up -d redis

# Test connection
docker compose exec redis redis-cli -a $REDIS_PASSWORD PING
# Expected: PONG

# Set memory policy (evict LRU saat full)
docker compose exec redis redis-cli CONFIG SET maxmemory-policy allkeys-lru
docker compose exec redis redis-cli CONFIG SET maxmemory 2gb
```

## Deployment Steps

### Production Deployment (Docker)

```bash
# 1. Pull latest images
docker compose pull

# 2. Build services
docker compose build --no-cache

# 3. Start infrastructure (DB, Redis, N8N)
docker compose up -d postgres redis n8n-main n8n-worker

# 4. Wait untuk ready
sleep 30

# 5. Start backend services
docker compose up -d api-gateway auth-service billing-service \
  wa-gateway wa-cloud-api ai-gateway notification-service \
  subscription-worker umkm-accounting umkm-chatbot campaign-api

# 6. Verify semua service running
docker compose ps

# 7. Check logs untuk errors
docker compose logs -f --tail=100

# 8. Start frontend
docker compose up -d umkm-web campaign-web superadmin-web

# 9. Start observability (Grafana, Loki, Promtail)
docker compose up -d grafana loki promtail
```

### Health Check

```bash
# API Gateway
curl http://localhost:8000/health
# Expected: {"status":"ok"}

# Auth Service
curl http://localhost:8001/health

# Billing Service
curl http://localhost:8003/health

# Check database connection
docker compose exec postgres pg_isready

# Check Redis
docker compose exec redis redis-cli -a $REDIS_PASSWORD PING
```

### Service Status Matrix

| Service | Port | Health Endpoint | Expected Response |
|:--------|:-----|:---------------|:------------------|
| API Gateway | 8000 | `/health` | `{"status":"ok"}` |
| Auth Service | 8001 | `/health` | `{"status":"ok"}` |
| AI Gateway | 8002 | `/health` | `{"status":"ok"}` |
| Billing Service | 8003 | `/health` | `{"status":"ok"}` |
| WA Gateway | 8202 | `/status` | `{"success":true}` |
| WA Cloud API | 8210 | `/health` | `{"status":"ok"}` |
| UMKM Accounting | 8201 | `/health` | `{"status":"ok"}` |
| Campaign API | 9002 | `/health` | `{"status":"ok"}` |

## Rollback Procedure

```bash
# 1. Identify previous stable version
git tag -l

# 2. Checkout previous version
git checkout tags/v0.9.5

# 3. Stop current services
docker compose down

# 4. Rebuild dengan version lama
docker compose build

# 5. Start services
docker compose up -d

# 6. Verify
docker compose ps
docker compose logs -f
```

## Database Backup & Restore

### Automated Backup (Daily Cron)

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /opt/wch-platform/scripts/backup-db.sh
```

**Backup Script (`scripts/backup-db.sh`):**
```bash
#!/bin/bash
BACKUP_DIR="/opt/backups/wch-platform"
DATE=$(date +%Y%m%d_%H%M%S)
FILENAME="wch_platform_$DATE.sql.gz"

mkdir -p $BACKUP_DIR

docker compose exec -T postgres pg_dump -U wch_admin wch_platform | gzip > "$BACKUP_DIR/$FILENAME"

# Keep only last 30 days
find $BACKUP_DIR -name "wch_platform_*.sql.gz" -mtime +30 -delete

echo "Backup completed: $FILENAME"
```

### Manual Backup

```bash
# Full backup
docker compose exec postgres pg_dump -U wch_admin wch_platform > backup.sql

# Compressed backup
docker compose exec -T postgres pg_dump -U wch_admin wch_platform | gzip > backup.sql.gz

# Schema only
docker compose exec postgres pg_dump -U wch_admin --schema-only wch_platform > schema.sql

# Data only
docker compose exec postgres pg_dump -U wch_admin --data-only wch_platform > data.sql
```

### Restore from Backup

```bash
# 1. Stop all services (CRITICAL!)
docker compose stop api-gateway auth-service billing-service \
  wa-gateway umkm-accounting umkm-chatbot campaign-api

# 2. Drop existing database
docker compose exec postgres psql -U wch_admin -c "DROP DATABASE wch_platform;"

# 3. Create fresh database
docker compose exec postgres psql -U wch_admin -c "CREATE DATABASE wch_platform;"

# 4. Restore dari backup
gunzip -c backup.sql.gz | docker compose exec -T postgres psql -U wch_admin -d wch_platform

# 5. Verify restore
docker compose exec postgres psql -U wch_admin -d wch_platform -c "SELECT COUNT(*) FROM tenants;"

# 6. Restart services
docker compose up -d
```

## Monitoring & Alerting

### Grafana Setup

```bash
# Access Grafana
http://your-server:3001

# Default credentials
Username: admin
Password: admin (ganti saat first login)

# Import dashboards
# 1. Go to Dashboards → Import
# 2. Upload dari infra/observability/grafana/dashboards/*.json
```

**Key Metrics to Monitor:**
- **CPU Usage** per service (target: <70%)
- **Memory Usage** per service (target: <80%)
- **Request Rate** (RPS) per endpoint
- **Error Rate** (target: <1%)
- **Response Time** p95 (target: <200ms)
- **Database Connections** (max pool size tracking)
- **Redis Memory** (target: <80% maxmemory)

### Loki (Log Aggregation)

```bash
# Query logs via Grafana Explore
# Or CLI:
docker compose exec loki logcli query '{service="api-gateway"}' --limit=100
```

### Alerting Rules (Prometheus)

Located at `infra/observability/prometheus/alerts.yml`:

**Critical Alerts:**
- Service down >5 minutes
- Error rate >5% sustained 5 minutes
- Database connection pool exhausted
- Redis memory >90%
- Disk space <10%

**Alert Channels:**
- Telegram: Configure `TELEGRAM_ALERT_CHAT_ID`
- Email: Configure SMTP in Grafana
- PagerDuty: (optional) via webhook

## Security Hardening

### 1. Firewall Configuration (UFW)

```bash
# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Block direct access to services (only via Nginx)
sudo ufw deny 8000:8210/tcp

# Allow PostgreSQL from localhost only (implicit via docker network)
# Allow Redis from localhost only (implicit via docker network)

# Enable firewall
sudo ufw enable
sudo ufw status
```

### 2. Fail2Ban (Brute Force Protection)

```bash
# Install
sudo apt install fail2ban

# Configure
sudo nano /etc/fail2ban/jail.local
```

**jail.local:**
```ini
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true

[nginx-limit-req]
enabled = true
filter = nginx-limit-req
logpath = /var/log/nginx/error.log
```

### 3. Docker Security

```bash
# Run containers as non-root user
# Already configured in Dockerfile with USER directive

# Enable Docker Content Trust (image verification)
export DOCKER_CONTENT_TRUST=1

# Scan images for vulnerabilities
docker scan wch-platform-api-gateway:latest
```

### 4. Rate Limiting (Nginx)

```nginx
# Limit request rate
limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;
limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;

server {
    location /auth/login {
        limit_req zone=login burst=10 nodelay;
    }
    
    location /api/ {
        limit_req zone=api burst=50 nodelay;
    }
}
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker compose logs <service-name> --tail=100

# Common issues:
# 1. Port already in use
sudo netstat -tulpn | grep :8000

# 2. Database connection failed
docker compose exec postgres pg_isready

# 3. Environment variable missing
docker compose config | grep -i jwt_secret
```

### High Memory Usage

```bash
# Check memory per container
docker stats

# Identify memory hog
docker stats --no-stream | sort -k4 -h

# Restart service to clear memory leak
docker compose restart <service-name>
```

### Database Performance Issues

```bash
# Check slow queries
docker compose exec postgres psql -U wch_admin -d wch_platform

# Enable slow query log
ALTER SYSTEM SET log_min_duration_statement = 1000; -- 1 second
SELECT pg_reload_conf();

# View active queries
SELECT pid, now() - query_start as duration, query 
FROM pg_stat_activity 
WHERE state = 'active' 
ORDER BY duration DESC;

# Kill long-running query
SELECT pg_terminate_backend(pid);
```

### Redis Out of Memory

```bash
# Check memory usage
docker compose exec redis redis-cli -a $REDIS_PASSWORD INFO memory

# Clear cache (DANGEROUS in production!)
# docker compose exec redis redis-cli -a $REDIS_PASSWORD FLUSHALL

# Better: increase maxmemory
docker compose exec redis redis-cli CONFIG SET maxmemory 4gb
```

### Webhook Not Received (Xendit)

**Debug Steps:**
1. Cek webhook URL di Xendit dashboard
2. Test dengan Xendit webhook simulator
3. Cek API Gateway logs:
   ```bash
   docker compose logs api-gateway | grep webhook
   ```
4. Verify signature token:
   ```bash
   docker compose exec postgres psql -U wch_admin -d wch_platform \
     -c "SELECT xendit_webhook_token FROM tenants WHERE id = '<tenantId>';"
   ```

## Scaling

### Horizontal Scaling (N8N Workers)

```bash
# Scale N8N workers untuk high load
docker compose up -d --scale n8n-worker=5

# Verify
docker compose ps | grep n8n-worker
```

### Vertical Scaling (Resource Limits)

Edit `docker-compose.yml`:
```yaml
services:
  api-gateway:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### Load Balancing (Multiple Instances)

```bash
# Deploy ke multiple servers
# Server 1: API Gateway + Auth + Billing
# Server 2: UMKM services + WA Gateway
# Server 3: Campaign + AI Gateway

# Use Nginx upstream untuk load balancing
upstream api_backend {
    server server1.internal:8000;
    server server2.internal:8000;
    server server3.internal:8000;
}
```

## Maintenance Window

### Planned Downtime Procedure

```bash
# 1. Notify users (24 hours advance)
# Kirim notifikasi via Telegram/Email

# 2. Enable maintenance mode (503 response)
# Update Nginx config:
return 503 "Platform sedang maintenance. Kembali dalam 2 jam.";

# 3. Stop services
docker compose stop

# 4. Perform maintenance (DB upgrade, migration, etc)

# 5. Start services
docker compose up -d

# 6. Health check
./scripts/health-check.sh

# 7. Disable maintenance mode
# Revert Nginx config

# 8. Notify users (maintenance complete)
```

## Emergency Contacts

```
Primary On-Call: +62-xxx-xxx-xxxx
Secondary On-Call: +62-yyy-yyy-yyyy
DevOps Lead: devops@wch-platform.com
Database Admin: dba@wch-platform.com

Escalation Matrix:
  L1 (0-15 min): On-call engineer
  L2 (15-30 min): DevOps lead
  L3 (30-60 min): CTO
```

## Post-Deployment Checklist

- [ ] All services health check passed
- [ ] Database migrations completed
- [ ] SSL certificate valid (check expiry)
- [ ] Monitoring dashboards accessible
- [ ] Alert rules tested
- [ ] Backup script running (check cron)
- [ ] Log rotation configured
- [ ] Documentation updated
- [ ] Team notified
- [ ] Smoke test critical flows:
  - [ ] User registration + OTP
  - [ ] Login (username & phone)
  - [ ] Create transaction
  - [ ] Payment webhook (test mode)
  - [ ] WhatsApp send message
  - [ ] AI Chatbot response
