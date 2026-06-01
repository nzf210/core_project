#!/bin/bash
# pre-commit-ai.sh — Pre-commit quality gate untuk Hermes AI agents
# Run before: git commit

set -e

echo "🔍 Hermes Pre-Commit Quality Gate"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILED=0

# 1. Go vet
echo -n "Running go vet... "
if go vet ./... 2>&1; then
  echo -e "${GREEN}PASS${NC}"
else
  echo -e "${RED}FAIL${NC}"
  FAILED=1
fi

# 2. gofmt check
echo -n "Checking gofmt... "
UNFORMATTED=$(gofmt -l .)
if [ -z "$UNFORMATTED" ]; then
  echo -e "${GREEN}PASS${NC}"
else
  echo -e "${RED}FAIL${NC}"
  echo "Unformatted files:"
  echo "$UNFORMATTED"
  FAILED=1
fi

# 3. Build check
echo -n "Building... "
if go build ./... 2>&1; then
  echo -e "${GREEN}PASS${NC}"
else
  echo -e "${RED}FAIL${NC}"
  FAILED=1
fi

# 4. Quick test (no race detector for speed)
echo -n "Running tests (quick)... "
if go test ./... -count=1 -short 2>&1; then
  echo -e "${GREEN}PASS${NC}"
else
  echo -e "${RED}FAIL${NC}"
  FAILED=1
fi

# 5. Check for .env in staging
echo -n "Checking for .env in staging... "
if git diff --cached --name-only | grep -q "\.env$"; then
  echo -e "${RED}FAIL${NC}"
  echo "ERROR: .env file is staged for commit!"
  echo "Remove it with: git reset HEAD <file>"
  FAILED=1
else
  echo -e "${GREEN}PASS${NC}"
fi

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
  echo -e "${GREEN}✅ All pre-commit checks passed${NC}"
  exit 0
else
  echo -e "${RED}❌ Pre-commit checks failed. Fix above issues before committing.${NC}"
  exit 1
fi