#!/bin/bash
# WCH Platform Staging VPS Setup Script
# Run this once on a fresh VPS to prepare for CI/CD deployments

set -e

echo "============================================="
echo " WCH Platform Staging VPS Setup"
echo "============================================="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
   echo "Please run as root (sudo bash staging-setup.sh)"
   exit 1
fi

# 1. Install Docker
echo "[1/8] Installing Docker..."
if ! command -v docker &> /dev/null; then
    # Download Docker install script to disk before executing (avoids piping from internet to shell)
    curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
    chmod +x /tmp/get-docker.sh
    /tmp/get-docker.sh
    usermod -aG docker $SUDO_USER
    rm /tmp/get-docker.sh
else
    echo "  ✅ Docker already installed"
fi

# 2. Install Docker Compose
echo "[2/8] Installing Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
else
    echo "  ✅ Docker Compose already installed"
fi

# 3. Install required packages
echo "[3/8] Installing system packages..."
apt-get update
apt-get install -y git make curl wget nginx certbot python3-certbot-nginx

# 4. Create application directory
echo "[4/8] Creating application directory..."
mkdir -p /opt/wch-platform
mkdir -p /backup/wch
chown -R $SUDO_USER:$SUDO_USER /opt/wch-platform
chown -R $SUDO_USER:$SUDO_USER /backup/wch

# 5. Clone repository (manual step - requires SSH key)
echo "[5/8] Repository setup..."
echo "  ⚠️  Manual step required:"
echo "     cd /opt/wch-platform"
echo "     git clone <your-repo-url> ."
echo "     (Setup SSH key first if using private repo)"
read -p "  Press Enter after you've cloned the repo..."

# 6. Setup environment file
echo "[6/8] Setting up .env file..."
if [ ! -f /opt/wch-platform/.env ]; then
    cp /opt/wch-platform/.env.example /opt/wch-platform/.env
    echo "  ⚠️  IMPORTANT: Edit /opt/wch-platform/.env and set:"
    echo "     - JWT_SECRET (generate: openssl rand -base64 32)"
    echo "     - ENCRYPTION_KEY (generate: openssl rand -base64 32)"
    echo "     - GRAFANA_ADMIN_PASSWORD (strong password)"
    echo "     - DB_PASSWORD (strong password)"
    echo "     - REDIS_PASSWORD (strong password)"
    echo "     - All API keys (MiniMax, Xendit, Telegram, etc.)"
    read -p "  Press Enter after you've edited .env..."
else
    echo "  ✅ .env already exists"
fi

# 7. Setup Nginx reverse proxy
echo "[7/8] Setting up Nginx..."
DOMAIN=$(grep -oP 'VPS_HOST=\K.*' /opt/wch-platform/.env 2>/dev/null || echo "api.example.com")
cat > /etc/nginx/sites-available/wch-platform <<EOF
server {
    listen 80;
    server_name $DOMAIN;

    location / {
        proxy_pass http://127.0.0.1:8010;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
EOF

ln -sf /etc/nginx/sites-available/wch-platform /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx

echo "  ✅ Nginx configured for HTTP"
echo "  ⚠️  Setup SSL with: certbot --nginx -d $DOMAIN"

# 8. Setup backup cron
echo "[8/8] Setting up backup cron..."
(crontab -l 2>/dev/null; echo "0 2 * * * /opt/wch-platform/scripts/backup.sh /backup/wch") | crontab -
echo "  ✅ Daily backup scheduled at 2 AM"

echo ""
echo "============================================="
echo " ✅ VPS Setup Complete!"
echo "============================================="
echo ""
echo "Next steps:"
echo "1. Edit .env file with production secrets"
echo "2. Setup SSL: certbot --nginx -d $DOMAIN"
echo "3. Deploy via CI/CD or manually:"
echo "   cd /opt/wch-platform"
echo "   docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d --build"
echo ""
echo "Verify:"
echo "  - http://$DOMAIN (should redirect to HTTPS after certbot)"
echo "  - docker-compose ps"
echo "  - curl http://localhost:8010/health"
echo ""
