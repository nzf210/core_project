#!/bin/bash
# post-merge-ai.sh — Post-merge automation untuk Hermes AI agents
# Run after: git merge (atau git pull)

set -e

echo "🤖 Hermes Post-Merge Automation"

# Check if this was a merge commit
if [ -z "$(git log --oneline -1 | grep 'Merge')" ]; then
  # Not a merge, just a regular commit — minimal action
  echo "Regular commit detected, skipping merge automation."
  exit 0
fi

# Colors
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}Running post-merge tasks...${NC}"

# 1. Check if go.mod/go.sum changed
if git diff --name-only HEAD~1 | grep -q "go\.\(mod\|sum\)"; then
  echo "go.mod/go.sum changed — running go mod tidy"
  go mod tidy
fi

# 2. Check if any new migrations exist
NEW_MIGRATIONS=$(git diff --name-only HEAD~1 | grep "shared/migrations/.*\.up\.sql" || true)
if [ -n "$NEW_MIGRATIONS" ]; then
  echo "New migrations detected:"
  echo "$NEW_MIGRATIONS"
  echo "Remember to run: make db-migrate"
fi

# 3. Check if frontend dependencies changed
if git diff --name-only HEAD~1 | grep -q "frontend/.*package\.json"; then
  echo "Frontend package.json changed — you may need to run:"
  echo "  cd frontend/umkm-web && npm install"
  echo "  cd frontend/campaign-web && npm install"
fi

# 4. Notify about binary rebuild
BUILT_BINARIES_CHANGED=$(git diff --name-only HEAD~1 | grep -v "bin/" | grep -q "." || true)
if git diff --name-only HEAD~1 | grep -q "\.go$"; then
  echo "Go code changed — rebuild binaries with: make build-all"
fi

echo -e "${GREEN}✅ Post-merge automation complete${NC}"