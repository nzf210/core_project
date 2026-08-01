<template>
  <div class="onboarding-page">
    <div class="onboarding-container">
      <h1 class="onboarding-title">Selamat Datang di WCH UMKM</h1>
      <p class="onboarding-subtitle">Pilih jenis usaha Anda untuk pengalaman yang optimal</p>

      <div class="step-indicator">
        <div :class="['step', currentStep >= 1 ? 'active' : '']">
          <div class="step-number">1</div>
          <span>Jenis Usaha</span>
        </div>
        <div class="step-line" :class="{ active: currentStep >= 2 }"></div>
        <div :class="['step', currentStep >= 2 ? 'active' : '']">
          <div class="step-number">2</div>
          <span>Detail Usaha</span>
        </div>
        <div class="step-line" :class="{ active: currentStep >= 3 }"></div>
        <div :class="['step', currentStep >= 3 ? 'active' : '']">
          <div class="step-number">3</div>
          <span>Aktivasi</span>
        </div>
      </div>

      <!-- Step 1: Pilih Jenis Usaha (tanpa gate) -->
      <div v-if="currentStep === 1" class="step-content">
        <div class="business-types-grid">
          <div v-for="bt in businessTypes" :key="bt.id" :class="['bt-card', selectedType === bt.id ? 'selected' : '']"
            @click="selectType(bt.id)">
            <div class="bt-icon">{{ bt.icon }}</div>
            <div class="bt-name">{{ bt.name }}</div>
            <div class="bt-desc">{{ bt.description }}</div>
          </div>
        </div>
        <p class="skip-hint">* opsional — boleh skip</p>
        <button class="btn btn-primary btn-large" @click="currentStep = 2">
          Lanjutkan
        </button>
      </div>

      <!-- Step 2: Detail Usaha (tanpa gate, tanpa plan selector) -->
      <div v-if="currentStep === 2" class="step-content">
        <div class="form-group">
          <label for="onboard-name">Nama Usaha</label>
          <input id="onboard-name" v-model="businessName" type="text" placeholder="Nama usaha Anda" class="input-field" />
        </div>
        <div class="form-group">
          <label for="onboard-address">Alamat Usaha (opsional)</label>
          <textarea id="onboard-address" v-model="businessAddress" placeholder="Alamat lengkap usaha" class="input-field"
            rows="2"></textarea>
        </div>
        <div class="form-group">
          <label for="onboard-wa">Nomor WhatsApp (untuk notifikasi)</label>
          <input id="onboard-wa" v-model="waNumber" type="text" placeholder="08xxxxxxxxxx" class="input-field" />
        </div>
        <div class="step-actions">
          <button class="btn btn-secondary" @click="currentStep = 1">Kembali</button>
          <button class="btn btn-primary btn-large" @click="submitBusinessDetails">
            Lanjut ke Aktivasi
          </button>
        </div>
      </div>

      <!-- Step 3: Completion + prompt if not yet activated -->
      <div v-if="currentStep === 3" class="step-content">
        <div class="completion-card">
          <div class="completion-icon">✓</div>
          <h3>Semua Siap!</h3>
          <p>Detail usaha Anda sudah disimpan untuk {{ getTypeName(selectedType) }}</p>
        </div>

        <!-- Aktivasi banner — muncul jika belum aktif -->
        <div v-if="!isActivated" class="activation-banner">
          <h3>Aktifkan Langganan Anda</h3>
          <p>Pilih metode aktivasi di bawah untuk mulai menggunakan WCH Platform</p>

          <!-- Tab: Beli Paket / Masukkan Voucher -->
          <div class="activation-tabs">
            <button :class="['tab-btn', activationTab === 'buy' ? 'active' : '']" @click="activationTab = 'buy'">
              Beli Paket
            </button>
            <button :class="['tab-btn', activationTab === 'voucher' ? 'active' : '']"
              @click="activationTab = 'voucher'">
              Masukkan Voucher
            </button>
          </div>

          <!-- Beli Paket -->
          <div v-if="activationTab === 'buy'" class="activation-panel">
            <!-- Billing cycle toggle -->
            <div class="billing-toggle">
              <button :class="['toggle-btn', billingCycle === 'monthly' ? 'active' : '']"
                @click="billingCycle = 'monthly'">Bulanan</button>
              <button :class="['toggle-btn', billingCycle === 'yearly' ? 'active' : '']"
                @click="billingCycle = 'yearly'">Tahunan <span class="save-badge">Hemat</span></button>
            </div>

            <!-- F058: Wallet balance indicator -->
            <div v-if="walletBalance > 0" class="wallet-indicator">
              <span class="wallet-icon">💳</span>
              <span>Saldo Wallet: <strong>{{ formatRupiah(walletBalance) }}</strong></span>
            </div>

            <div class="plan-selector">
              <div v-for="plan in plans" :key="plan.id"
                :class="['plan-card', selectedPlan === plan.id ? 'selected' : '']" @click="selectedPlan = plan.id">
                <div class="plan-badge" v-if="plan.sort_order === 2">Populer</div>
                <div class="plan-name">{{ plan.name }}</div>
                <div class="plan-price">
                  <span v-if="billingCycle === 'monthly'">{{ formatRupiah(plan.price_monthly)
                  }}<span>/bulan</span></span>
                  <span v-else>{{ formatRupiah(plan.price_yearly) }}<span>/tahun</span></span>
                </div>
                <ul class="plan-features">
                  <li v-for="f in plan.features" :key="f.feature_key">{{ f.feature_name }}</li>
                </ul>
              </div>
            </div>

            <!-- F058: Wallet payment option -->
            <div v-if="walletBalance > 0" class="wallet-pay-option">
              <label class="pay-radio">
                <input type="radio" :value="false" v-model="payViaWallet" />
                <span>Xendit (Transfer Bank / QRIS / EWallet)</span>
              </label>
              <label class="pay-radio">
                <input type="radio" :value="true" v-model="payViaWallet" />
                <span>Bayar dari Wallet ({{ formatRupiah(plans.find(p => p.id === selectedPlan)?.price_monthly || 0) }})</span>
              </label>
            </div>

            <button class="btn btn-primary btn-large" :disabled="isActivating" @click="buyPackage">
              <span v-if="isActivating">Memproses...</span>
              <span v-else-if="payViaWallet">Beli Paket — Bayar dari Wallet</span>
              <span v-else>Beli Paket — Bayar Sekarang</span>
            </button>
            <p v-if="paymentInfo" class="payment-info">
              Invoice dibuat! Klik <a :href="paymentInfo" target="_blank">di sini</a> untuk bayar.
              Setelah bayar, akun Anda akan aktif otomatis.
            </p>
          </div>

          <!-- Masukkan Voucher -->
          <div v-if="activationTab === 'voucher'" class="activation-panel">
            <div class="voucher-input-group">
              <label for="onboard-voucher" class="sr-only">Kode Voucher</label>
              <input id="onboard-voucher" v-model="voucherCode" type="text" placeholder="Kode voucher / referral agen (AGEN-XXX)"
                class="input-field" @keyup.enter="redeemVoucher" />
              <button class="btn btn-primary" :disabled="!voucherCode || isActivating" @click="redeemVoucher">
                <span v-if="isActivating">...</span>
                <span v-else>Aktivasi</span>
              </button>
            </div>
            <p class="voucher-hint">Voucher admin atau kode referral agen (AGEN-XXXXXX)</p>
          </div>

          <p v-if="activationError" class="error-text">{{ activationError }}</p>
          <p v-if="activationSuccess" class="success-text">{{ activationSuccess }}</p>
        </div>

        <!-- Sudah aktif -->
        <div v-else class="activated-notice">
          <span class="check-icon">✓</span>
          <p>Langganan Anda sudah aktif! <button class="link-btn" @click="goToDashboard">Buka Dashboard →</button></p>
        </div>

        <button v-if="!isActivated" class="btn btn-secondary btn-large skip-btn" @click="goToDashboard">
          Buka Dashboard (tanpa aktivasi)
        </button>
        <button v-else class="btn btn-primary btn-large" @click="goToDashboard">
          Buka Dashboard
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api, sanitizeText } from '../api'
import { useRouter } from 'vue-router'
import { formatRupiah } from '../composables/useCurrency'

const router = useRouter()

const currentStep = ref(1)
const selectedType = ref('')
const businessName = ref('')
const businessAddress = ref('')
const waNumber = ref('')
const selectedPlan = ref('')
const billingCycle = ref<'monthly' | 'yearly'>('monthly')
const activationTab = ref<'buy' | 'voucher'>('buy')
const isActivating = ref(false)
const isActivated = ref(false)
const activationError = ref('')
const activationSuccess = ref('')
const voucherCode = ref('')
const paymentInfo = ref('')
const walletBalance = ref(0)
const payViaWallet = ref(false)
const plans = ref<any[]>([])

const iconMap: Record<string, string> = {
  umum: '🏪',
  warung: '🛒',
  laundry: '👕',
  industri_kreatif: '🎨',
  toko_online: '🌐',
  restoran: '🍽️',
  jasa: '💼'
}

interface BusinessType {
  id: string
  name: string
  description: string
  icon: string
  defaultModules: string[]
}

const businessTypes = ref<BusinessType[]>([])

const loadBusinessTypes = async () => {
  try {
    const data = await api.get('/api/umkm/business/business-types')
    if (data.data) {
      businessTypes.value = (data.data as BusinessType[]).map(bt => ({
        ...bt,
        icon: iconMap[bt.id] || '📦'
      }))
    }
  } catch {
    businessTypes.value = [
      { id: 'umum', name: 'Umum', description: 'Semua jenis usaha', icon: '🏪', defaultModules: [] },
      { id: 'warung', name: 'Warung / Toko', description: 'Toko kelontong & retail', icon: '🛒', defaultModules: [] },
      { id: 'laundry', name: 'Laundry', description: 'Jasa cuci & setrika', icon: '👕', defaultModules: [] },
      { id: 'industri_kreatif', name: 'Industri Kreatif', description: 'Desain, foto, kerajinan', icon: '🎨', defaultModules: [] },
      { id: 'toko_online', name: 'Toko Online', description: 'E-commerce & marketplace', icon: '🌐', defaultModules: [] },
      { id: 'restoran', name: 'Restoran / F&B', description: 'Rumah makan & cafe', icon: '🍽️', defaultModules: [] },
      { id: 'jasa', name: 'Jasa / Service', description: 'Konsultan & profesional', icon: '💼', defaultModules: [] }
    ]
  }
}

const loadPlans = async () => {
  try {
    const data = await api.get('/plans')
    if (data.data) {
      plans.value = data.data
      if (plans.value.length > 0) selectedPlan.value = plans.value[0].id
    }
  } catch (e) {
    console.warn('Failed to load plans', e)
  }
}

const loadWalletBalance = async () => {
  try {
    const data: any = await api.get('/wallet')
    if (data?.data?.balance_rupiah != null) {
      walletBalance.value = data.data.balance_rupiah
    }
  } catch { /* silent */ }
}

const checkActivation = async () => {
  try {
    const data = await api.get('/subscription')
    isActivated.value = data.data?.has_subscription === true
  } catch {
    isActivated.value = false
  }
}

const selectType = (id: string) => {
  selectedType.value = id
}

const getTypeName = (id: string) => {
  const bt = businessTypes.value.find(b => b.id === id)
  return bt ? bt.name : id
}



const submitBusinessDetails = async () => {
  try {
    await api.post('/api/umkm/business/onboarding', {
      businessType: selectedType.value,
      businessName: businessName.value,
      businessAddress: businessAddress.value,
      waNumber: waNumber.value,
    })
  } catch {
    // non-fatal: continue anyway
  }
  localStorage.setItem('onboarding_completed', 'true')
  localStorage.setItem('business_type', sanitizeText(selectedType.value, 50))
  localStorage.setItem('business_name', sanitizeText(businessName.value))
  currentStep.value = 3
  await checkActivation()
}

const buyPackage = async () => {
  if (!selectedPlan.value) return
  isActivating.value = true
  activationError.value = ''
  paymentInfo.value = ''
  try {
    const data: any = await api.post('/subscribe', {
      plan_id: selectedPlan.value,
      billing_cycle: billingCycle.value,
      pay_via_wallet: payViaWallet.value,
    })
    if (data.status >= 400) {
      activationError.value = data.message || 'Gagal membuat invoice'
      return
    }
    if (data.data?.payment_url) {
      paymentInfo.value = data.data.payment_url
      activationError.value = ''
    } else {
      // Dev mode atau lite plan: langsung aktif
      activationSuccess.value = 'Langganan berhasil diaktifkan!'
      isActivated.value = true
      sessionStorage.setItem('chatbot_wizard_pending', '1')
    }
  } catch (e: any) {
    activationError.value = e?.message || 'Terjadi kesalahan'
  } finally {
    isActivating.value = false
  }
}

const redeemVoucher = async () => {
  if (!voucherCode.value.trim()) return
  isActivating.value = true
  activationError.value = ''
  activationSuccess.value = ''
  const code = voucherCode.value.trim().toUpperCase()
  try {
    if (code.startsWith('AGEN-')) {
      const data = await api.redeemReferral(code)
      if (data.status >= 400) {
        activationError.value = data.message || 'Kode referral tidak valid'
        return
      }
      activationSuccess.value = 'Kode referral berhasil diterapkan! Akun Anda kini terhubung dengan agen.'
    } else {
      const data = await api.post('/voucher/redeem', { code })
      if (data.status >= 400) {
        activationError.value = data.message || 'Kode voucher tidak valid'
        return
      }
      activationSuccess.value = 'Voucher berhasil diaktifkan! Selamat menikmati WCH Platform.'
      isActivated.value = true
      sessionStorage.setItem('chatbot_wizard_pending', '1')
    }
    voucherCode.value = ''
  } catch (e: any) {
    activationError.value = e?.message || 'Kode voucher tidak valid atau sudah digunakan'
  } finally {
    isActivating.value = false
  }
}

const goToDashboard = () => {
  // F020: after first-time activation, send user to the AI CS setup wizard
  // once. After that, the button just opens the regular dashboard.
  if (sessionStorage.getItem('chatbot_wizard_pending') === '1') {
    sessionStorage.removeItem('chatbot_wizard_pending')
    router.push('/wa-setup?activeTab=ai_config&first_run=1')
    return
  }
  router.push('/')
}

onMounted(async () => {
  await Promise.all([loadBusinessTypes(), loadPlans(), loadWalletBalance()])
})
</script>

<style scoped>
.onboarding-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
}

.onboarding-container {
  max-width: 800px;
  width: 100%;
  text-align: center;
}

.onboarding-title {
  font-size: 2rem;
  font-weight: 700;
  color: #f1f5f9;
  margin-bottom: 0.5rem;
}

.onboarding-subtitle {
  color: #94a3b8;
  margin-bottom: 2rem;
}

.step-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0;
  margin-bottom: 2.5rem;
}

.step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  color: #475569;
}

.step.active {
  color: #3b82f6;
}

.step-number {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #1e293b;
  border: 2px solid #475569;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
}

.step.active .step-number {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
}

.step-line {
  width: 60px;
  height: 2px;
  background: #334155;
  margin: 0 0.5rem 1.5rem;
}

.step-line.active {
  background: #3b82f6;
}

.business-types-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.bt-card {
  background: #1e293b;
  border: 2px solid #334155;
  border-radius: 12px;
  padding: 1.5rem 1rem;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
}

.bt-card:hover {
  border-color: #3b82f6;
  transform: translateY(-2px);
}

.bt-card.selected {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.bt-icon {
  font-size: 2.5rem;
  margin-bottom: 0.75rem;
}

.bt-name {
  font-weight: 600;
  color: #f1f5f9;
  font-size: 1.1rem;
  margin-bottom: 0.25rem;
}

.bt-desc {
  color: #64748b;
  font-size: 0.875rem;
}

.form-group {
  margin-bottom: 1.5rem;
  text-align: left;
}

.form-group label {
  display: block;
  color: #94a3b8;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.input-field {
  width: 100%;
  padding: 0.75rem 1rem;
  background: #1e293b;
  border: 1px solid #334155;
  border-radius: 8px;
  color: #f1f5f9;
  font-size: 1rem;
  box-sizing: border-box;
}

.input-field:focus {
  outline: none;
  border-color: #3b82f6;
}

.plan-selector {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.plan-card {
  background: #1e293b;
  border: 2px solid #334155;
  border-radius: 12px;
  padding: 1.5rem;
  cursor: pointer;
  transition: all 0.2s;
  position: relative;
}

.plan-card:hover {
  border-color: #3b82f6;
}

.plan-card.selected {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.plan-badge {
  position: absolute;
  top: -10px;
  right: -10px;
  background: #22c55e;
  color: #fff;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
}

.plan-name {
  font-weight: 600;
  color: #f1f5f9;
  font-size: 1.1rem;
  margin-bottom: 0.5rem;
}

.plan-price {
  font-size: 1.5rem;
  font-weight: 700;
  color: #3b82f6;
  margin-bottom: 1rem;
}

.plan-price span {
  font-size: 0.875rem;
  font-weight: 400;
  color: #64748b;
}

.plan-features {
  list-style: none;
  padding: 0;
  text-align: left;
}

.plan-features li {
  color: #94a3b8;
  padding: 0.25rem 0;
  font-size: 0.875rem;
}

.plan-features li::before {
  content: '✓ ';
  color: #22c55e;
}

.btn {
  padding: 0.75rem 2rem;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
}

.btn-primary {
  background: #3b82f6;
  color: #fff;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: #334155;
  color: #94a3b8;
}

.btn-secondary:hover {
  background: #475569;
}

.btn-large {
  padding: 1rem 2.5rem;
  font-size: 1.1rem;
  margin-top: 1rem;
}

.step-actions {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 1rem;
}

.completion-card {
  padding: 3rem;
  text-align: center;
}

.completion-icon {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: #22c55e;
  color: #fff;
  font-size: 2.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1.5rem;
}

.completion-card h3 {
  color: #f1f5f9;
  font-size: 1.5rem;
  margin-bottom: 0.5rem;
}

.completion-card p {
  color: #94a3b8;
  margin-bottom: 2rem;
}

@media (max-width: 768px) {
  .onboarding-title {
    font-size: 1.5rem;
  }

  .business-types-grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  }

  .plan-selector {
    grid-template-columns: 1fr;
  }
}

.skip-hint {
  color: #475569;
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.activation-banner {
  background: #1e293b;
  border: 1px solid #334155;
  border-radius: 12px;
  padding: 2rem;
  margin: 1.5rem 0;
  text-align: left;
}

.activation-banner h3 {
  color: #f1f5f9;
  margin-bottom: 0.5rem;
}

.activation-banner>p {
  color: #94a3b8;
  margin-bottom: 1.5rem;
}

.activation-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  border-bottom: 2px solid #334155;
}

.tab-btn {
  padding: 0.5rem 1.5rem;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #94a3b8;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
  font-size: 0.95rem;
  margin-bottom: -2px;
}

.tab-btn.active {
  color: #3b82f6;
  border-bottom-color: #3b82f6;
}

.activation-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.voucher-input-group {
  display: flex;
  gap: 0.75rem;
}

.voucher-input-group .input-field {
  flex: 1;
}

.voucher-hint {
  color: #475569;
  font-size: 0.875rem;
}

.error-text {
  color: #f87171;
  font-size: 0.875rem;
  padding: 0.5rem;
  background: rgba(248, 113, 113, 0.1);
  border-radius: 6px;
}

.success-text {
  color: #4ade80;
  font-size: 0.875rem;
  padding: 0.5rem;
  background: rgba(74, 222, 128, 0.1);
  border-radius: 6px;
}

.activated-notice {
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid #22c55e;
  border-radius: 10px;
  padding: 1rem 1.5rem;
  margin: 1.5rem 0;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: #4ade80;
}

.check-icon {
  font-size: 1.5rem;
}

.activated-notice p {
  color: #4ade80;
  margin: 0;
}

.link-btn {
  background: none;
  border: none;
  color: #3b82f6;
  cursor: pointer;
  font-size: inherit;
  font-weight: 600;
  padding: 0;
  font-family: inherit;
  text-decoration: underline;
}

.skip-btn {
  margin-top: 0.75rem;
  opacity: 0.7;
}

.payment-info {
  color: #94a3b8;
  font-size: 0.875rem;
}

.payment-info a {
  color: #3b82f6;
}

.billing-toggle {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 10px;
  padding: 4px;
  width: fit-content;
}

.toggle-btn {
  flex: 1;
  padding: 0.5rem 1.25rem;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 600;
  font-family: inherit;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.toggle-btn.active {
  background: #3b82f6;
  color: #fff;
}

.save-badge {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 0.7rem;
  font-weight: 700;
}

.wallet-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.3);
  border-radius: 10px;
  padding: 0.6rem 1rem;
  margin-bottom: 1rem;
  font-size: 0.88rem;
  color: #86efac;
}

.wallet-pay-option {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.pay-radio {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.88rem;
  color: #94a3b8;
  cursor: pointer;
  padding: 0.5rem 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  transition: border-color 0.2s;
}

.pay-radio:has(input:checked) {
  border-color: #22c55e;
  color: #86efac;
  background: rgba(34, 197, 94, 0.06);
}

.pay-radio input {
  accent-color: #22c55e;
  cursor: pointer;
}
</style>
