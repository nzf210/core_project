# Plan: Staff/Karyawan Management UI

**Goal**: Menampilkan daftar karyawan di UMKM dan memungkinkan manajemen (ubah username, reset password, ganti no HP) sesuai dengan Spec-First Workflow.

## Langkah-langkah

1. **Tambahkan endpoint Backend `GET /staff` dan `PUT /staff/{id}` (Auth Service)**
   - `GET /staff` (Auth Service): Mengambil semua user dengan role `kasir` atau `staff` untuk tenant aktif.
   - `PUT /staff/{id}` (Auth Service): Memperbarui data staff (username, phone_number, reset password).
   - `DELETE /staff/{id}` (opsional): Menghapus staff.

2. **Perbarui Frontend API (umkm-web)**
   - Tambah methods di `api.ts` atau komponen untuk fetch list staff dan update staff.

3. **Buat/Perbarui UI Management Staff (Settings.vue atau komponen baru)**
   - Tambah tab/section "Daftar Pegawai" di `Settings.vue` di bawah form tambah pegawai.
   - Menampilkan tabel: Username, Email, No. WA, Role, Action.
   - Action: Edit (modal), Hapus.
   - Edit Modal: Input username, phone, reset password (opsional).

## Konvensi yang harus diikuti
- Gunakan `auth-service` untuk manajemen user.
- Parameterize query database.
- Terapkan `X-Tenant-ID` filter.

