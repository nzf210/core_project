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
            <span class="price-value">{{ formatRupiah(addon.price_rupiah) }}</span>
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
            <span class="confirm-price">{{ formatRupiah(confirmAddon.price_rupiah) }}</span>
          </div>
          <p class="confirm-subtitle">
            Saldo wallet Anda: <strong>Rp {{ formattedWalletBalance }}</strong>
          </p>
          <div v-if="walletBalanceRupiah < confirmAddon.price_rupiah" class="balance-warning">
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
              v-if="walletBalanceRupiah >= confirmAddon.price_rupiah"
              class="btn btn-primary"
              @click="executePurchase"
              :disabled="purchasing === confirmAddon.addon_key"
            >
              {{ purchasing === confirmAddon.addon_key ? 'Memproses...' : `Beli — ${formatRupiah(confirmAddon.price_rupiah)}` }}
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
import { formatRupiah } from '../composables/useCurrency'

interface AddonItem {
  addon_key: string
  feature_name: string
  description: string
  category: string
  price_rupiah: number
  addon_unit: string
  has_addon: boolean
  addon_status?: string
  expires_at?: string
  purchase_price_rupiah?: number
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
  formatRupiah(walletBalance.value)
)
const walletBalanceRupiah = computed(() => walletBalance.value)

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
    const res = await fetch(`${import.meta.env.VITE_API_URL || 'http://localhost:8000'}/api/umkm/addon-marketplace`, {
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
  } catch {
    loadError.value = 'Gagal memuat marketplace. Pastikan koneksi stabil.'
  } finally {
    loading.value = false
  }
}

const loadWallet = async () => {
  try {
    const res = await api.getWallet()
    if (res.success && res.data) {
      walletBalance.value = res.data.balance_rupiah || 0
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
  } catch (e) {
    alert(e instanceof Error ? e.message : 'Terjadi kesalahan')
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
@import './Addons.css';

</style>
