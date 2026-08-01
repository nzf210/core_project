#!/bin/bash
# scripts/release.sh — buat annotated tag dan push ke origin
# Dipanggil dari Makefile: make stg-be-v1.0.1 atau make prod-umkm-v1.0.1
#
# Usage:
#   bash scripts/release.sh stg-be-v1.0.1
#   bash scripts/release.sh prod-umkm-v1.0.1 "pesan opsional"

set -e

cd "$(dirname "$0")/.."

TAG="$1"
MESSAGE="$2"

if [[ -z "$TAG" ]]; then
  echo "ERROR: Tag wajib diisi."
  echo "Contoh: make stg-be-v1.0.1"
  exit 1
fi

# Validasi format: harus mengandung vX.Y.Z di bagian akhir
if ! [[ "$TAG" =~ v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "ERROR: Tag '$TAG' harus diakhiri dengan vX.Y.Z (contoh: stg-be-v1.0.1)"
  exit 1
fi

# Cek tag sudah ada
if git tag --list "$TAG" | grep -q "^${TAG}$"; then
  echo "ERROR: Tag $TAG sudah ada."
  exit 1
fi

# Cek uncommitted changes
if [[ -n "$(git status --porcelain)" ]]; then
  echo "WARNING: Ada perubahan yang belum di-commit:"
  git status --short
  echo ""
  read -rp "Lanjutkan? (y/N): " CONFIRM
  [[ "$CONFIRM" =~ ^[Yy]$ ]] || exit 0
fi

BRANCH=$(git rev-parse --abbrev-ref HEAD)
MESSAGE="${MESSAGE:-Release $TAG}"

echo "==> Tag    : $TAG"
echo "==> Branch : $BRANCH"
echo "==> Pesan  : $MESSAGE"
echo ""

git tag -a "$TAG" -m "$MESSAGE"
git push origin "$BRANCH"
git push origin "$TAG"

echo ""
echo "✓ $TAG berhasil dipush."
