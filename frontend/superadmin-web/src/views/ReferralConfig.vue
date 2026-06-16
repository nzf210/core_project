<template>
  <div class="page">
    <h2>🎯 Konfigurasi Referral</h2>
    <p class="desc">Atur persentase diskon untuk user yang pakai kode referral, dan komisi untuk agen.</p>

    <div v-if="loading" class="spinner">⏳ Memuat...</div>

    <form v-else @submit.prevent="handleSave" class="form-card">
      <div class="field">
        <label>Diskon untuk User Baru (%)</label>
        <input 
          v-model.number="discountPercent" 
          type="number" 
          min="0" max="100" step="0.5"
          class="input" 
          placeholder="10"
        />
        <span class="hint">Potongan harga saat tenant pertama kali aktivasi pakai kode referral</span>
      </div>

      <div class="field">
        <label>Komisi Agen Afiliasi (%)</label>
        <input 
          v-model.number="commissionPercent" 
          type="number" 
          min="0" max="100" step="0.5"
          class="input" 
          placeholder="10"
        />
        <span class="hint">Persentase dari setiap pembayaran tenant yang masuk ke saldo agen (lifetime)</span>
      </div>

      <div class="actions">
        <button type="submit" class="btn-save" :disabled="saving">
          {{ saving ? 'Menyimpan...' : '💾 Simpan' }}
        </button>
      </div>

      <p v-if="errorMsg" class="error">{{ errorMsg }}</p>
      <p v-if="successMsg" class="success">{{ successMsg }}</p>
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

const discountPercent = ref(10)
const commissionPercent = ref(10)

async function loadConfig() {
  loading.value = true
  errorMsg.value = ''
  try {
    const data = await request('/admin/referral-config')
    if (data.success && data.data) {
      discountPercent.value = data.data.discount_percent ?? 10
      commissionPercent.value = data.data.commission_percent ?? 10
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
        discount_percent: discountPercent.value,
        commission_percent: commissionPercent.value,
      }),
    })
    if (data.success) {
      successMsg.value = 'Konfigurasi berhasil disimpan!'
      setTimeout(() => successMsg.value = '', 3000)
    } else {
      errorMsg.value = data.message || 'Gagal menyimpan'
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
.page {
  max-width: 560px;
  margin: 2rem auto;
  padding: 0 1rem;
}

h2 { margin-bottom: 0.25rem; }
.desc { color: #94a3b8; margin-bottom: 2rem; font-size: 0.9rem; }

.spinner { text-align: center; padding: 3rem; color: #94a3b8; }

.form-card {
  background: #1e293b;
  border: 1px solid #334155;
  border-radius: 12px;
  padding: 2rem;
}

.field { margin-bottom: 1.5rem; }

.field label {
  display: block;
  color: #e2e8f0;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.input {
  width: 100%;
  padding: 0.75rem;
  background: #0f172a;
  border: 1px solid #334155;
  border-radius: 8px;
  color: #e2e8f0;
  font-size: 1rem;
  box-sizing: border-box;
}
.input:focus { outline: none; border-color: #3b82f6; }

.hint {
  display: block;
  margin-top: 0.35rem;
  font-size: 0.78rem;
  color: #64748b;
}

.actions { margin-top: 1.5rem; }

.btn-save {
  width: 100%;
  padding: 0.85rem;
  background: #3b82f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
}
.btn-save:hover:not(:disabled) { background: #2563eb; }
.btn-save:disabled { opacity: 0.5; }

.error { color: #ef4444; margin-top: 1rem; font-size: 0.9rem; }
.success { color: #10b981; margin-top: 1rem; font-size: 0.9rem; }
</style>
