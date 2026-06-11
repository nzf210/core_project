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
    ],
  },
  {
    group: 'Keuangan',
    items: [
      { label: 'Jurnal Keuangan', to: '/journal', icon: '📒' },
    ],
  },
  {
    group: 'Sistem',
    items: [
      { label: 'Automasi', to: '/automations', icon: '⚡' },
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