#!/bin/bash
# Quick deployment status checker for staging VPS

echo "=== Latest Deployment Tags ==="
git tag | grep "stg-be-v" | tail -5

echo ""
echo "=== Latest Commits ==="
git log --oneline -3

echo ""
echo "=== GitHub Actions Status ==="
echo "Check: https://github.com/nzf210/core_project/actions"

echo ""
echo "=== VPS Staging Connection Test ==="
echo "To check VPS status, SSH to:"
echo "  ssh -p 3209 deploy@157.15.40.27"
echo ""
echo "Then run:"
echo "  cd /opt/wch-staging"
echo "  docker compose ps | grep -E '(postgres|pgbouncer|redis)'"
echo "  docker compose logs pgbouncer --tail=50"
