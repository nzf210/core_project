#!/bin/bash
# One-time setup script for VPS — run once per environment.
# Usage:
#   ./scripts/vps-setup.sh staging
#   ./scripts/vps-setup.sh production
#
# Prerequisites on VPS:
#   - Docker + Docker Compose v2 installed
#   - User in docker group
#   - GHCR_TOKEN exported or passed as env var

set -e

ENV="${1:-}"
if [[ "$ENV" != "staging" && "$ENV" != "production" ]]; then
  echo "Usage: $0 <staging|production>"
  exit 1
fi

if [[ "$ENV" == "staging" ]]; then
  DEPLOY_DIR="/opt/wch-staging"
  ENV_FILE=".env.staging"
  COMPOSE_OVERRIDE="docker-compose.staging.yml"
else
  DEPLOY_DIR="/opt/wch-production"
  ENV_FILE=".env"
  COMPOSE_OVERRIDE="docker-compose.prod.yml"
fi

echo "=== Setting up $ENV environment at $DEPLOY_DIR ==="

# Create deploy directory
sudo mkdir -p "$DEPLOY_DIR"
sudo chown "$USER:$USER" "$DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR/uploads"

# Copy compose files
cp docker-compose.yml "$DEPLOY_DIR/docker-compose.yml"
cp "$COMPOSE_OVERRIDE" "$DEPLOY_DIR/$COMPOSE_OVERRIDE"

# Create env file from example if it doesn't exist
if [[ ! -f "$DEPLOY_DIR/$ENV_FILE" ]]; then
  if [[ -f ".env.example" ]]; then
    cp .env.example "$DEPLOY_DIR/$ENV_FILE"
    echo ""
    echo "IMPORTANT: Edit $DEPLOY_DIR/$ENV_FILE and fill in all secrets before starting services."
  else
    touch "$DEPLOY_DIR/$ENV_FILE"
    echo "Created empty $DEPLOY_DIR/$ENV_FILE — fill in all required env vars."
  fi
else
  echo "Existing $DEPLOY_DIR/$ENV_FILE kept intact."
fi

# Login to GHCR
if [[ -n "${GHCR_TOKEN:-}" ]]; then
  echo "=== Logging in to GHCR ==="
  echo "$GHCR_TOKEN" | docker login ghcr.io -u nzf210 --password-stdin
else
  echo "GHCR_TOKEN not set — skipping docker login. Run manually:"
  echo "  echo \$GHCR_TOKEN | docker login ghcr.io -u nzf210 --password-stdin"
fi

echo ""
echo "=== Setup complete ==="
echo "Deploy directory : $DEPLOY_DIR"
echo "Env file         : $DEPLOY_DIR/$ENV_FILE"
echo ""
echo "Next steps:"
echo "  1. Edit $DEPLOY_DIR/$ENV_FILE — set DB passwords, JWT secret, API keys, etc."
echo "  2. Push a tag to trigger deployment:"
if [[ "$ENV" == "staging" ]]; then
  echo "     git tag stg-be-v1.0.0 && git push origin stg-be-v1.0.0"
else
  echo "     git tag prod-be-v1.0.0 && git push origin prod-be-v1.0.0"
fi
