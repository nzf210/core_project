# F069: Redis-Backed WA Registration Session Persistence

**Spec Status:** ✅ Approved
**Implementation:** 🔨 In Progress

## Deskripsi

`waRegistrationSessions` di `services/wa-gateway` saat ini disimpan sebagai in-memory map. Jika wa-gateway restart di tengah wizard registrasi, semua session aktif hilang dan user harus mulai ulang dari awal tanpa notifikasi.

Fitur ini memindahkan penyimpanan session ke Redis menggunakan `redisShared`, sehingga session bertahan melewati restart.

## Spec

- Session disimpan di Redis dengan key `wa:reg-session:{senderJID}`, TTL 30 menit
- Data session di-serialize ke JSON
- `startWARegistration()` load session dari Redis jika ada (lanjut dari step terakhir)
- Setiap update step → re-write ke Redis
- `deleteWARegistrationSession()` hapus dari Redis setelah selesai/cancel
- Fallback: jika Redis tidak tersedia (`redisShared == nil`) → gunakan in-memory map seperti semula
- Backward compatible: tidak ada perubahan API/endpoint
- TTL 30 menit — cukup untuk wizard yang biasanya selesai < 5 menit

## Acceptance Criteria

- [x] AC-1: Session registrasi WA tetap aktif setelah wa-gateway restart
- [x] AC-2: User yang sedang di tengah wizard dapat melanjutkan dari step terakhir
- [x] AC-3: Session expired otomatis setelah 30 menit via Redis TTL
- [x] AC-4: Jika Redis tidak tersedia, fallback ke in-memory (tidak error)
- [x] AC-5: Build dan test lulus

## Files yang perlu diubah

- `services/wa-gateway/registration_handler.go` — ganti map dengan Redis CRUD helpers
