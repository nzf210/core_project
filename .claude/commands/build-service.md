# Slash Command: /build-service

## Deskripsi
Build dan verifikasi service Go tertentu dalam monorepo.

## Penggunaan
`/build-service <nama-service>`

## Instruksi untuk Claude
Ketika command ini dijalankan dengan argumen `$ARGUMENTS`:

1. Pergi ke direktori `services/$ARGUMENTS/` atau `apps/$ARGUMENTS/`
2. Jalankan `go build ./services/$ARGUMENTS/...` atau `go build ./apps/$ARGUMENTS/...`
3. Jika gagal, baca error dan perbaiki masalahnya
4. Jalankan `go vet ./services/$ARGUMENTS/...` untuk lint check
5. Jalankan `go test ./services/$ARGUMENTS/...` untuk unit test
6. Laporkan hasil: berhasil atau daftar error yang ditemukan dan sudah diperbaiki

## Contoh
`/build-service auth-service` → build, vet, dan test `services/auth-service`
`/build-service crypto` → build, vet, dan test `apps/crypto`
