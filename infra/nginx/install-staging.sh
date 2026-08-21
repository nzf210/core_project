#!/bin/bash
# =============================================================================
# WCH Platform — Install Nginx as reverse proxy for staging
# Jalankan sebagai user dengan sudo (bukan deploy):
#   ssh administrator@157.15.40.27 -p 3209
#   sudo bash /opt/wch-staging/infra/nginx/install-staging.sh
# =============================================================================
set -euo pipefail

CONF_SRC="$(dirname "$0")/staging.conf"
CONF_DEST="/etc/nginx/sites-available/wch-staging"
CONF_LINK="/etc/nginx/sites-enabled/wch-staging"

echo "==> Installing Nginx..."
apt-get update -qq
apt-get install -y nginx

echo "==> Copying config..."
cp "$CONF_SRC" "$CONF_DEST"
ln -sf "$CONF_DEST" "$CONF_LINK"

# Hapus default site agar tidak bentrok di port 80
rm -f /etc/nginx/sites-enabled/default

echo "==> Testing config..."
nginx -t

echo "==> Starting Nginx..."
systemctl enable nginx
systemctl restart nginx

echo ""
echo "Done. Nginx aktif sebagai reverse proxy."
echo "Sekarang update Cloudflare Tunnel agar semua hostname → http://localhost:80"
echo "Lihat: infra/nginx/cloudflare-tunnel-config.yml"
