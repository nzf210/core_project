#!/bin/bash
set -e

echo "======================================"
echo " Setting up WCH Platform on new VPS"
echo "======================================"

# 1. Update system & Install Dependencies
sudo apt-get update && sudo apt-get upgrade -y
sudo apt-get install -y git curl wget jq ufw nginx certbot python3-certbot-nginx

# 2. Install Docker & Docker Compose
if ! command -v docker &> /dev/null; then
    curl --proto '=https' --tlsv1.2 -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
fi

if ! command -v docker-compose &> /dev/null; then
    sudo curl --proto "=https" --tlsv1.2 -sSfL "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
fi

# 3. Setup Firewall
sudo ufw allow OpenSSH
sudo ufw allow 'Nginx Full'
sudo ufw --force enable

# 4. Clone Repository (Requires SSH Key setup on GitHub)
INSTALL_DIR="/opt/wch-platform"
if [[ ! -d "$INSTALL_DIR" ]]; then
    sudo mkdir -p $INSTALL_DIR
    sudo chown $USER:$USER $INSTALL_DIR
    # git clone git@github.com:your-repo/wch-platform.git $INSTALL_DIR
    echo "Please clone the repository manually to $INSTALL_DIR"
fi

echo "Setup complete! Please configure your .env file in $INSTALL_DIR and run docker-compose up -d"
