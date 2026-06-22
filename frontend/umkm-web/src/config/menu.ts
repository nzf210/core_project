export interface MenuItem {
  label: string
  to: string
  icon: string
  roles?: string[]
  /**
   * businessTypes: jika di-set, menu HANYA tampil untuk business_type tertentu.
   * - undefined / kosong = tampil untuk semua jenis usaha
   * - ['clinic'] = tampil hanya untuk klinik
   * - ['clinic', 'restoran'] = tampil untuk klinik & restoran
   */
  businessTypes?: string[]
}

export interface MenuGroup {
  group: string
  items: MenuItem[]
}

export const menuConfig: MenuGroup[] = [
  {
    group: 'Operasi',
    items: [
      { label: 'Dashboard', to: '/', icon: '📊' },
      { label: 'Kasir/POS', to: '/pos', icon: '💰' },
      { label: 'Katalog Produk', to: '/catalog', icon: '📦' },
      // Impor/Ekspor: admin-only (risiko data besar; kasir tidak perlu)
      { label: 'Impor / Ekspor', to: '/data-transfer', icon: '📥', roles: ['owner', 'admin', 'superadmin'] },
      // ===== Klinik-only modules (F047) =====
      { label: 'Antrean Klinik', to: '/clinic/frontdesk', icon: '🏥', businessTypes: ['clinic'], roles: ['owner', 'admin', 'superadmin'] },
      { label: 'Rekam Medis', to: '/clinic/medical-record', icon: '📋', businessTypes: ['clinic'], roles: ['owner', 'admin', 'superadmin'] },
      { label: 'Jadwal Dokter', to: '/clinic/schedule', icon: '📅', businessTypes: ['clinic'], roles: ['owner', 'admin', 'superadmin'] },
      { label: 'Notifikasi WA Klinik', to: '/clinic/notifications', icon: '📲', businessTypes: ['clinic'], roles: ['owner', 'admin', 'superadmin'] },
    ],
  },
  {
    group: 'Keuangan',
    items: [
      { label: 'Jurnal Keuangan', to: '/journal', icon: '📒' },
      { label: 'Laporan Keuangan', to: '/reports', icon: '📊' },
    ],
  },
  {
    group: 'Sistem',
    items: [
      // Automasi & AI CS: owner-only (advanced config; kasir tidak perlu akses)
      { label: 'Automasi', to: '/automations', icon: '⚡', roles: ['owner', 'admin', 'superadmin'] },
      { label: 'WhatsApp & AI CS', to: '/wa-setup', icon: '🤖', roles: ['owner', 'admin', 'superadmin'] },
      { label: 'Agen Afiliasi', to: '/affiliate', icon: '🤝' },
      { label: 'Pengaturan', to: '/settings', icon: '⚙️' },
      { label: 'Wallet', to: '/wallet', icon: '💳', roles: ['owner', 'admin', 'superadmin'] },
      { label: 'Add-ons', to: '/addons', icon: '⚙️' },
    ],
  },
  {
    group: 'Admin',
    items: [
      { label: 'Super Admin', to: '/superadmin', icon: '🔐', roles: ['superadmin'] },
    ],
  },
]
