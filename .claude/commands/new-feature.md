# Slash Command: /new-feature

## Deskripsi
Scaffold fitur baru lengkap: handler, service layer, DB migration, dan unit test.

## Penggunaan
`/new-feature <nama-fitur> <nama-app>`

## Instruksi untuk Claude
Ketika command ini dijalankan dengan argumen `$ARGUMENTS` (format: `<feature-name> <app-name>`):

Parse argumen:
- `FEATURE_NAME` = kata pertama (contoh: `bot-pnl-report`)
- `APP_NAME` = kata kedua (contoh: `crypto`)

Kemudian lakukan langkah berikut:

1. **Baca konteks app**: Baca file-file yang sudah ada di `apps/$APP_NAME/` untuk memahami pola yang digunakan
2. **Buat migration file**: `shared/migrations/[timestamp]_add_$FEATURE_NAME.up.sql` — buat tabel/kolom yang dibutuhkan fitur
3. **Buat handler**: `apps/$APP_NAME/api/$FEATURE_NAME_handler.go` — implementasikan endpoint HTTP
4. **Buat service layer**: `apps/$APP_NAME/domain/$FEATURE_NAME_service.go` — business logic
5. **Buat unit test**: `apps/$APP_NAME/api/$FEATURE_NAME_handler_test.go` — cover semua happy path dan error case
6. **Register route**: Tambahkan route baru ke main router di `apps/$APP_NAME/`
7. **Jalankan test**: `go test ./apps/$APP_NAME/...` — pastikan tidak ada yang gagal

## Konvensi Penamaan
- File: `snake_case.go`
- Function/Type: `PascalCase`
- Variable: `camelCase`
- DB table: `snake_case` (plural)
- DB column: `snake_case`

## Contoh
`/new-feature bot-pnl-report crypto` → buat fitur laporan PnL bot di app crypto
`/new-feature receipt-ocr umkm` → buat fitur OCR nota di app umkm
