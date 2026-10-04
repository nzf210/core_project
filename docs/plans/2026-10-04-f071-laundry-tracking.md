# Implementation Plan: F071 Modular Business Workflow — Laundry Order & Wash Tracking

**Date:** 2026-10-04  
**Feature ID:** F071  
**Target Module:** UMKM (`apps/umkm/accounting`, `frontend/umkm-web`)  
**Status:** Waiting for User Approval  

---

## 🎯 Goal
Menyediakan fitur spesialisasi operasional untuk tenant UMKM berjenis usaha Laundry (`tenants.business_type = 'laundry'`). Fitur mencakup penerimaan order cucian (kiloan/satuan), pelacakan progres cucian (Diterima ➔ Cuci ➔ Kering ➔ Setrika ➔ Siap Ambil ➔ Selesai), nomor rak/keranjang, auto-notifikasi WhatsApp saat siap diambil, dan integrasi kasir/pembayaran.

---

## 🏗️ Architecture & Component Boundaries
- **Database:** Migrasi `000088_laundry_orders.up.sql` mendefinisikan tabel `laundry_orders` dengan indeks multi-tenant.
- **Backend (`apps/umkm/accounting`):**
  - `laundry_middleware.go`: Gating akses hanya untuk tenant laundry (`requireLaundryType`).
  - `laundry_handlers.go`: HTTP REST handler (`listLaundryOrders`, `createLaundryOrder`, `updateLaundryOrderStatus`, `payLaundryOrder`). Maksimal 450 baris kode (SonarQube compliant).
  - `laundry_test.go`: Unit tests mencakup validasi input, transisi status yang valid & tidak valid, dan isolasi tenant.
- **Frontend (`frontend/umkm-web`):**
  - `src/config/menu.ts`: Registrasi menu `Tracking Laundry` dengan `businessTypes: ['laundry']`.
  - `src/api.ts`: Helper API client untuk laundry (`laundryApi.getOrders`, `laundryApi.createOrder`, `laundryApi.updateStatus`, dll).
  - `src/components/LaundryTracking.vue`: Kanban board status cucian & modal penerimaan order. Maksimal 500 baris kode.
  - `src/router/index.ts`: Registrasi route `/laundry/tracking`.

---

## 📋 Step-by-Step Execution Plan

### Step 1: Database Migration
- [ ] Buat file `shared/migrations/000088_laundry_orders.up.sql`:
  - `CREATE TABLE IF NOT EXISTS laundry_orders (...)`
  - Kolom: `id`, `tenant_id`, `order_no`, `customer_name`, `customer_phone`, `service_type`, `weight_grams`, `item_count`, `rack_location`, `status`, `total_amount`, `is_paid`, `payment_method`, `estimated_completion_at`, `completed_at`, `notes`, `created_at`, `updated_at`.
  - Indeks pada `(tenant_id, status)`, `(tenant_id, customer_phone)`, dan `(tenant_id, order_no)`.
- [ ] Buat file `shared/migrations/000088_laundry_orders.down.sql`:
  - `DROP TABLE IF EXISTS laundry_orders;`

### Step 2: Backend Middleware & Handlers (Go)
- [ ] Buat `apps/umkm/accounting/laundry_middleware.go`:
  - `func requireLaundryType(next http.HandlerFunc) http.HandlerFunc` yang memverifikasi `SELECT business_type FROM tenants WHERE id = $1` bernilai `'laundry'`.
- [ ] Buat `apps/umkm/accounting/laundry_handlers.go`:
  - Struct `LaundryOrder` dan request binding DTOs.
  - Helper generate order number unik per tenant (misal `LND-YYYYMMDD-XXXX`).
  - Handler `handleLaundryOrders`: GET (list + filter + search), POST (buat order baru).
  - Handler `handleLaundryOrderStatus`: PATCH (update status state machine).
  - Integrasi WhatsApp: ketika status berubah ke `ready`, kirim notifikasi pengambilan ke `customer_phone` via wa-gateway / helper notifikasi.
  - Handler `handleLaundryOrderPay`: POST (tandai lunas & catat jurnal kasir).
- [ ] Daftarkan routes di `apps/umkm/accounting/main.go`:
  - `/api/umkm/laundry/orders`
  - `/api/umkm/laundry/orders/`
- [ ] Buat unit tests di `apps/umkm/accounting/laundry_test.go`:
  - `TestRequireLaundryType`
  - `TestValidateLaundryOrderInput`
  - `TestLaundryStatusTransitions`

### Step 3: Frontend API & Routing (Vue 3 + TypeScript)
- [ ] Update `frontend/umkm-web/src/api.ts`:
  - Tambahkan type definition `LaundryOrder` dan object `laundryApi`.
- [ ] Update `frontend/umkm-web/src/config/menu.ts`:
  - Tambahkan menu `{ label: 'Tracking Laundry', to: '/laundry/tracking', icon: '🧺', businessTypes: ['laundry'], roles: ['owner', 'admin', 'staff', 'kasir'] }`.
- [ ] Update `frontend/umkm-web/src/router/index.ts`:
  - Tambahkan route `/laundry/tracking` ke komponen `LaundryTracking.vue`.

### Step 4: Frontend UI (LaundryTracking.vue)
- [ ] Buat `frontend/umkm-web/src/components/LaundryTracking.vue`:
  - Layout Kanban atau Tab Status:
    - `Antrean (received)`
    - `Dicuci (washing)`
    - `Pengeringan (drying)`
    - `Setrika/Packing (ironing)`
    - `Siap Diambil (ready)`
    - `Selesai (completed)`
  - Tombol aksi cepat untuk memindahkan status cucian ke tahap berikutnya.
  - Modal form penerimaan cucian baru:
    - Nama & No WhatsApp pelanggan
    - Pilihan jenis layanan (Kiloan / Satuan / Dry Clean)
    - Berat (kg) / Jumlah potong
    - Estimasi selesai & Lokasi rak/keranjang
    - Total biaya (Rupiah) & Status bayar langsung/nanti
  - Kartu order menampilkan: Nomor nota, nama pelanggan, jenis layanan, rak, status pembayaran, dan tombol chat WA / kirim info siap ambil.

### Step 5: Verification & Quality Gate
- [ ] Jalankan unit test: `go test ./apps/umkm/accounting/ -v -run "TestLaundry"`
- [ ] Jalankan full check: `make check` (go vet + build + all tests)
- [ ] Verifikasi batas SonarQube: backend Go <= 450 baris, Vue <= 500 baris.
- [ ] Update status di `docs/FEATURE_MAP.md` dari Draft ➔ Approved ➔ Done.
- [ ] Perbarui knowledge graph: `uv tool run --from graphifyy graphify update .`.
