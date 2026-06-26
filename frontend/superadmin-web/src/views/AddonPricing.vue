<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { request } from '../api/client'

interface AddonFeature {
  feature_key: string
  feature_name: string
  description: string
  category: string
  is_addon: boolean
  addon_price_rupiah: number
  addon_unit: string
  default_enabled: boolean
  is_active?: boolean
}

const addons = ref<AddonFeature[]>([])
const loading = ref(true)
const saving = ref<Record<string, boolean>>({})
const error = ref('')
const successMsg = ref('')

const UNITS = ['per_request', 'per_month', 'per_minute', 'per_session', 'per_user']

function formatPrice(rupiah: number): string {
  return 'Rp ' + rupiah.toLocaleString('id-ID')
}

function priceToRupiah(rupiah: number): number {
  return rupiah
}

function rupiahToRupiah(rp: number): number {
  return Math.round(rp)
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await request('/admin/available-features')
    addons.value = (res.data || []).filter((f: AddonFeature) => f.is_addon)
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function saveAddon(addon: AddonFeature) {
  saving.value[addon.feature_key] = true
  successMsg.value = ''
  try {
    await request(`/admin/available-features/${addon.feature_key}`, {
      method: 'PATCH',
      body: JSON.stringify({
        addon_price_rupiah: addon.addon_price_rupiah,
        addon_unit: addon.addon_unit,
        description: addon.description,
        default_enabled: addon.default_enabled,
      }),
    })
    successMsg.value = `${addon.feature_name} berhasil disimpan`
    setTimeout(() => successMsg.value = '', 3000)
  } catch (e: any) {
    error.value = 'Gagal simpan: ' + e.message
  } finally {
    saving.value[addon.feature_key] = false
  }
}

// Store rupiah display value separately
const priceDisplay = ref<Record<string, number>>({})

function initPriceDisplay() {
  addons.value.forEach(a => {
    priceDisplay.value[a.feature_key] = priceToRupiah(a.addon_price_rupiah)
  })
}

function onPriceInput(addon: AddonFeature, val: string) {
  const n = Number.parseFloat(val) || 0
  priceDisplay.value[addon.feature_key] = n
  addon.addon_price_rupiah = rupiahToRupiah(n)
}

onMounted(async () => {
  await load()
  initPriceDisplay()
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Addon Pricing</h1>
        <p class="subtitle">Atur harga dan konfigurasi setiap addon yang tersedia di marketplace tenant.</p>
      </div>
      <button class="btn-refresh" @click="load">🔄 Refresh</button>
    </div>

    <div v-if="successMsg" class="success-banner">✅ {{ successMsg }}</div>
    <div v-if="error" class="error-banner">⚠️ {{ error }}</div>

    <div v-if="loading" class="loading">Memuat daftar addon...</div>

    <div v-else-if="addons.length === 0" class="empty">
      Tidak ada addon yang terdaftar di <code>available_features</code>.
    </div>

    <div v-else class="addon-grid">
      <div v-for="addon in addons" :key="addon.feature_key" class="addon-card">
        <div class="addon-header">
          <div class="addon-meta">
            <h3>{{ addon.feature_name }}</h3>
            <code class="key">{{ addon.feature_key }}</code>
          </div>
          <span :class="['status-badge', addon.default_enabled ? 'active' : 'inactive']">
            {{ addon.default_enabled ? 'Aktif' : 'Nonaktif' }}
          </span>
        </div>

        <div class="addon-form">
          <div class="form-row">
            <label>
              Harga (Rp)
              <input
                type="number"
                :value="priceDisplay[addon.feature_key]"
                @input="onPriceInput(addon, ($event.target as HTMLInputElement).value)"
                min="0"
                step="1000"
                placeholder="0"
              />
              <span class="price-preview">= {{ formatPrice(addon.addon_price_rupiah) }}</span>
            </label>

            <label>
              Unit / Satuan
              <select v-model="addon.addon_unit">
                <option v-for="u in UNITS" :key="u" :value="u">{{ u }}</option>
              </select>
            </label>
          </div>

          <label>
            Deskripsi
            <textarea v-model="addon.description" rows="2" placeholder="Deskripsi singkat addon ini..."></textarea>
          </label>

          <div class="form-row align-center">
            <label class="toggle-label">
              <input type="checkbox" v-model="addon.default_enabled" class="checkbox" />
              <span>Aktif di Marketplace</span>
            </label>

            <button
              class="btn-save"
              @click="saveAddon(addon)"
              :disabled="saving[addon.feature_key]"
            >
              {{ saving[addon.feature_key] ? 'Menyimpan...' : '💾 Simpan' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { color: var(--muted); font-size: 14px; }

.btn-refresh {
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 13px;
}
.btn-refresh:hover { background: var(--border); }

.success-banner {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: var(--success);
  border-radius: 8px;
  padding: 10px 16px;
  margin-bottom: 16px;
  font-size: 14px;
}

.error-banner {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: var(--danger);
  border-radius: 8px;
  padding: 10px 16px;
  margin-bottom: 16px;
  font-size: 14px;
}

.loading, .empty {
  padding: 60px 20px;
  text-align: center;
  color: var(--muted);
  background: var(--card);
  border-radius: 12px;
  border: 1px dashed var(--border);
}

.addon-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(480px, 1fr));
  gap: 20px;
}

.addon-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 20px;
  transition: border-color 0.2s;
}
.addon-card:hover { border-color: var(--accent); }

.addon-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}

.addon-meta h3 { font-size: 15px; font-weight: 600; margin-bottom: 4px; }
.key { font-size: 11px; color: var(--muted); }

.status-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 20px;
  text-transform: uppercase;
  white-space: nowrap;
}
.status-badge.active { background: rgba(16, 185, 129, 0.15); color: var(--success); }
.status-badge.inactive { background: rgba(148, 163, 184, 0.15); color: var(--muted); }

.addon-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.form-row.align-center { align-items: center; }

.addon-form label {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 12px;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.addon-form input[type="number"],
.addon-form select,
.addon-form textarea {
  font-size: 14px;
  width: 100%;
}

.price-preview {
  font-size: 11px;
  color: var(--success);
  font-weight: 600;
  text-transform: none;
  letter-spacing: 0;
}

.toggle-label {
  flex-direction: row !important;
  align-items: center;
  gap: 8px !important;
  text-transform: none !important;
  letter-spacing: 0 !important;
  font-size: 14px !important;
  color: var(--text) !important;
  cursor: pointer;
}

.checkbox {
  width: 16px;
  height: 16px;
  accent-color: var(--accent);
}

.btn-save {
  background: var(--accent);
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  justify-self: end;
}
.btn-save:hover:not(:disabled) { background: #2563eb; }
</style>
