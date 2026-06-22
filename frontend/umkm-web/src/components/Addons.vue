<template>
  <div class="addons-page">
    <!-- Page Header -->
    <div class="page-header">
      <div class="header-left">
        <h2 class="page-title">Add-ons</h2>
        <p class="page-subtitle">Perluas kemampuan toko dengan fitur AI dan WhatsApp premium</p>
      </div>
      <div class="wallet-chip" @click="$router.push('/wallet')">
        <span class="wallet-icon">💰</span>
        <span class="wallet-label">Wallet</span>
        <span class="wallet-balance">Rp {{ formattedWalletBalance }}</span>
      </div>
    </div>

    <!-- Tab Navigation -->
    <div class="tab-nav">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['tab-btn', { active: activeTab === tab.key }]"
        @click="activeTab = tab.key"
      >
        <span class="tab-icon">{{ tab.icon }}</span>
        <span class="tab-label">{{ tab.label }}</span>
        <span class="tab-count">{{ tab.count }}</span>
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-grid">
      <div v-for="n in 4" :key="n" class="addon-card skeleton"></div>
    </div>

    <!-- Error State -->
    <div v-else-if="loadError" class="empty-state">
      <div class="empty-icon">⚠️</div>
      <p>{{ loadError }}</p>
      <button class="btn btn-secondary" @click="loadMarketplace">Coba Lagi</button>
    </div>

    <!-- Empty Tab -->
    <div v-else-if="filteredAddons.length === 0" class="empty-state">
      <div class="empty-icon">📦</div>
      <p>Tidak ada add-on di kategori ini</p>
    </div>

    <!-- Addon Grid -->
    <div v-else class="addon-grid">
      <div
        v-for="(addon, idx) in filteredAddons"
        :key="addon.addon_key"
        :class="['addon-card', { active: addon.has_addon }]"
        :style="{ animationDelay: `${idx * 60}ms` }"
      >
        <!-- Active Badge -->
        <div v-if="addon.has_addon" class="active-badge">
          <span class="badge-dot"></span>
          Aktif
        </div>

        <!-- Category Icon -->
        <div class="addon-icon">
          {{ getAddonIcon(addon.addon_key) }}
        </div>

        <!-- Info -->
        <div class="addon-info">
          <h3 class="addon-name">{{ addon.feature_name }}</h3>
          <p class="addon-desc">{{ addon.description || 'Deskripsi tidak tersedia' }}</p>
        </div>

        <!-- Pricing -->
        <div class="addon-pricing">
          <div class="price-main">
            <span class="price-currency">Rp</span>
            <span class="price-value">{{ formatRupiah(addon.price_cents) }}</span>
          </div>
          <div class="price-unit">{{ formatUnit(addon.addon_unit) }}</div>
          <div v-if="addon.has_addon && addon.expires_at" class="expires-info">
            Aktif hingga {{ formatDate(addon.expires_at) }}
          </div>
        </div>

        <!-- Action -->
        <div class="addon-action">
          <template v-if="addon.has_addon">
            <button class="btn btn-ghost btn-sm" disabled>
              ✓ Sudah Dimiliki
            </button>
          </template>
          <template v-else-if="purchasing === addon.addon_key">
            <button class="btn btn-primary btn-sm" disabled>
              <span class="spinner"></span>
              Memproses...
            </button>
          </template>
          <template v-else>
            <button
              class="btn btn-primary btn-sm"
              @click="handlePurchase(addon)"
            >
              Beli Add-on
            </button>
          </template>
        </div>

        <!-- Purchase Success Overlay -->
        <div v-if="justPurchased === addon.addon_key" class="purchase-success">
          <div class="success-checkmark">✓</div>
          <p>Berhasil dibeli!</p>
        </div>
      </div>
    </div>

    <!-- Confirmation Modal -->
    <Teleport to="body">
      <div v-if="confirmAddon" class="modal-overlay" @click.self="confirmAddon = null">
        <div class="modal-content confirm-modal">
          <div class="modal-icon">{{ getAddonIcon(confirmAddon.addon_key) }}</div>
          <h3>Beli Add-on</h3>
          <div class="confirm-detail">
            <span class="confirm-name">{{ confirmAddon.feature_name }}</span>
            <span class="confirm-price">Rp {{ formatRupiah(confirmAddon.price_cents) }}</span>
          </div>
          <p class="confirm-subtitle">
            Saldo wallet Anda: <strong>Rp {{ formattedWalletBalance }}</strong>
          </p>
          <div v-if="walletBalanceCents < confirmAddon.price_cents" class="balance-warning">
            ⚠️ Saldo tidak cukup.
            <button class="link-btn" @click="$router.push('/wallet'); confirmAddon = null">
              Top-up wallet →
            </button>
          </div>
          <div v-else class="confirm-note">
            Saldo akan langsung deducted dari wallet Anda.
          </div>
          <div class="modal-actions">
            <button class="btn btn-ghost" @click="confirmAddon = null">Batal</button>
            <button
              v-if="walletBalanceCents >= confirmAddon.price_cents"
              class="btn btn-primary"
              @click="executePurchase"
              :disabled="purchasing === confirmAddon.addon_key"
            >
              {{ purchasing === confirmAddon.addon_key ? 'Memproses...' : `Beli — Rp ${formatRupiah(confirmAddon.price_cents)}` }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'

interface AddonItem {
  addon_key: string
  feature_name: string
  description: string
  category: string
  price_cents: number
  addon_unit: string
  has_addon: boolean
  addon_status?: string
  expires_at?: string
  purchase_price_cents?: number
}

const marketplace = ref<AddonItem[]>([])
const loading = ref(true)
const loadError = ref('')
const purchasing = ref('')
const justPurchased = ref('')
const confirmAddon = ref<AddonItem | null>(null)
const walletBalance = ref(0)

const activeTab = ref('all')

const tabs = computed(() => [
  { key: 'all', label: 'Semua', icon: '⊞', count: marketplace.value.length },
  { key: 'ai', label: 'AI', icon: '🤖', count: marketplace.value.filter(a => a.category === 'ai').length },
  { key: 'wa', label: 'WhatsApp', icon: '📱', count: marketplace.value.filter(a => a.category === 'wa').length },
  { key: 'storage', label: 'Storage', icon: '💾', count: marketplace.value.filter(a => a.category === 'storage').length },
  { key: 'user', label: 'User', icon: '👤', count: marketplace.value.filter(a => a.category === 'user').length },
])

const filteredAddons = computed(() =>
  activeTab.value === 'all'
    ? marketplace.value
    : marketplace.value.filter(a => a.category === activeTab.value)
)

const formattedWalletBalance = computed(() =>
  (walletBalance.value / 100).toLocaleString('id-ID')
)
const walletBalanceCents = computed(() => walletBalance.value)

const getAddonIcon = (key: string): string => {
  const map: Record<string, string> = {
    ai_vision: '👁️',
    ai_audio: '🎙️',
    wa_blast: '📣',
    wa_meta_session: '☁️',
    extra_store: '🏪',
    extra_user: '👥',
  }
  return map[key] || '⚙️'
}

const formatRupiah = (cents: number): string =>
  (cents / 100).toLocaleString('id-ID')

const formatUnit = (unit: string): string => {
  const map: Record<string, string> = {
    per_request: '/request',
    per_minute: '/menit',
    per_session: '/session',
    per_month: '/bulan',
    per_user: '/user',
    per_store: '/store',
    once: 'sekali',
  }
  return map[unit] || unit
}

const formatDate = (iso: string): string => {
  try {
    return new Date(iso).toLocaleDateString('id-ID', {
      day: 'numeric', month: 'short', year: 'numeric'
    })
  } catch {
    return iso
  }
}

const loadMarketplace = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await fetch(`${import.meta.env.VITE_API_BASE || 'http://localhost:8010'}/api/umkm/addon-marketplace`, {
      headers: {
        Authorization: `Bearer ${localStorage.getItem('access_token')}`,
        'X-Tenant-ID': localStorage.getItem('tenant_id') || '',
      }
    })
    const data = await res.json()
    if (data.success && data.data && Array.isArray(data.data.addons)) {
      marketplace.value = data.data.addons
    } else {
      marketplace.value = []
    }
  } catch (e: any) {
    loadError.value = 'Gagal memuat marketplace. Pastikan koneksi stabil.'
  } finally {
    loading.value = false
  }
}

const loadWallet = async () => {
  try {
    const res = await api.getWallet()
    if (res.success && res.data) {
      walletBalance.value = res.data.balance_cents || 0
    }
  } catch { /* silent */ }
}

const handlePurchase = (addon: AddonItem) => {
  if (walletBalance.value === 0) {
    loadWallet()
  }
  confirmAddon.value = addon
}

const executePurchase = async () => {
  if (!confirmAddon.value) return
  purchasing.value = confirmAddon.value.addon_key
  try {
    const res = await api.purchaseAddon(confirmAddon.value.addon_key)
    if (res.success) {
      justPurchased.value = confirmAddon.value.addon_key
      marketplace.value = marketplace.value.map(a =>
        a.addon_key === confirmAddon.value!.addon_key
          ? { ...a, has_addon: true, addon_status: 'active' }
          : a
      )
      confirmAddon.value = null
      setTimeout(() => { justPurchased.value = '' }, 3000)
      loadWallet()
    } else {
      alert(res.message || 'Pembelian gagal')
    }
  } catch (e: any) {
    alert(e.message || 'Terjadi kesalahan')
  } finally {
    purchasing.value = ''
  }
}

onMounted(() => {
  loadMarketplace()
  loadWallet()
})
</script>

<style scoped>
.addons-page {
  max-width: 1100px;
  margin: 0 auto;
}

/* ── Header ─────────────────────────────── */
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 2rem;
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1.1;
  background: linear-gradient(135deg, var(--text-primary), var(--accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.page-subtitle {
  color: var(--text-muted);
  font-size: 0.9rem;
  margin-top: 0.25rem;
}

.wallet-chip {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: var(--surface-1);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-full);
  padding: 0.5rem 1rem;
  cursor: pointer;
  font-size: 0.875rem;
  transition: border-color 0.2s, background 0.2s;
  flex-shrink: 0;
}
.wallet-chip:hover {
  border-color: var(--accent);
  background: var(--accent-subtle);
}
.wallet-icon { font-size: 1rem; }
.wallet-label { color: var(--text-muted); }
.wallet-balance { font-weight: 600; color: var(--text-primary); }

/* ── Tabs ────────────────────────────────── */
.tab-nav {
  display: flex;
  gap: 0.375rem;
  margin-bottom: 1.75rem;
  overflow-x: auto;
  padding-bottom: 0.25rem;
  scrollbar-width: none;
}
.tab-nav::-webkit-scrollbar { display: none; }

.tab-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.5rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-muted);
  font-size: 0.875rem;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}
.tab-btn:hover { border-color: var(--accent); color: var(--text-primary); }
.tab-btn.active {
  background: var(--accent-subtle);
  border-color: var(--accent);
  color: var(--accent);
}
.tab-icon { font-size: 0.9rem; }
.tab-count {
  background: var(--surface-2);
  border-radius: var(--radius-full);
  padding: 0 0.4rem;
  font-size: 0.7rem;
  color: var(--text-muted);
}
.tab-btn.active .tab-count { background: var(--accent); color: white; }

/* ── Grid ───────────────────────────────── */
.addon-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}

/* ── Card ───────────────────────────────── */
.addon-card {
  position: relative;
  background: var(--surface-1);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-xl);
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.875rem;
  animation: cardIn 0.4s ease-out both;
  transition: border-color 0.2s, transform 0.2s, box-shadow 0.2s;
  overflow: hidden;
}
.addon-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, var(--accent-subtle) 0%, transparent 60%);
  opacity: 0;
  transition: opacity 0.3s;
  border-radius: inherit;
}
.addon-card:hover {
  border-color: var(--accent);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(99, 102, 241, 0.12);
}
.addon-card:hover::before { opacity: 1; }
.addon-card.active { border-color: rgba(16, 185, 129, 0.4); }
.addon-card.active:hover { border-color: var(--success); box-shadow: 0 8px 24px rgba(16, 185, 129, 0.1); }

@keyframes cardIn {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* ── Active Badge ───────────────────────── */
.active-badge {
  position: absolute;
  top: 1rem;
  right: 1rem;
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--success);
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.25);
  border-radius: var(--radius-full);
  padding: 0.2rem 0.6rem;
}
.badge-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--success);
  animation: pulse 2s infinite;
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

/* ── Icon ───────────────────────────────── */
.addon-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  background: var(--surface-2);
  border: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.4rem;
  flex-shrink: 0;
}

/* ── Info ───────────────────────────────── */
.addon-info { flex: 1; }
.addon-name { font-size: 1rem; font-weight: 600; color: var(--text-primary); }
.addon-desc {
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ── Pricing ─────────────────────────────── */
.addon-pricing {
  padding: 0.75rem;
  background: var(--surface-0);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
}
.price-main { display: flex; align-items: baseline; gap: 0.2rem; }
.price-currency { font-size: 0.8rem; font-weight: 500; color: var(--text-muted); }
.price-value { font-size: 1.25rem; font-weight: 700; color: var(--text-primary); }
.price-unit { font-size: 0.75rem; color: var(--text-muted); margin-top: 0.1rem; }
.expires-info { font-size: 0.72rem; color: var(--success); margin-top: 0.3rem; }

/* ── Action ──────────────────────────────── */
.addon-action { margin-top: auto; }
.btn-sm { padding: 0.5rem 1rem; font-size: 0.875rem; width: 100%; }
.btn-ghost {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-muted);
  border-radius: var(--radius-md);
  font-family: inherit;
  cursor: not-allowed;
}

/* ── Success Overlay ─────────────────────── */
.purchase-success {
  position: absolute;
  inset: 0;
  background: rgba(16, 185, 129, 0.95);
  border-radius: inherit;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  animation: fadeIn 0.3s ease;
}
.purchase-success .success-checkmark {
  font-size: 2.5rem;
  font-weight: 700;
  color: white;
  animation: scaleIn 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}
.purchase-success p { color: white; font-weight: 600; font-size: 0.9rem; }
@keyframes scaleIn {
  from { transform: scale(0); opacity: 0; }
  to   { transform: scale(1); opacity: 1; }
}
@keyframes fadeIn {
  from { opacity: 0; }
  to   { opacity: 1; }
}

/* ── Skeleton ───────────────────────────── */
.loading-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1rem;
}
.skeleton {
  height: 260px;
  background: linear-gradient(90deg, var(--surface-1) 25%, var(--surface-2) 50%, var(--surface-1) 75%);
  background-size: 200% 100%;
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-color);
  animation: shimmer 1.5s infinite;
}
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* ── Empty ──────────────────────────────── */
.empty-state {
  text-align: center;
  padding: 4rem 2rem;
  color: var(--text-muted);
}
.empty-icon { font-size: 3rem; margin-bottom: 1rem; }
.empty-state p { font-size: 0.95rem; margin-bottom: 1rem; }

/* ── Confirm Modal ──────────────────────── */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  animation: fadeIn 0.2s ease;
}
.modal-content {
  background: var(--surface-1);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-xl);
  padding: 2rem;
  max-width: 420px;
  width: 90%;
}
.confirm-modal { text-align: center; }
.modal-icon { font-size: 3rem; margin-bottom: 0.75rem; }
.confirm-modal h3 { font-size: 1.25rem; font-weight: 700; margin-bottom: 1rem; }
.confirm-detail {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--surface-0);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 0.75rem 1rem;
  margin-bottom: 0.75rem;
}
.confirm-name { font-weight: 600; }
.confirm-price { font-weight: 700; color: var(--accent); }
.confirm-subtitle { font-size: 0.85rem; color: var(--text-muted); margin-bottom: 0.5rem; }
.balance-warning {
  font-size: 0.85rem;
  color: var(--warning);
  background: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
  border-radius: var(--radius-md);
  padding: 0.75rem;
  margin: 0.5rem 0;
}
.link-btn {
  background: none;
  border: none;
  color: var(--warning);
  text-decoration: underline;
  font-family: inherit;
  font-size: inherit;
  cursor: pointer;
  padding: 0;
}
.confirm-note { font-size: 0.8rem; color: var(--text-muted); margin: 0.5rem 0; }
.modal-actions {
  display: flex;
  gap: 0.75rem;
  margin-top: 1.25rem;
}
.modal-actions .btn { flex: 1; }
.btn-primary {
  background: var(--accent);
  border: none;
  color: white;
  border-radius: var(--radius-md);
  font-family: inherit;
  font-weight: 600;
  cursor: pointer;
  padding: 0.625rem 1.25rem;
  transition: background 0.2s, transform 0.1s;
}
.btn-primary:hover:not(:disabled) { background: var(--accent-hover); }
.btn-primary:active:not(:disabled) { transform: scale(0.98); }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-secondary {
  background: var(--surface-2);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: var(--radius-md);
  font-family: inherit;
  font-weight: 500;
  cursor: pointer;
  padding: 0.625rem 1.25rem;
  transition: background 0.2s;
}
.btn-secondary:hover { background: var(--surface-3); }

/* ── Spinner ─────────────────────────────── */
.spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  margin-right: 0.4rem;
  vertical-align: middle;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
