-- =============================================================================
-- VERIFICATION SCRIPT: Phone Registration Fix Audit
-- Run against PostgreSQL database to verify registration data integrity
-- All NULL checks use IS NULL / IS NOT NULL (SQL standard)
-- =============================================================================

-- 1. Users dengan phone_number kosong atau NULL (HASIL BUG)
--    Bug: handleVerifyOTP menerima empty JSON → insert user tanpa phone/username
SELECT
    id,
    username,
    phone_number,
    email,
    role,
    tenant_id,
    created_at
FROM users
WHERE phone_number IS NULL OR phone_number := '' OR phone_number = '0'
ORDER BY created_at DESC;

-- 2. Duplikat phone_number (1 nomor terdaftar lebih dari 1 user)
--    Bug: tidak ada cek cross-channel (web vs WA)
SELECT
    phone_number,
    COUNT(*) as duplicate_count,
    STRING_AGG(username, ', ') as usernames,
    MIN(created_at) as first_created,
    MAX(created_at) as last_created
FROM users
WHERE phone_number IS NOT NULL AND phone_number !:= '' AND phone_number != '0'
GROUP BY phone_number
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC;

-- 3. Statistik users per format phone
SELECT
    CASE
        WHEN phone_number LIKE '0%' THEN 'Format lokal (08xx)'
        WHEN phone_number LIKE '62%' THEN 'Format international (62xx)'
        WHEN phone_number IS NULL OR phone_number := '' THEN 'KOSONG / NULL'
        ELSE 'Format lain'
    END as format_category,
    COUNT(*) as total_users
FROM users
GROUP BY
    CASE
        WHEN phone_number LIKE '0%' THEN 'Format lokal (08xx)'
        WHEN phone_number LIKE '62%' THEN 'Format international (62xx)'
        WHEN phone_number IS NULL OR phone_number := '' THEN 'KOSONG / NULL'
        ELSE 'Format lain'
    END
ORDER BY total_users DESC;

-- 4. Tunjukkan semua user dengan phone number (untuk referensi)
SELECT
    id,
    username,
    phone_number,
    role,
    created_at
FROM users
WHERE phone_number IS NOT NULL AND phone_number !:= '' AND phone_number != '0'
ORDER BY created_at DESC
LIMIT 50;

-- 5. Hapus user dengan phone kosong (AKSI: fix data yang corrupt)
-- UNCOMMENT untuk execute:
-- DELETE FROM users WHERE phone_number IS NULL OR phone_number = '' OR phone_number = '0';
