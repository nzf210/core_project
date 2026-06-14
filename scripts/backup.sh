#!/bin/bash
# WCH Platform E2E Backup Script (Database + N8N Workflows)
# Run manually or via crontab

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$DIR/.env" 2>/dev/null || true

# Default backup destination
BACKUP_DIR="${1:-/tmp/wch_backups}"
DATE=$(date +"%Y%m%d_%H%M")
BACKUP_FILE="$BACKUP_DIR/backup_wch_$DATE.tar.gz"

echo "============================================="
echo " WCH Platform E2E Backup"
echo "============================================="

# Ensure backup directory exists
mkdir -p "$BACKUP_DIR"
cd "$DIR" # Move to core_project root

# Temporary staging dir for this backup
STAGING_DIR="/tmp/wch_backup_staging_$DATE"
mkdir -p "$STAGING_DIR"

echo "[1/3] Dumping Databases from Docker..."
# Ensure postgres container is running
if ! docker ps | grep -q "wch-postgres"; then
    echo "ERROR: wch-postgres container is not running. Cannot dump database."
    rm -rf "$STAGING_DIR"
    exit 1
fi

# 1. Main Application DB
echo "  -> Dumping ${DB_NAME:-wch_platform}..."
docker exec -t wch-postgres pg_dump -U ${DB_USER:-wch_admin} -d ${DB_NAME:-wch_platform} -F c > "$STAGING_DIR/${DB_NAME:-wch_platform}.dump"

# 2. N8N Persistence DB
echo "  -> Dumping ${N8N_DB_NAME:-wch_n8n_db} (N8N)..."
docker exec -t wch-postgres pg_dump -U ${N8N_DB_USER:-wch_n8n} -d ${N8N_DB_NAME:-wch_n8n_db} -F c > "$STAGING_DIR/${N8N_DB_NAME:-wch_n8n_db}.dump"

echo "[2/3] Copying N8N Workflows & Configs..."
mkdir -p "$STAGING_DIR/n8n_workflows"
if [ -d "infra/n8n/workflows" ]; then
    cp -r infra/n8n/workflows/* "$STAGING_DIR/n8n_workflows/" 2>/dev/null || true
fi
# Copy .env safely (strip secrets later if you want, but backup usually needs them)
cp .env "$STAGING_DIR/.env.backup" 2>/dev/null || true

echo "[3/3] Archiving..."
tar -czf "$BACKUP_FILE" -C "$STAGING_DIR" .

# Cleanup staging
rm -rf "$STAGING_DIR"

echo "============================================="
echo "✅ Backup Completed Successfully!"
echo "📁 Location: $BACKUP_FILE"
echo "============================================="
echo "To restore in a new VPS:"
echo "1. docker exec -i wch-postgres pg_restore -U postgres -d wch_core < wch_core.dump"
echo "2. docker exec -i wch-postgres pg_restore -U postgres -d wch_n8n_db < wch_n8n_db.dump"
echo "3. Copy back .env.backup to .env"
echo "4. docker-compose up -d"
