#!/bin/bash
# Seed Data Script — Populate test database with sample data
# Usage: ./scripts/seed-data.sh

set -e

echo "=== WCH Platform Seed Data Script ==="
echo ""

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-wch_core}

echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# Check if psql is available
if ! command -v psql &> /dev/null; then
    echo "Error: psql not found. Install PostgreSQL client first."
    exit 1
fi

# Test database connection
echo "Testing database connection..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1;" > /dev/null 2>&1 || {
    echo "Error: Cannot connect to database"
    exit 1
}
echo "✅ Database connected"
echo ""

# Execute seed SQL
echo "Inserting seed data..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << 'EOF'

-- Seed Data for WCH Platform

-- 1. Create test superadmin (password: admin123)
INSERT INTO users (id, name, email, phone_number, password_hash, role, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Super Admin',
    'admin@wch.id',
    '+6281234567890',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewY5GyYIv3r5sW0G', -- admin123
    'superadmin',
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 2. Create test tenant (UMKM - Warung Maju)
INSERT INTO tenants (id, name, business_type, status, plan, plan_expired_at, owner_id, created_at)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Warung Maju',
    'warung',
    'active',
    'pro',
    NOW() + INTERVAL '30 days',
    '00000000-0000-0000-0000-000000000001',
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 3. Create test owner user (password: owner123)
INSERT INTO users (id, tenant_id, name, email, phone_number, password_hash, role, created_at)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'Budi Santoso',
    'budi@warungmaju.com',
    '+6281234567891',
    '$2a$12$xQ7gZJ5YqH8vLQJ4gZJ5YOzJ4gZJ5YqH8vLQJ4gZJ5YOzJ4gZJ5Yq', -- owner123
    'owner',
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Update tenant owner
UPDATE tenants SET owner_id = '22222222-2222-2222-2222-222222222222'
WHERE id = '11111111-1111-1111-1111-111111111111';

-- 4. Create test staff/kasir (password: staff123)
INSERT INTO users (id, tenant_id, name, email, phone_number, password_hash, role, created_at)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    'Siti Rahayu',
    'siti@warungmaju.com',
    '+6281234567892',
    '$2a$12$yR8hAK6ZrI9wMRK5hAK6ZPzK5hAK6ZrI9wMRK5hAK6ZPzK5hAK6Zr', -- staff123
    'staff',
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- 5. Create sample products
INSERT INTO products (id, tenant_id, name, sku, price, stock, category, created_at)
VALUES
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Indomie Goreng', 'IMI-001', 300000, 100, 'Makanan', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Aqua 600ml', 'AQA-001', 400000, 50, 'Minuman', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Teh Pucuk', 'TEA-001', 500000, 30, 'Minuman', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Kopi Kapal Api', 'KOP-001', 1500000, 20, 'Minuman', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Roti Tawar', 'ROT-001', 1200000, 15, 'Makanan', NOW())
ON CONFLICT DO NOTHING;

-- 6. Create chart of accounts (COA) for double-entry accounting
INSERT INTO chart_of_accounts (id, tenant_id, code, name, account_type, created_at)
VALUES
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1-1000', 'Kas', 'asset', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1-1100', 'Bank', 'asset', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '1-2000', 'Persediaan Barang', 'asset', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '4-1000', 'Pendapatan Penjualan', 'revenue', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5-1000', 'Beban Pokok Penjualan', 'expense', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', '5-2000', 'Beban Operasional', 'expense', NOW())
ON CONFLICT DO NOTHING;

-- 7. Create sample FAQ for chatbot
INSERT INTO chatbot_faqs (id, tenant_id, question, answer, created_at)
VALUES
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Jam buka?', 'Warung Maju buka setiap hari dari jam 08:00 - 22:00 WIB', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Alamat?', 'Jl. Raya Merdeka No. 123, Jakarta Selatan', NOW()),
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'Apakah bisa pesan online?', 'Ya, bisa pesan via WhatsApp atau langsung datang ke toko', NOW())
ON CONFLICT DO NOTHING;

-- 8. Create sample voucher program
INSERT INTO voucher_programs (id, name, description, plan_id, duration_months, max_uses, created_at)
VALUES
    (gen_random_uuid(), 'Promo Tahun Baru 2026', 'Voucher gratis 1 bulan paket Pro', 'pro', 1, 1, NOW())
ON CONFLICT DO NOTHING;

-- 9. Create tenant chatbot config
INSERT INTO tenant_chatbot_configs (
    id, tenant_id, language, tone, system_prompt, welcome_message,
    fallback_message, outside_hours_message, business_hours_start,
    business_hours_end, business_days, is_active, created_at
)
VALUES (
    gen_random_uuid(),
    '11111111-1111-1111-1111-111111111111',
    'id',
    'friendly',
    'Kamu adalah asisten AI untuk Warung Maju. Jawab pertanyaan pelanggan dengan ramah dan informatif.',
    'Halo! Selamat datang di Warung Maju. Ada yang bisa saya bantu?',
    'Maaf, saya tidak mengerti pertanyaan Anda. Bisa dijelaskan lebih detail?',
    'Terima kasih sudah menghubungi. Kami buka jam 08:00-22:00 WIB. Silakan hubungi lagi saat jam operasional.',
    '08:00:00',
    '22:00:00',
    ARRAY['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'],
    true,
    NOW()
) ON CONFLICT (tenant_id) DO NOTHING;

EOF

echo ""
echo "✅ Seed data inserted successfully"
echo ""
echo "Test Accounts:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Superadmin:"
echo "  Email: admin@wch.id"
echo "  Password: admin123"
echo ""
echo "Tenant Owner (Warung Maju):"
echo "  Email: budi@warungmaju.com"
echo "  Password: owner123"
echo "  Phone: +6281234567891"
echo ""
echo "Staff/Kasir:"
echo "  Email: siti@warungmaju.com"
echo "  Password: staff123"
echo ""
echo "Sample Data:"
echo "  - 5 Products (Indomie, Aqua, Teh Pucuk, dll)"
echo "  - 6 Chart of Accounts (Kas, Bank, Persediaan, dll)"
echo "  - 3 FAQ entries for chatbot"
echo "  - 1 Voucher program"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Ready to test! Login with credentials above."
