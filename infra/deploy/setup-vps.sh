#!/bin/bash
# =============================================================================
# Setup VPS untuk WCH Platform
# Jalankan dari LOCAL machine:
#
#   Shared VPS (staging + production di satu server):
#     bash infra/deploy/setup-vps.sh shared
#
#   Dedicated VPS staging saja:
#     bash infra/deploy/setup-vps.sh staging
#
#   Dedicated VPS production saja:
#     bash infra/deploy/setup-vps.sh production
# =============================================================================
set -e

MODE=${1:-}
VPS_USER="deploy"
SSH_KEY="$HOME/.ssh/github_actions_wch"

if [ -z "$MODE" ]; then
    echo "Usage: bash setup-vps.sh [shared|staging|production]"
    exit 1
fi

# Tanya VPS host jika tidak di-set
if [ -z "$VPS_HOST" ]; then
    read -rp "VPS IP/hostname: " VPS_HOST
fi
if [ -z "$VPS_PORT" ]; then
    read -rp "VPS SSH port [3209]: " VPS_PORT
    VPS_PORT=${VPS_PORT:-3209}
fi

SSH="ssh -i $SSH_KEY -p $VPS_PORT $VPS_USER@$VPS_HOST"
SCP="scp -i $SSH_KEY -P $VPS_PORT"

echo ""
echo "=== Target: MODE=$MODE | $VPS_USER@$VPS_HOST:$VPS_PORT ==="
echo ""

# -----------------------------------------------------------------------------
# 1. Buat direktori deploy
# -----------------------------------------------------------------------------
echo "--- [1/6] Buat direktori deploy ---"
if [ "$MODE" = "shared" ] || [ "$MODE" = "staging" ]; then
    $SSH "sudo mkdir -p /opt/wch-staging && sudo chown $VPS_USER:$VPS_USER /opt/wch-staging && echo 'staging dir: OK'"
fi
if [ "$MODE" = "shared" ] || [ "$MODE" = "production" ]; then
    $SSH "sudo mkdir -p /opt/wch-production && sudo chown $VPS_USER:$VPS_USER /opt/wch-production && echo 'production dir: OK'"
fi

# -----------------------------------------------------------------------------
# 2. Setup passwordless sudo untuk docker
# -----------------------------------------------------------------------------
echo "--- [2/6] Setup passwordless sudo untuk docker ---"
# Cek apakah sudo sudah passwordless — kalau sudah, skip (jangan overwrite diri sendiri)
if $SSH "sudo -n true 2>/dev/null"; then
    echo "sudoers: already configured (skipped)"
else
    echo "ERROR: sudo masih butuh password di VPS."
    echo "Jalankan ini dulu dari VPS secara interaktif:"
    echo "  echo -e 'Defaults !requiretty\\ndeploy ALL=(ALL) NOPASSWD: ALL' | sudo tee /etc/sudoers.d/deploy-nopasswd"
    echo "  sudo chmod 440 /etc/sudoers.d/deploy-nopasswd"
    exit 1
fi

# -----------------------------------------------------------------------------
# 3. Kernel tuning untuk 100K concurrent connections
# -----------------------------------------------------------------------------
echo "--- [3/6] Kernel & fd tuning ---"
$SSH "sudo tee /etc/sysctl.d/99-wch-highload.conf > /dev/null << 'EOF'
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15
net.core.netdev_max_backlog = 65535
fs.file-max = 1000000
EOF
sudo sysctl -p /etc/sysctl.d/99-wch-highload.conf && echo 'sysctl: OK'"

$SSH "sudo tee /etc/security/limits.d/99-wch-highload.conf > /dev/null << 'EOF'
* soft nofile 1000000
* hard nofile 1000000
root soft nofile 1000000
root hard nofile 1000000
EOF
echo 'fd limits: OK'"

# -----------------------------------------------------------------------------
# 4. Install Nginx + Certbot
# -----------------------------------------------------------------------------
echo "--- [4/6] Install Nginx + Certbot ---"
$SSH "which nginx > /dev/null 2>&1 && echo 'nginx: already installed' || (sudo apt-get update -qq && sudo apt-get install -y nginx certbot python3-certbot-nginx && echo 'nginx: installed')"

# -----------------------------------------------------------------------------
# 5. Deploy Nginx config
# -----------------------------------------------------------------------------
echo "--- [5/6] Deploy Nginx config ---"


# Hapus default config
$SSH "sudo rm -f /etc/nginx/sites-enabled/default"

if [ "$MODE" = "shared" ] || [ "$MODE" = "staging" ]; then
    $SCP "infra/deploy/nginx-staging.conf" "$VPS_USER@$VPS_HOST:/tmp/wch-nginx-staging.conf"
    $SSH "sudo cp /tmp/wch-nginx-staging.conf /etc/nginx/sites-available/wch-staging.conf && \
          sudo ln -sf /etc/nginx/sites-available/wch-staging.conf /etc/nginx/sites-enabled/wch-staging.conf && \
          echo 'nginx staging config: OK'"
fi

if [ "$MODE" = "shared" ] || [ "$MODE" = "production" ]; then
    $SCP "infra/deploy/nginx-prod.conf" "$VPS_USER@$VPS_HOST:/tmp/wch-nginx-prod.conf"
    $SSH "sudo cp /tmp/wch-nginx-prod.conf /etc/nginx/sites-available/wch-prod.conf && \
          sudo ln -sf /etc/nginx/sites-available/wch-prod.conf /etc/nginx/sites-enabled/wch-prod.conf && \
          echo 'nginx production config: OK'"
fi

# Test dan enable Nginx
$SSH "sudo nginx -t && sudo systemctl enable nginx && sudo systemctl reload nginx && echo 'nginx: running'"

# -----------------------------------------------------------------------------
# 5. Reminder .env dan SSL
# -----------------------------------------------------------------------------
echo "--- [6/6] Selesai ---"
echo ""
echo "======================================================"
echo " VPS $VPS_HOST ($MODE) siap!"
echo "======================================================"
echo ""

if [ "$MODE" = "shared" ] || [ "$MODE" = "staging" ]; then
    echo "[STAGING] Copy file .env ke VPS:"
    echo "  scp -i $SSH_KEY -P $VPS_PORT .env.staging $VPS_USER@$VPS_HOST:/opt/wch-staging/.env.staging"
    echo ""
    echo "[STAGING] Request SSL (pastikan DNS sudah pointing ke $VPS_HOST):"
    echo "  ssh -i $SSH_KEY -p $VPS_PORT $VPS_USER@$VPS_HOST \\"
    echo "    \"sudo certbot --nginx --non-interactive --agree-tos -m admin@umkmai.id \\"
    echo "      -d stg-api.umkmai.id -d stg-grf.umkmai.id -d stg-n8n.umkmai.id \\"
    echo "      -d stg.umkmai.id -d stg-spadmin.umkmai.id\""
    echo ""
fi

if [ "$MODE" = "shared" ] || [ "$MODE" = "production" ]; then
    echo "[PRODUCTION] Copy file .env ke VPS:"
    echo "  scp -i $SSH_KEY -P $VPS_PORT .env.production $VPS_USER@$VPS_HOST:/opt/wch-production/.env.production"
    echo ""
    echo "[PRODUCTION] Request SSL (pastikan DNS sudah pointing ke $VPS_HOST):"
    echo "  ssh -i $SSH_KEY -p $VPS_PORT $VPS_USER@$VPS_HOST \\"
    echo "    \"sudo certbot --nginx --non-interactive --agree-tos -m admin@umkmai.id \\"
    echo "      -d api.umkmai.id -d grf.umkmai.id -d n8n.umkmai.id \\"
    echo "      -d umkmai.id -d spadmin.umkmai.id\""
    echo ""
fi

echo "[INFO] Untuk trigger deploy via GitHub Actions:"
echo "  Staging : git tag stg-be-v1.0.0 && git push origin stg-be-v1.0.0"
echo "  Production: git tag prod-be-v1.0.0 && git push origin prod-be-v1.0.0"
