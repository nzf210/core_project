<template>
  <div class="redeem-page-container">
    <div class="redeem-card animate-fade-in">
      <div class="logo-wrapper">
        <h2 class="brand-title">WCH Platform</h2>
        <p class="brand-subtitle">Klaim & Aktivasi Voucher</p>
      </div>

      <!-- State 1: Loading / Redeeming -->
      <div v-if="loading" class="state-box">
        <div class="spinner"></div>
        <h3>Memverifikasi & Mengaktifkan Voucher...</h3>
        <p class="text-muted">Mohon tunggu sebentar, sistem sedang menghubungkan paket Anda.</p>
      </div>

      <!-- State 2: Success -->
      <div v-else-if="success" class="state-box success-box">
        <div class="icon-circle success-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
        </div>
        <h2>Voucher Berhasil Diaktifkan!</h2>
        <p class="text-muted">Selamat! Akun Anda kini aktif dengan paket langganan berikut:</p>

        <div class="voucher-details" v-if="voucherResult">
          <div class="detail-row">
            <span>Paket Layanan:</span>
            <strong>{{ voucherResult.plan_name || voucherResult.plan_id?.toUpperCase() || 'LITE' }}</strong>
          </div>
          <div class="detail-row" v-if="voucherResult.duration_months">
            <span>Durasi:</span>
            <strong>{{ voucherResult.duration_months }} Bulan</strong>
          </div>
          <div class="detail-row" v-if="voucherResult.expires_at">
            <span>Masa Aktif Hingga:</span>
            <strong>{{ formatDate(voucherResult.expires_at) }}</strong>
          </div>
          <div class="detail-row" v-if="voucherResult.ticket_number">
            <span>No. Tiket:</span>
            <code class="ticket-code">{{ voucherResult.ticket_number }}</code>
          </div>
        </div>

        <button class="btn btn-primary btn-block" @click="goToDashboard">
          Buka Dashboard Sekarang →
        </button>
      </div>

      <!-- State 3: Guest / Need Login -->
      <div v-else-if="needLogin" class="state-box">
        <div class="icon-circle gift-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="20 12 20 22 4 22 4 12"></polyline>
            <rect x="2" y="7" width="20" height="5"></rect>
            <line x1="12" y1="22" x2="12" y2="7"></line>
            <path d="M12 7H7.5a2.5 2.5 0 0 1 0-5C11 2 12 7 12 7z"></path>
            <path d="M12 7h4.5a2.5 2.5 0 0 0 0-5C13 2 12 7 12 7z"></path>
          </svg>
        </div>
        <h2>Voucher Siap Digunakan!</h2>
        <p class="text-muted">Link voucher Anda valid. Silakan masuk atau buat akun untuk menautkan paket ini ke toko Anda.</p>

        <div class="action-buttons">
          <router-link to="/login" class="btn btn-primary btn-block">
            Masuk ke Akun Toko
          </router-link>
          <router-link to="/register" class="btn btn-secondary btn-block">
            Daftar Akun Baru
          </router-link>
        </div>
      </div>

      <!-- State 4: Error or Manual Input -->
      <div v-else class="state-box error-box">
        <div class="icon-circle error-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="15" y1="9" x2="9" y2="15"></line>
            <line x1="9" y1="9" x2="15" y2="15"></line>
          </svg>
        </div>
        <h2>Gagal Mengklaim Voucher</h2>
        <p class="error-text">{{ errorMessage }}</p>

        <form class="manual-input-box" @submit.prevent="handleManualRedeem">
          <label>Masukkan Kode Voucher atau Link Lain:</label>
          <div class="input-group">
            <input v-model="manualInput" type="text" class="form-control" placeholder="cth: LITE-83921-A1B2 atau link klaim" required />
            <button type="submit" class="btn btn-primary" :disabled="loadingManual">
              {{ loadingManual ? 'Proses...' : 'Klaim' }}
            </button>
          </div>
        </form>

        <div class="action-buttons" style="margin-top: 1rem;">
          <router-link to="/dashboard" class="btn btn-secondary btn-block">
            Kembali ke Dashboard
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const loadingManual = ref(false)
const success = ref(false)
const needLogin = ref(false)
const errorMessage = ref('')
const voucherResult = ref<any>(null)
const manualInput = ref('')

function getTokenFromUrl(): string {
  if (route.query.token) {
    return String(route.query.token).trim()
  }
  const hash = window.location.hash
  if (hash.includes('token=')) {
    const match = hash.match(/[?&]token=([^&#]+)/)
    if (match) return match[1]
  }
  return ''
}

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr)
    return d.toLocaleDateString('id-ID', { year: 'numeric', month: 'long', day: 'numeric' })
  } catch {
    return dateStr
  }
}

async function redeemToken(token: string) {
  const accessToken = localStorage.getItem('access_token')
  const tenantId = localStorage.getItem('tenant_id')

  if (!accessToken || !tenantId) {
    // Save for post-login auto redemption
    localStorage.setItem('pending_voucher_token', token)
    needLogin.value = true
    return
  }

  loading.value = true
  errorMessage.value = ''
  try {
    const res = await api.redeemVoucherLink(token, tenantId)
    if (res.success && res.data) {
      voucherResult.value = res.data
      success.value = true
      localStorage.setItem('onboarding_completed', 'true')
      if (res.data.plan_id) localStorage.setItem('plan', res.data.plan_id)
      localStorage.removeItem('pending_voucher_token')
    } else {
      errorMessage.value = res.message || 'Voucher link tidak valid atau sudah pernah digunakan.'
    }
  } catch (err: any) {
    errorMessage.value = err.message || 'Terjadi kesalahan saat memproses klaim voucher.'
  } finally {
    loading.value = false
  }
}

async function handleManualRedeem() {
  if (!manualInput.value.trim()) return
  const raw = manualInput.value.trim()
  loadingManual.value = true
  errorMessage.value = ''

  try {
    let token = ''
    if (raw.includes('token=')) {
      try {
        const u = new URL(raw.startsWith('http') ? raw : `https://${raw}`)
        token = u.searchParams.get('token') || ''
      } catch {
        const match = raw.match(/[?&]token=([^&#]+)/)
        if (match) token = match[1]
      }
    } else if (raw.startsWith('ey') && raw.includes('.')) {
      token = raw
    }

    if (token) {
      await redeemToken(token)
    } else {
      const res = await api.post('/voucher/redeem', { code: raw.toUpperCase() })
      if (res.success && res.data) {
        voucherResult.value = res.data
        success.value = true
        localStorage.setItem('onboarding_completed', 'true')
        if (res.data.plan_id) localStorage.setItem('plan', res.data.plan_id)
      } else {
        errorMessage.value = res.message || 'Kode voucher tidak valid atau sudah digunakan.'
      }
    }
  } catch (e: any) {
    errorMessage.value = e.message || 'Gagal memproses voucher.'
  } finally {
    loadingManual.value = false
  }
}

function goToDashboard() {
  router.push('/dashboard')
}

onMounted(async () => {
  const token = getTokenFromUrl()
  if (token) {
    await redeemToken(token)
  } else {
    // If no token in URL, check if pending in storage or prompt user
    const pending = localStorage.getItem('pending_voucher_token')
    if (pending && localStorage.getItem('access_token')) {
      await redeemToken(pending)
    } else {
      errorMessage.value = 'Silakan masukkan kode voucher atau link klaim Anda di bawah ini.'
    }
  }
})
</script>

<style scoped>
.redeem-page-container {
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: var(--bg-color);
}
.redeem-card {
  width: 100%;
  max-width: 520px;
  background: var(--surface-0);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: 2.5rem;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  text-align: center;
}
.logo-wrapper { margin-bottom: 2rem; }
.brand-title { font-size: 24px; font-weight: 800; color: var(--accent-primary); margin: 0; }
.brand-subtitle { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }
.state-box { display: flex; flex-direction: column; align-items: center; gap: 12px; }
.spinner {
  width: 44px; height: 44px;
  border: 4px solid rgba(59, 130, 246, 0.2);
  border-top-color: var(--accent-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.icon-circle {
  width: 68px; height: 68px; border-radius: 50%;
  display: flex; align-items: center; justify-content: center; margin-bottom: 8px;
}
.success-icon { background: rgba(16, 185, 129, 0.15); color: #10b981; }
.gift-icon { background: rgba(59, 130, 246, 0.15); color: #3b82f6; }
.error-icon { background: rgba(239, 68, 68, 0.15); color: #ef4444; }
.voucher-details {
  width: 100%; background: var(--bg-tertiary);
  border-radius: var(--radius-md); padding: 14px; margin: 12px 0;
  display: flex; flex-direction: column; gap: 8px; text-align: left; font-size: 13px;
}
.detail-row { display: flex; justify-content: space-between; align-items: center; }
.ticket-code { font-family: monospace; font-size: 12px; background: var(--surface-0); padding: 2px 6px; border-radius: 4px; }
.action-buttons { display: flex; flex-direction: column; gap: 10px; width: 100%; margin-top: 8px; }
.btn-block { width: 100%; padding: 10px 16px; font-weight: 600; text-decoration: none; display: inline-block; box-sizing: border-box; }
.error-text { color: #ef4444; font-size: 14px; margin-bottom: 12px; }
.manual-input-box { width: 100%; text-align: left; margin-top: 12px; }
.manual-input-box label { font-size: 12px; color: var(--text-secondary); margin-bottom: 6px; display: block; }
.input-group { display: flex; gap: 8px; }
.input-group input { flex: 1; }
</style>
