export interface MenuItem {
  label: string
  to: string
  icon: string
  roles?: string[]
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
      { label: 'Antrean Klinik', to: '/clinic/frontdesk', icon: '🏥', roles: ['owner', 'admin', 'superadmin'] },
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
      { label: 'AI Customer Service', to: '/chatbot-config', icon: '🤖', roles: ['owner', 'admin', 'superadmin'] },
      { label: 'Agen Afiliasi', to: '/affiliate', icon: '🤝' },
      { label: 'Pengaturan', to: '/settings', icon: '⚙️' },
    ],
  },
  {
    group: 'Admin',
    items: [
      { label: 'Super Admin', to: '/superadmin', icon: '🔐', roles: ['admin', 'superadmin'] },
    ],
  },
]