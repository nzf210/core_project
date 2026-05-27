# Slash Command: /db-migrate

## Deskripsi
Buat, validasi, dan jalankan database migration files.

## Penggunaan
`/db-migrate <action> [nama-migration]`

Actions:
- `create <nama>` — buat migration file baru
- `validate` — validasi semua migration files (SQL syntax check)
- `status` — tampilkan daftar migrations yang sudah/belum dijalankan

## Instruksi untuk Claude
Ketika command ini dijalankan dengan argumen `$ARGUMENTS`:

### Action: `create <nama>`
1. Generate timestamp: format `YYYYMMDDHHMMSS`
2. Buat dua file di `shared/migrations/`:
   - `[timestamp]_[nama].up.sql` — berisi SQL untuk membuat/mengubah schema
   - `[timestamp]_[nama].down.sql` — berisi SQL untuk rollback (kebalikan dari up)
3. Isi template awal di kedua file sesuai konteks nama migration
4. Pastikan menggunakan konvensi: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`

### Action: `validate`
1. Baca semua file `.sql` di `shared/migrations/`
2. Verifikasi tidak ada syntax error umum (unclosed quotes, missing semicolon, dll)
3. Pastikan setiap `.up.sql` punya pasangan `.down.sql`
4. Laporkan hasil validasi

### Action: `status`
1. Baca daftar file migration di `shared/migrations/`
2. Tampilkan dalam format tabel: `[timestamp] | [nama] | [status: applied/pending]`

## Contoh
`/db-migrate create add-bots-table` → buat migration untuk tabel bots
`/db-migrate validate` → validasi semua migration files
