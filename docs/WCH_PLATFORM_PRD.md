# Product Requirements Document (PRD) — WCH Platform
**Versi:** 1.1.0
**Tanggal Update Terakhir:** 2026-05-29
**Ringkasan Perubahan:** Sinkronisasi Detail Spesifikasi dengan Feature List UMKM terbaru (penambahan tipe bisnis Hotel/Penginapan, fitur daftar pengeluaran & arus kas, penyesuaian channel chatbot, penambahan notifikasi WA & Telegram pada laporan AI, serta skema promo kupon tier Lite).

---

## 1. Pendahuluan
WCH Platform adalah ekosistem SaaS *multi-product* berbasis monorepo yang dirancang untuk melayani tiga vertikal bisnis utama: UMKM, Trader Kripto, dan Tim Kampanye Politik. Dokumen ini adalah acuan terpusat (PRD) untuk seluruh tim produk dan developer.

Jika ada penambahan/pengurangan fitur di masa depan, PRD ini akan diperbarui dengan mencatat perubahan di bagian **Ringkasan Perubahan** di atas, lalu menyesuaikan *Feature List* dan spesifikasi di bawahnya.

---

## 2. Feature List Saat Ini (MVP & Live)
*(Hanya mencatat fitur yang sudah ada sesuai MVP, tidak diubah)*

### A. SaaS UMKM (AI Agent & Pembukuan)
1. **Multi-Tenant Onboarding & Dynamic Dashboard Widget**: Onboarding instan dengan dashboard khusus berdasarkan tipe bisnis (Warung, Laundry, Restoran, Hotel atau Penginapan dll).
2. **Double-Entry Accounting (SAK-EMKM)**: Seeding COA otomatis, Jurnal Validasi (Debit=Kredit), Laporan Laba Rugi, Neraca, Arus Kas, dan bisa buat daftar pengeluaran beserta biayanya.
3. **Point of Sale (POS) & Dynamic QRIS Checkout**: Manajemen produk, import/export CSV, kasir mini, QRIS dinamis (CRC16-CCITT), webhook pembayaran Xendit.
4. **AI Conversational Accounting & WhatsApp Agent**: Chatbot ( Web), Redis Async Queue (100 workers), RAG prompt dinamis, input jurnal via chat (NLP), eskalasi manual ke admin
5. **Automation & Background Insights**: Analisis data berkala, laporan bulanan AI via email, whatsapps dan telegram.

### B. SaaS Crypto Trading Bot
1. **Dashboard & Analytics**: Tampilan harga kripto real-time, indikator teknikal, manajemen API key exchange (CRUD), registrasi/login pengguna.

### C. Campaign & Political Management (Aplikasi Pemenangan)
1. **Dashboard & Campaign**: Statistik kampanye, progress dukungan, timeline, multi-kandidat/periode.
2. **Candidate Verification**: Registrasi, upload dokumen, approval admin, status suspend/verifikasi.
3. **Volunteer Manager**: Registrasi relawan, struktur tim, assign wilayah, tracking aktivitas, leaderboard.
4. **Support Manager**: Input dan validasi data dukungan, tracking sumber, statistik.
5. **Region Manager**: Manajemen wilayah berjenjang (Provinsi hingga TPS), target wilayah.
6. **GIS & Map**: Peta sebaran dukungan (Leaflet/Mapbox), marker, heatmap, filter.
7. **Voter CRM**: Manajemen data pemilih, segmentasi, status relasi, catatan komunikasi.
8. **Survey & Polling**: Form builder, survey lapangan, polling internal, hasil survey.
9. **Event Manager**: Manajemen event, registrasi peserta, absensi QR, dokumentasi.
10. **Task & Operations**: Task management, assign tim, deadline, monitoring.
11. **Notification Center**: Notifikasi In-app, Email, Broadcast.
12. **User & Access**: Auth, RBAC, Multi-role, Audit log.
13. **Reports**: Dashboard laporan, export PDF/Excel, rekap per wilayah.

---

## 3. Detail Spesifikasi per Produk

---

### A. SaaS UMKM & Customer Service AI

#### 1. Multi-Tenant Onboarding & Dynamic Dashboard Widget
- **Sub-fitur**:
  - Onboarding instan dengan seleksi tipe bisnis (Warung, Laundry, Restoran, Hotel atau Penginapan, dll).
  - Dashboard widget dinamis yang menyesuaikan KPI metrik setiap tipe bisnis.
- **Use Cases / User Stories**:
  - *Sebagai pemilik Hotel/Penginapan, saya ingin dashboard saya menampilkan tingkat okupansi kamar dan pendapatan hari ini.*
- **Acceptance Criteria**:
  - Tampilan dashboard langsung berubah sesuai dengan tipe bisnis yang dipilih saat pendaftaran.
- **Prioritas & Catatan**: Live (MVP).

#### 2. Pembukuan & Laporan Keuangan (Double-Entry Accounting)
- **Sub-fitur**:
  - Generator Laporan Keuangan Otomatis (Laba Rugi, Neraca, Arus Kas).
  - Manajemen Chart of Accounts (COA) standar SAK-EMKM.
  - Pencatatan daftar pengeluaran beserta rincian biayanya.
  - Penjurnalan manual dan otomatis.
- **Use Cases / User Stories**:
  - *Sebagai pemilik UMKM, saya ingin mencatat daftar pengeluaran operasional beserta rincian biaya yang terperinci agar mudah dilacak.*
  - *Sebagai kasir, saya ingin setiap transaksi penjualan langsung tercatat di jurnal tanpa perlu input ulang.*
- **Acceptance Criteria**:
  - Jurnal tidak bisa disimpan jika Debit ≠ Kredit.
  - Form pencatatan pengeluaran harus bisa merincikan item biaya.
  - Laporan laba rugi dan arus kas dapat di-generate untuk rentang tanggal tertentu (PDF/Excel).
- **Prioritas & Catatan**: Live (MVP) - Peningkatan fitur daftar pengeluaran dan Arus Kas aktif.

#### 3. Order Management & Point of Sale (POS)
- **Sub-fitur**:
  - Manajemen Katalog (Stok, Harga Dasar, Foto).
  - Import/Export Katalog via CSV.
  - Checkout Multi-Payment (Cash & QRIS Dinamis).
  - Integrasi Webhook Gateway (Xendit).
- **Use Cases / User Stories**:
  - *Sebagai kasir, saya ingin generate QRIS dengan nominal pas agar pelanggan tinggal scan.*
  - *Sebagai admin toko, saya ingin update harga ratusan barang pakai CSV.*
- **Acceptance Criteria**:
  - QRIS yang digenerate harus lolos validasi checksum CRC16-CCITT.
  - Status pesanan otomatis berubah 'Paid' saat webhook Xendit diterima.
  - Stok otomatis berkurang saat pesanan 'Paid'.
- **Prioritas & Catatan**: Live (MVP).

#### 4. AI Customer Service & Conversational Accounting
- **Sub-fitur**:
  - Omni-Channel Chatbot (Web Widget).
  - Dynamic RAG (Menjawab stok/harga berdasar katalog terbaru).
  - Conversational Accounting (Pencatatan keuangan via NLP).
  - Smart Escalation (Auto-forward ke admin).
- **Use Cases / User Stories**:
  - *Sebagai pembeli, saya ingin tanya ketersediaan stok via Web Chatbot dan dibalas instan.*
  - *Sebagai owner, saya ingin chat "Saya bayar listrik 50rb" dan AI merespon "Tercatat: Beban Listrik (+), Kas (-)".*
  - *Sebagai pelanggan yang marah karena barang rusak, saya ingin chat saya langsung diteruskan ke manusia, bukan AI.*
- **Acceptance Criteria**:
  - AI merespon chat dalam < 5 detik menggunakan Redis async queue (100 workers).
  - Role-based prompting berfungsi: Pembeli tidak bisa tanya laba rugi toko.
  - Tag `[FORWARD_TO_ADMIN]` ter-trigger otomatis jika sentimen chat sangat negatif.
- **Prioritas & Catatan**: Live (MVP).

#### 5. Automation & Background Insights
- **Sub-fitur**:
  - Job Scheduler untuk analisis data keuangan berkala.
  - Pengiriman Laporan Bulanan AI secara otomatis.
  - Multi-channel notification (Email, WhatsApps, dan Telegram).
- **Use Cases / User Stories**:
  - *Sebagai pemilik bisnis, saya ingin menerima ringkasan performa penjualan dan laba bulan ini langsung ke WhatsApp atau Telegram saya.*
- **Acceptance Criteria**:
  - Cron job background berjalan lancar memicu AI MiniMax untuk menganalisis data.
  - Laporan terkirim sukses melalui Email, WhatsApps, dan Telegram sesuai pengaturan.
- **Prioritas & Catatan**: Live (MVP).

---

### B. SaaS Crypto Trading Bot

#### 1. Integrasi Multi-Exchange & Keamanan
- **Sub-fitur**:
  - Manajemen Kunci API Exchange (Binance, Tokocrypto, Indodax, Bybit).
  - Enkripsi AES-256 GCM untuk penyimpanan API Secret.
  - Dukungan arsitektur untuk tambah exchange baru dengan mudah.
- **Use Cases / User Stories**:
  - *Sebagai trader, saya ingin menghubungkan akun Binance saya tanpa takut saldo saya dicuri.*
  - *Sebagai user, saya ingin melihat saldo dari 2 exchange berbeda di 1 layar.*
- **Acceptance Criteria**:
  - API Secret tidak pernah dikembalikan ke Frontend/API Response (write-only view).
  - Koneksi CCXT berhasil menarik saldo (Read) tanpa error authorization.
- **Prioritas & Catatan**: Core Infra (P1).

#### 2. Bot Trading Otomatis (Grid, DCA, Signal)
- **Sub-fitur**:
  - Strategy Builder (Parameter upper/lower price, grids, investment amount).
  - Execution Worker (Background process Go, asinkron).
  - Monitoring Status Bot (Running, Paused, Stopped).
- **Use Cases / User Stories**:
  - *Sebagai trader pasif, saya ingin pasang DCA Bot agar tiap minggu otomatis beli BTC $10.*
  - *Sebagai trader swing, saya ingin pasang Grid Bot ETH/USDT di range $3000-$3500.*
- **Acceptance Criteria**:
  - Bot tereksekusi otomatis berdasarkan trigger waktu (DCA) atau pergerakan harga (Grid) menggunakan TimescaleDB & Redis.
  - Sistem sanggup menangani concurrent bot eksekusi tanpa *slippage* yang parah.
- **Prioritas & Catatan**: Utama (P1).

#### 3. Backtesting & Portfolio Analytics
- **Sub-fitur**:
  - Mesin Backtesting historis.
  - Dashboard PnL Real-time.
  - Visualisasi Chart (TradingView / Lightweight Charts).
- **Use Cases / User Stories**:
  - *Sebelum deposit besar, saya ingin tes strategi Grid Bot saya ke data harga bulan lalu.*
  - *Saya ingin lihat kurva keuntungan (Win Rate) portofolio saya bulan ini.*
- **Acceptance Criteria**:
  - Backtest selesai dijalankan di bawah 10 detik untuk data 1 bulan.
  - Dashboard PnL menampilkan total profit/loss dengan warna (Hijau/Merah) yang jelas.
- **Prioritas & Catatan**: Backtesting (Future/P2), Dashboard Analytics (MVP Live).

---

### C. Campaign & Political Management

#### 1. Manajemen Relawan & Tim Sukses
- **Sub-fitur**:
  - Hierarki Tim (Koordinator Provinsi -> TPS).
  - Tracker GPS Check-in & Kinerja.
  - Leaderboard Relawan & Gamification.
- **Use Cases / User Stories**:
  - *Sebagai Ketua Timses, saya ingin tahu relawan mana yang paling rajin sosialisasi hari ini.*
  - *Sebagai relawan TPS, saya ingin lapor hadir di lokasi blusukan menggunakan HP saya.*
- **Acceptance Criteria**:
  - Aplikasi mobile-friendly bisa menangkap koordinat GPS saat check-in.
  - Leaderboard ter-update real-time berdasarkan jumlah pemilih yang didata relawan.
- **Prioritas & Catatan**: Live (MVP).

#### 2. Voter CRM, GIS & Mengukur Elektabilitas
- **Sub-fitur**:
  - Survei Lapangan Mobile & Form Builder.
  - Database Pemilih & Sentimen.
  - Heatmap Demografis (Peta Dukungan Leaflet/Mapbox).
- **Use Cases / User Stories**:
  - *Sebagai kandidat, saya ingin lihat di kelurahan mana dukungan saya paling lemah.*
  - *Sebagai surveyor, saya butuh form yang bisa diisi offline dan otomatis sinkron saat online.*
- **Acceptance Criteria**:
  - Map menampilkan marker merah (lemah) dan hijau (kuat) berdasarkan persentase survei per wilayah.
  - Duplikasi NIK pemilih ditolak sistem.
- **Prioritas & Catatan**: Fitur Map Live (MVP), AI Predictive Analytics (Future).

#### 3. Mengawal Suara & Hasil Pemilu (Real Count)
- **Sub-fitur**:
  - Input Formulir C1 Plano.
  - Validasi Input Berjenjang (Saksi TPS -> Admin Posko).
  - AI OCR Deteksi C1 (Mencegah kecurangan input).
- **Use Cases / User Stories**:
  - *Sebagai saksi TPS, setelah hitung suara selesai, saya ingin foto papan C1 dan kirim angkanya agar posko pusat tahu.*
  - *Sebagai Admin, saya ingin sistem memperingatkan jika foto C1 tertulis "50" tapi saksi ketik "500".*
- **Acceptance Criteria**:
  - Form input wajib menyertakan bukti unggah foto.
  - (Future AI) MiniMax M2.7 Vision / OCR mengekstrak angka dari foto dan membandingkan dengan input manual saksi.
- **Prioritas & Catatan**: Input Manual Live (MVP), AI OCR Validation (High Priority Next).

---

## 4. Ketentuan Teknis Platform (Shared Services)
*(Fitur yang melayani ketiga produk di atas)*

- **Auth & Tenant Service**: SSO terpusat, RBAC, Isolasi data (UMKM A tidak bisa lihat data UMKM B).
- **Billing Service**: SaaS tiering (Lite, Pro, Enterprise), Kuota Usage, Integrasi Gateway, Buatkan Kupon untuk gratis 3 bulan untuk Lite. jaadi untuk promosi di berikan kupon 
- **AI Gateway**: Proksi sentral untuk LLM (MiniMax M2.7), Semantic Cache (Redis), Rate limiting agar biaya token terkontrol.
- **Workflow & Notifications**: Email transaksional, Blast WA, Telegram notification, n8n automation engine.

---
*Dokumen ini merupakan sumber kebenaran utama (Single Source of Truth) untuk pengembangan fitur WCH Platform.*
