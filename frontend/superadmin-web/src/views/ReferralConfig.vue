<template>
  <div class="page">
    <h1>🎯 Konfigurasi Referral</h1>
    <p class="desc">Atur persentase diskon untuk user yang pakai kode referral, dan komisi untuk agen afiliasi.</p>

    <div v-if="loading" class="spinner">⏳ Memuat...</div>

    <form v-else @submit.prevent="handleSave" class="form-card">
      <div class="form-grid">
        <!-- Discount % -->
        <div class="field">
          <label>Diskon untuk Downline (%)</label>
          <input
            v-model.number="config.discount_percent"
            type="number"
            min="0" max="100" step="0.5"
            class="input"
            placeholder="10"
          />
          <span class="hint">Potongan harga setiap kali downline bayar (subscription, addon, campaign)</span>
        </div>

        <!-- Commission % -->
        <div class="field">
          <label>Komisi untuk Upline (%)</label>
          <input
            v-model.number="config.commission_percent"
            type="number"
            min="0" max="100" step="0.5"
            class="input"
            placeholder="10"
          />
          <span class="hint">Persentase komisi ke agen dari setiap pembayaran downline (lifetime)</span>
        </div>

        <!-- Min Purchase -->
        <div class="field">
          <label>Min. Pembelian untuk Komisi (Rp)</label>
          <input
            v-model.number="minPurchaseRp"
            type="number"
            min="0" step="1000"
            class="input"
            placeholder="0"
            @input="config.min_purchase_cents = minPurchaseRp * 100"
          />
          <span class="hint">Transaksi di bawah nilai ini tidak menghasilkan komisi (0 = semua)</span>
        </div>

        <!-- Max Commission -->
        <div class="field">
          <label>Max. Komisi per Transaksi (Rp)</label>
          <input
            v-model.number="maxCommissionRp"
            type="number"
            min="0" step="1000"
            class="input"
            placeholder="0 = unlimited"
            @input="config.max_commission_cents = maxCommissionRp * 100"
          />
          <span class="hint">Batas maksimal komisi per transaksi (0 = unlimited)</span>
        </div>
      </div>

      <!-- Is Active toggle -->
      <div class="toggle-row">
        <label class="toggle-label">
          <input type="checkbox" v-model="config.is_active" class="checkbox" />
          <span class="toggle-text">Aktifkan Sistem Referral</span>
        </label>
        <span class="toggle-hint">{{ config.is_active ? '✅ Sistem referral aktif' : '⏸️ Sistem referral dinonaktifkan' }}</span>
      </div>

      <!-- Preview -->
      <div class="preview-box">
        <div class="preview-title">📊 Preview Kalkulasi</div>
        <div class="preview-row">
          <span class="preview-label">Downline beli paket Rp 100.000</span>
          <span></span>
        </div>
        <div class="preview-row">
          <span>→ Dapat potongan:</span>
          <span class="preview-value discount">Rp {{ formatRp(100000 * config.discount_percent / 100) }}</span>
        </div>
        <div class="preview-row">
          <span>→ Harga akhir yang dibayar:</span>
          <span class="preview-value">Rp {{ formatRp(100000 - 100000 * config.discount_percent / 100) }}</span>
        </div>
        <div class="preview-row">
          <span>→ Upline dapat komisi:</span>
          <span class="preview-value commission">
            Rp {{ formatRp(Math.min(
              (100000 - 100000 * config.discount_percent / 100) * config.commission_percent / 100,
              config.max_commission_cents > 0 ? config.max_commission_cents : Infinity
            )) }}
            <span v-if="config.max_commission_cents > 0" class="capped">(capped Rp {{ formatRp(config.max_commission_cents) }})</span>
          </span>
        </div>
      </div>

      <div class="actions">
        <p v-if="errorMsg" class="error">{{ errorMsg }}</p>
        <p v-if="successMsg" class="success">{{ successMsg }}</p>
        <button type="submit" class="btn-save" :disabled="saving">
          {{ saving ? '⏳ Menyimpan...' : '💾 Simpan Konfigurasi' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { request } from '../api/client'

const loading = ref(true)
const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

interface ReferralConfig {
  discount_percent: number
  commission_percent: number
  min_purchase_cents: number
  max_commission_cents: number
  is_active: boolean
}

const config = ref<ReferralConfig>({
  discount_percent: 10,
  commission_percent: 10,
  min_purchase_cents: 0,
  max_commission_cents: 0,
  is_active: true,
})

// Rupiah helpers for display inputs
const minPurchaseRp = ref(0)
const maxCommissionRp = ref(0)

function formatRp(n: number): string {
  return Math.max(0, Math.round(n)).toLocaleString('id-ID')
}

async function loadConfig() {
  loading.value = true
  errorMsg.value = ''
  try {
    const data = await request('/admin/referral-config')
    if (data.data) {
      config.value = {
        discount_percent: data.data.discount_percent ?? 10,
        commission_percent: data.data.commission_percent ?? 10,
        min_purchase_cents: data.data.min_purchase_cents ?? 0,
        max_commission_cents: data.data.max_commission_cents ?? 0,
        is_active: data.data.is_active ?? true,
      }
      minPurchaseRp.value = config.value.min_purchase_cents / 100
      maxCommissionRp.value = config.value.max_commission_cents / 100
    }
  } catch (e: any) {
    errorMsg.value = e?.message || 'Gagal memuat konfigurasi'
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const data = await request('/admin/referral-config', {
      method: 'PUT',
      body: JSON.stringify({
        discount_percent: config.value.discount_percent,
        commission_percent: config.value.commission_percent,
        min_purchase_cents: config.value.min_purchase_cents,
        max_commission_cents: config.value.max_commission_cents,
        is_active: config.value.is_active,
      }),
    })
    if (data.status === 200 || data.message) {
      successMsg.value = data.message || 'Konfigurasi berhasil disimpan!'
      setTimeout(() => successMsg.value = '', 3000)
    }
  } catch (e: any) {
    errorMsg.value = e?.message || 'Gagal menyimpan'
  } finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>

<style scoped>
.page { max-width: 680px; margin: 0 auto; }

h1 { font-size: 22px; margin-bottom: 6px; }
.desc { color: var(--muted); margin-bottom: 24px; font-size: 14px; }

.spinner { text-align: center; padding: 3rem; color: var(--muted); }

.form-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 28px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-weight: 600;
  font-size: 13px;
  color: var(--text);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}

.input {
  width: 100%;
  padding: 10px 14px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  color: var(--text);
  font-size: 15px;
}
.input:focus { outline: none; border-color: var(--accent); }

.hint { display: block; font-size: 11px; color: var(--muted); line-height: 1.4; }

.toggle-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 0;
  border-top: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}

.checkbox {
  width: 18px;
  height: 18px;
  accent-color: var(--accent);
  cursor: pointer;
}

.toggle-text { font-size: 15px; font-weight: 600; }
.toggle-hint { font-size: 13px; color: var(--muted); }

/* Preview box */
.preview-box {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 16px;
}

.preview-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 12px;
}

.preview-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
  font-size: 13px;
  color: var(--muted);
}

.preview-value { font-weight: 700; color: var(--text); }
.preview-value.discount { color: var(--danger); }
.preview-value.commission { color: var(--success); }
.capped { font-size: 11px; color: var(--muted); font-weight: normal; margin-left: 4px; }

.actions { display: flex; flex-direction: column; gap: 8px; }

.btn-save {
  width: 100%;
  padding: 12px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}
.btn-save:hover:not(:disabled) { background: #2563eb; }
.btn-save:disabled { opacity: 0.5; }

.error { color: var(--danger); font-size: 13px; }
.success { color: var(--success); font-size: 13px; }

@media (max-width: 600px) {
  .form-grid { grid-template-columns: 1fr; }
}
</style>
