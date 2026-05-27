# Slash Command: /run-tests

## Deskripsi
Jalankan seluruh test suite monorepo dan laporkan hasilnya.

## Penggunaan
`/run-tests` atau `/run-tests <nama-app-atau-service>`

## Instruksi untuk Claude
Ketika command ini dijalankan:

Jika ada argumen `$ARGUMENTS`:
- Jalankan: `go test -v -race ./services/$ARGUMENTS/...` atau `go test -v -race ./apps/$ARGUMENTS/...`

Jika tidak ada argumen:
1. Jalankan: `go test -race ./...`
2. Jalankan: `go vet ./...`

Setelah test selesai:
- Jika **semua test lulus**: laporkan ringkasan "✅ Semua N test lulus. Coverage: X%"
- Jika **ada yang gagal**: identifikasi test yang gagal, analisis error message, dan **langsung perbaiki** kode yang menyebabkan kegagalan, kemudian jalankan ulang test
- Ulangi siklus fix → test maksimal 3 kali, jika masih gagal → laporkan ke user dengan detail error

## Opsi Tambahan
- `/run-tests --coverage` → tambahkan flag `-coverprofile=coverage.out` dan jalankan `go tool cover -html=coverage.out`
- `/run-tests --bench` → jalankan benchmark test dengan `-bench=.`
