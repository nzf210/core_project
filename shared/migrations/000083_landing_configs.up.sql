-- 000083_landing_configs.up.sql
-- Landing page dynamic content configuration (F060 extension)
-- Superadmin can edit landing page content via JSON editor

CREATE TABLE IF NOT EXISTS landing_configs (
    id          VARCHAR(50) PRIMARY KEY,   -- e.g. 'hero', 'features', 'steps', 'testimonials', 'trust', 'cta', 'footer'
    content     JSONB NOT NULL DEFAULT '{}',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_landing_configs_is_active ON landing_configs(is_active);

-- Seed default content matching static LandingPage.vue
INSERT INTO landing_configs (id, content) VALUES
(
  'hero',
  '{"badge":"Kasir · Pembukuan · AI Chatbot — dalam satu platform","title_line1":"Kelola Usaha","title_line2":"Tanpa Ribet,","title_line3":"Tanpa Accountant","subtitle":"WCH Platform adalah all-in-one aplikasi kasir, pembukuan double-entry, dan AI Customer Service untuk UMKM Indonesia. Mulai gratis, upgrade kapan saja.","cta_primary":"Mulai Gratis","cta_secondary":"Lihat Fitur","stats":[{"value":"100+","label":"UMKM Aktif"},{"value":"50K+","label":"Transaksi\/Bulan"},{"value":"24\/7","label":"AI Chatbot"}]}'
),
(
  'features',
  '[{"icon":"💰","title":"Kasir POS","desc":"Catat transaksi jual-beli dengan cepat. Dukung multi-pembayaran: Tunai, QRIS, Transfer, E-Wallet."},{"icon":"📒","title":"Pembukuan Otomatis","desc":"Double-entry accounting. Setiap transaksi POS langsung tercatat di jurnal — tidak perlu accountant."},{"icon":"🤖","title":"AI Customer Service","desc":"Bot WhatsApp otomatis jawab pertanyaan pelanggan 24\/7. Bisa di-training dengan FAQ toko Anda."},{"icon":"📊","title":"Laporan Keuangan","desc":"Laba rugi, arus kas, neraca — siap pakai untuk pajak dan pengajuan kredit bank."},{"icon":"🏪","title":"Multi-Toko","desc":"Kelola beberapa cabang dalam satu akun. Cocok untuk franchise dan warung kopi."},{"icon":"🔗","title":"Integrasi Marketplace","desc":"Hubungkan dengan GoFood, Grab, Shopee. Stok tersinkron otomatis di satu dashboard."}]'
),
(
  'steps',
  '[{"title":"Daftar dalam 30 detik","desc":"Masukkan nomor WhatsApp, terima OTP, langsung masuk. Tanpa verifikasi email."},{"title":"Pilih paket atau mulai gratis","desc":"Lite gratis selamanya. Atau pilih Pro\/Ultimate untuk fitur AI dan multi-user."},{"title":"Mulai berjualan","desc":"Kasir langsung bisa dipakai. Pembukuan berjalan otomatis di belakang layar."}]'
),
(
  'testimonials',
  '[{"quote":"Dulu pembukuan pakai Excel, sekarang semua otomatis. Owner warung kopi saya bisa fokus layani pelanggan.","name":"Budi Santoso","role":"Pemilik Warung Kopi, Bandung"},{"quote":"AI chatbot-nya jawab pertanyaan pelanggan di malam hari. Servis tetap prima meski saya lagi tutup.","name":"Siti Rahayu","role":"Pemilik Toko Fashion, Surabaya"},{"quote":"Laporan keuangan langsung jadi, tinggal export ke PDF untuk pengajuan KUR ke bank.","name":"Ahmad Hidayat","role":"Pemilik Klinik Pratama, Medan"}]'
),
(
  'trust',
  '{"label":"Telah dipercaya oleh pelaku UMKM di","cities":["Jakarta","Surabaya","Bandung","Medan","Makassar","Semarang"]}'
),
(
  'cta',
  '{"title":"Siap mengelola usaha dengan lebih cerdas?","subtitle":"Bergabung dengan 100+ UMKM Indonesia hari ini. Gratis selamanya.","button":"Daftar Sekarang — Gratis"}'
),
(
  'footer',
  '{"description":"All-in-one aplikasi kasir, pembukuan, dan AI chatbot untuk UMKM Indonesia.","links":{"produk":["Fitur","Harga","Daftar"],"perusahaan":["Tentang Kami","Blog","Karir"],"bantuan":["Pusat Bantuan","Syarat & Ketentuan","Kebijakan Privasi"]},"copyright":"© 2026 WCH Platform. Hak cipta dilindungi."}'
);
