#!/bin/bash
# =============================================================================
# Setup Nginx di VPS — jalankan dari LOCAL machine:
#   bash infra/deploy/setup-nginx.sh staging
#   bash infra/deploy/setup-nginx.sh production
# =============================================================================
set -e

ENV=${1:-staging}
VPS_USER="deploy"
VPS_HOST="157.15.40.27"
VPS_PORT="3209"
SSH_KEY="$HOME/.ssh/github_actions_wch"

SSH="ssh -i $SSH_KEY -p $VPS_PORT $VPS_USER@$VPS_HOST"
SCP="scp -i $SSH_KEY -P $VPS_PORT"

echo "=== Target: $ENV ==="

# 1. Install Nginx + Certbot jika belum ada
$SSH "which nginx > /dev/null 2>&1 || (sudo apt-get update -qq && sudo apt-get install -y nginx certbot python3-certbot-nginx)"

# 2. Copy config yang sesuai
if [ "$ENV" = "production" ]; then
    CONF_SRC="infra/deploy/nginx-prod.conf"
    CONF_DEST="/etc/nginx/sites-available/wch-prod.conf"
    CONF_LINK="/etc/nginx/sites-enabled/wch-prod.conf"
    CERTBOT_DOMAINS="-d api.umkmai.id -d grf.umkmai.id -d n8n.umkmai.id -d umkmai.id -d spadmin.umkmai.id"
else
    CONF_SRC="infra/deploy/nginx-staging.conf"
    CONF_DEST="/etc/nginx/sites-available/wch-staging.conf"
    CONF_LINK="/etc/nginx/sites-enabled/wch-staging.conf"
    CERTBOT_DOMAINS="-d stg-api.umkmai.id -d stg-grf.umkmai.id -d stg-n8n.umkmai.id -d stg.umkmai.id -d stg-spadmin.umkmai.id"
fi

echo "=== Copy Nginx config ke VPS ==="
# Upload ke /tmp dulu karena butuh sudo untuk pindah ke /etc/nginx
$SCP "$CONF_SRC" "$VPS_USER@$VPS_HOST:/tmp/wch-nginx.conf"
$SSH "sudo mv /tmp/wch-nginx.conf $CONF_DEST && sudo ln -sf $CONF_DEST $CONF_LINK"

# 3. Hapus default nginx config jika masih ada
$SSH "sudo rm -f /etc/nginx/sites-enabled/default"

# 4. Test dan reload
echo "=== Test dan reload Nginx ==="
$SSH "sudo nginx -t && sudo systemctl enable nginx && sudo systemctl reload nginx"

echo ""
echo "=== Nginx aktif! ==="
echo ""
echo "=== Langkah selanjutnya: Request SSL ==="
echo "Pastikan DNS domain sudah pointing ke $VPS_HOST, lalu jalankan:"
echo ""
echo "  ssh -i $SSH_KEY -p $VPS_PORT $VPS_USER@$VPS_HOST \\"
echo "    \"sudo certbot --nginx --non-interactive --agree-tos -m admin@umkmai.id $CERTBOT_DOMAINS\""
