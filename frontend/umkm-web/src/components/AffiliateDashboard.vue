<template>
  <div class="affiliate-dashboard">
    <h2>🤝 Agen Afiliasi</h2>

    <!-- Loading -->
    <div v-if="loading" class="loading-block">
      <div class="spinner"></div>
      <p>Memuat...</p>
    </div>

    <!-- Belum terdaftar -->
    <div v-else-if="!isAffiliate" class="card join-card">
      <div class="join-icon">🚀</div>
      <h3>Jadi Agen Afiliasi WCH</h3>
      <p>
        Dapatkan komisi <strong>20%</strong> dari setiap pembayaran tenant yang kamu rekrut — <em>seumur hidup</em>.
      </p>
      <ul class="benefit-list">
        <li>🔗 Kode referral unik</li>
        <li>💸 Komisi otomatis cair tiap tenant bayar</li>
        <li>🏆 Masuk papan peringkat publik</li>
        <li>💰 Minimal tarik dana Rp 100.000</li>
      </ul>
      <button class="btn btn-primary" @click="handleRegister" :disabled="registering">
        {{ registering ? 'Mendaftarkan...' : '✨ Daftar Sekarang' }}
      </button>
      <p v-if="errorMsg" class="error-msg">{{ errorMsg }}</p>
      <p v-if="successMsg" class="success-msg">{{ successMsg }}</p>
    </div>

    <!-- Sudah affiliate -->
    <div v-else class="affiliate-content">
      <!-- Card Referral -->
      <div class="card referral-card">
        <span class="stat-label">Kode Referral Kamu</span>
        <div class="code-row">
          <code id="referral-code">{{ referralCode }}</code>
          <button class="btn btn-sm" @click="copyCode" :title="copyTooltip">{{ copyIcon }}</button>
        </div>
        <p class="hint">Bagikan kode ini ke calon tenant. Mereka masukkan saat daftar.</p>
      </div>

      <!-- Card Saldo -->
      <div class="stats-row">
        <div class="card stat-card">
          <span class="stat-label">Saldo Tersedia</span>
          <div class="stat-value">{{ formatRupiah(balance) }}</div>
          <p class="hint">Bisa ditarik kapan saja</p>
        </div>
        <div class="card stat-card">
          <span class="stat-label">Total Pendapatan</span>
          <div class="stat-value gold">{{ formatRupiah(totalEarnings) }}</div>
          <p class="hint">Seumur hidup</p>
        </div>
      </div>

      <!-- Withdraw -->
      <div class="card withdraw-card">
        <h3>💸 Tarik Dana</h3>
        <div class="form-row">
          <div class="form-group">
            <label for="withdraw-amount">Jumlah (Rp)</label>
            <input
              id="withdraw-amount"
              v-model="withdrawAmount"
              type="number"
              class="form-control"
              placeholder="Minimal 100.000"
              min="100000"
              step="100000"
            />
          </div>
          <button 
            class="btn btn-primary" 
            @click="handleWithdraw" 
            :disabled="withdrawing || !withdrawAmount || withdrawAmount < 100000 || withdrawAmount * 100 > balance"
          >
            {{ withdrawing ? 'Memproses...' : 'Tarik' }}
          </button>
        </div>
        <p v-if="withdrawError" class="error-msg">{{ withdrawError }}</p>
        <p v-if="withdrawSuccess" class="success-msg">{{ withdrawSuccess }}</p>
      </div>

      <!-- Refresh -->
      <button class="btn btn-outline" @click="loadProfile" :disabled="loading" style="margin-top:1rem;">
        🔄 Refresh
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { formatRupiah } from '../composables/useCurrency'

const loading = ref(true)
const registering = ref(false)
const withdrawing = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const withdrawError = ref('')
const withdrawSuccess = ref('')

const isAffiliate = ref(false)
const affiliateId = ref(0)
const referralCode = ref('')
const balance = ref(0)         // in rupiah
const totalEarnings = ref(0)   // in rupiah
const withdrawAmount = ref<number | null>(null)

const copyTooltip = ref('Copy')
const copyIcon = ref('📋')


async function loadProfile() {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await api.getAffiliateProfile()
    // Backend wraps in {status, message, data}; data itself contains {is_affiliate, ...}
    const d = (res && res.data) ? res.data : null
    if (d) {
      isAffiliate.value = !!d.is_affiliate
      if (d.is_affiliate) {
        affiliateId.value = d.affiliate_id || 0
        referralCode.value = d.referral_code || ''
        balance.value = d.cash_balance_rupiah || 0
        totalEarnings.value = d.total_earnings_rupiah || 0
      }
    }
  } catch (e: any) {
    errorMsg.value = e?.message || 'Gagal memuat profil'
  } finally {
    loading.value = false
  }
}

async function handleRegister() {
  registering.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await api.registerAffiliate()
    // Backend returns {status, message, data}; treat status 2xx as success
    if (res && res.status >= 200 && res.status < 300) {
      successMsg.value = res.message || 'Berhasil terdaftar!'
      await loadProfile()
    } else {
      errorMsg.value = res?.message || 'Gagal mendaftar'
    }
  } catch (e: any) {
    errorMsg.value = e?.message || 'Kesalahan jaringan'
  } finally {
    registering.value = false
  }
}

async function handleWithdraw() {
  if (!withdrawAmount.value || withdrawAmount.value < 100000) return
  withdrawing.value = true
  withdrawError.value = ''
  withdrawSuccess.value = ''
  try {
    // API takes rupiah, user inputs rupiah
    const rupiah = withdrawAmount.value * 100
    const res = await api.withdrawAffiliate(rupiah)
    if (res && res.status >= 200 && res.status < 300) {
      withdrawSuccess.value = 'Permintaan tarik dana berhasil! Diproses dalam 1-3 hari kerja.'
      withdrawAmount.value = null
      await loadProfile()
    } else {
      withdrawError.value = res?.message || 'Gagal memproses penarikan'
    }
  } catch (e: any) {
    withdrawError.value = e?.message || 'Kesalahan jaringan'
  } finally {
    withdrawing.value = false
  }
}

async function copyCode() {
  try {
    await navigator.clipboard.writeText(referralCode.value)
    copyTooltip.value = 'Tersalin!'
    copyIcon.value = '✅'
    setTimeout(() => {
      copyTooltip.value = 'Copy'
      copyIcon.value = '📋'
    }, 2000)
  } catch {
    copyTooltip.value = 'Gagal'
  }
}

onMounted(loadProfile)
</script>

<style scoped>
.affiliate-dashboard {
  padding: 1.5rem;
  max-width: 720px;
  margin: 0 auto;
}

h2 { margin-bottom: 1.5rem; }
h3 { margin: 0 0 0.75rem; }

.card {
  background: var(--surface-0);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  margin-bottom: 1rem;
  box-shadow: 0 1px 3px rgba(0,0,0,0.08);
}

.loading-block {
  text-align: center;
  padding: 3rem 0;
  color: var(--text-secondary);
}

.spinner {
  width: 32px; height: 32px;
  border: 3px solid var(--border-color);
  border-top-color: var(--accent-primary);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  margin: 0 auto 0.75rem;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* --- Join Card --- */
.join-card { text-align: center; }
.join-icon { font-size: 3rem; margin-bottom: 0.75rem; }
.join-card p { color: var(--text-secondary); margin-bottom: 1rem; }

.benefit-list {
  list-style: none;
  padding: 0;
  text-align: left;
  display: inline-block;
  margin-bottom: 1.5rem;
}
.benefit-list li { padding: 0.4rem 0; }

/* --- Referral --- */
.referral-card label {
  font-size: 0.85rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  font-weight: 600;
}

.code-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin: 0.5rem 0;
}

.code-row code {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--accent-primary);
  background: var(--bg-tertiary);
  padding: 0.25rem 0.75rem;
  border-radius: var(--radius-sm);
  letter-spacing: 0.05em;
}

.hint {
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-top: 0.25rem;
}

/* --- Stats --- */
.stats-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.stat-card label {
  font-size: 0.8rem;
  text-transform: uppercase;
  color: var(--text-secondary);
  font-weight: 600;
}

.stat-value {
  font-size: 1.6rem;
  font-weight: 800;
  margin-top: 0.25rem;
}

.stat-value.gold { color: #f59e0b; }

/* --- Withdraw --- */
.withdraw-card .form-row {
  display: flex;
  gap: 0.75rem;
  align-items: end;
}

.form-group { flex: 1; }
.form-group label {
  display: block;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 0.25rem;
}
.form-control {
  width: 100%;
  padding: 0.6rem 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-color);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.95rem;
}

.btn {
  padding: 0.6rem 1.2rem;
  border: none;
  border-radius: var(--radius-sm);
  font-family: inherit;
  font-size: 0.9rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--accent-primary);
  color: #fff;
}
.btn-primary:hover:not(:disabled) {
  filter: brightness(1.1);
}

.btn-outline {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  width: 100%;
}
.btn-outline:hover:not(:disabled) {
  background: var(--bg-tertiary);
}

.btn-sm {
  padding: 0.35rem 0.65rem;
  font-size: 0.85rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.btn-sm:hover { background: var(--border-color); }

.error-msg { color: #ef4444; font-size: 0.85rem; margin-top: 0.5rem; }
.success-msg { color: #10b981; font-size: 0.85rem; margin-top: 0.5rem; }
</style>
