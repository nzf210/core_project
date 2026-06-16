#!/bin/bash
# scripts/push_and_pr.sh
# Helper untuk push branch & buka PR.
# Usage:
#   1. Set GH_TOKEN di environment (atau pakai `gh auth login` + GitHub CLI)
#   2. bash scripts/push_and_pr.sh
#
# Atau pakai GitHub CLI langsung:
#   gh auth login
#   gh pr create --base main --head fix/tier1-onboarding-loop \
#                --title "Priority 1+2: onboarding sync, AI CS wizard, cash flow PDF, Excel I/O" \
#                --body-file PR_DESCRIPTION.md

set -e

BRANCH="fix/tier1-onboarding-loop"
TITLE="Priority 1+2 (F019, F020, F021, F022): onboarding sync, AI CS wizard, cash flow PDF, Excel/Sheet I/O"

echo "==> Cek working tree"
cd "$(dirname "$0")/.."
git status

echo ""
echo "==> Push branch $BRANCH ke origin"
if command -v gh &> /dev/null; then
  echo "(menggunakan GitHub CLI)"
  git push -u origin "$BRANCH"
  gh pr create --base main --head "$BRANCH" --title "$TITLE" --body-file PR_DESCRIPTION.md
else
  echo "(tanpa GitHub CLI — Anda perlu set GH_TOKEN / GITHUB_TOKEN)"
  echo ""
  echo "Cara 1 — Personal Access Token (HTTPS):"
  echo "  export GITHUB_TOKEN=ghp_xxx..."
  echo "  git push https://x-access-token:\$GITHUB_TOKEN@github.com/nzf210/core_project.git $BRANCH"
  echo "  curl -X POST -H \"Authorization: token \$GITHUB_TOKEN\" \\"
  echo "       -H \"Accept: application/vnd.github.v3+json\" \\"
  echo "       https://api.github.com/repos/nzf210/core_project/pulls \\"
  echo "       -d '{\"title\": \"$TITLE\", \"head\": \"$BRANCH\", \"base\": \"main\", \"body\": \"<PR_DESCRIPTION.md content>\"}'"
  echo ""
  echo "Cara 2 — SSH (kalau sudah set ssh-agent):"
  echo "  git remote set-url origin git@github.com:nzf210/core_project.git"
  echo "  git push -u origin $BRANCH"
  echo ""
  echo "Cara 3 — Install GitHub CLI (recommended):"
  echo "  # macOS:  brew install gh"
  echo "  # Linux:  sudo apt install gh  /  https://cli.github.com/"
  echo "  gh auth login"
  echo "  gh pr create --base main --head $BRANCH --title \"$TITLE\" --body-file PR_DESCRIPTION.md"
fi
