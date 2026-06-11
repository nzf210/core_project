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
          <span>Selesai</span>
        </div>
      </div>

      <div v-if="currentStep === 1" class="step-content">
        <div class="business-types-grid">
          <div
            v-for="bt in businessTypes"
            :key="bt.id"
            :class="['bt-card', selectedType === bt.id ? 'selected' : '']"
            @click="selectType(bt.id)"
          >
            <div class="bt-icon">{{ bt.icon }}</div>
            <div class="bt-name">{{ bt.name }}</div>
            <div class="bt-desc">{{ bt.description }}</div>
          </div>
        </div>
        <button class="btn btn-primary btn-large" :disabled="!selectedType" @click="nextStep">
          Lanjutkan
        </button>
      </div>

      <div v-if="currentStep === 2" class="step-content">
        <div class="form-group">
          <label>Nama Usaha</label>
          <input v-model="businessName" type="text" placeholder="Nama usaha Anda" class="input-field" />
        </div>
        <div class="form-group">
          <label>Alamat Usaha (opsional)</label>
          <textarea v-model="businessAddress" placeholder="Alamat lengkap usaha" class="input-field" rows="2"></textarea>
        </div>
        <div class="form-group">
          <label>Paket yang Dipilih</label>
          <div class="plan-selector">
            <div :class="['plan-card', selectedPlan === 'free' ? 'selected' : '']" @click="selectedPlan = 'free'">
              <div class="plan-name">Free</div>
              <div class="plan-price">Rp 0<span>/bulan</span></div>
              <ul class="plan-features">
                <li>1 User</li>
                <li>100 Transaksi/bulan</li>
                <li>5 AI Request/bulan</li>
                <li>Laporan Dasar</li>
              </ul>
            </div>
            <div :class="['plan-card', selectedPlan === 'lite' ? 'selected' : '']" @click="selectedPlan = 'lite'">
              <div class="plan-badge">Populer</div>
              <div class="plan-name">Lite</div>
              <div class="plan-price">Rp 49K<span>/bulan</span></div>
              <ul class="plan-features">
                <li>3 User</li>
                <li>1.000 Transaksi/bulan</li>
                <li>250 AI Request/bulan</li>
                <li>Export Laporan</li>
              </ul>
            </div>
            <div :class="['plan-card', selectedPlan === 'pro' ? 'selected' : '']" @click="selectedPlan = 'pro'">
              <div class="plan-name">Pro</div>
              <div class="plan-price">Rp 149K<span>/bulan</span></div>
              <ul class="plan-features">
                <li>Unlimited User</li>
                <li>10.000 Transaksi/bulan</li>
                <li>5.000 AI Request/bulan</li>
                <li>Full Inventory & Reports</li>
              </ul>
            </div>
          </div>
        </div>
        <div class="step-actions">
          <button class="btn btn-secondary" @click="currentStep = 1">Kembali</button>
          <button class="btn btn-primary btn-large" :disabled="!businessName" @click="completeOnboarding">
            Mulai Gunakan
          </button>
        </div>
      </div>

      <div v-if="currentStep === 3" class="step-content">
        <div class="completion-card">
          <div class="completion-icon">✓</div>
          <h3>Semua Siap!</h3>
          <p>Dashboard Anda sudah disiapkan untuk {{ getTypeName(selectedType) }}</p>
          <button class="btn btn-primary btn-large" @click="goToDashboard">Buka Dashboard</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { api } from '../api';
import { useRouter } from 'vue-router'

const router = useRouter()


const currentStep = ref(1)
const selectedType = ref('')
const businessName = ref('')
const businessAddress = ref('')
const selectedPlan = ref('free')

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

const selectType = (id: string) => {
  selectedType.value = id
}

const nextStep = () => {
  if (selectedType.value) {
    currentStep.value = 2
  }
}

const getTypeName = (id: string) => {
  const bt = businessTypes.value.find(b => b.id === id)
  return bt ? bt.name : id
}

const completeOnboarding = async () => {
  try {
    const data = await api.post('/api/umkm/business/onboarding', {
      businessType: selectedType.value,
      businessName: businessName.value,
      businessAddress: businessAddress.value,
      plan: selectedPlan.value,
    })
    if (data.status >= 400) {
      alert('Gagal menyimpan: ' + data.message)
      return
    }

    localStorage.setItem('onboarding_completed', 'true')
    localStorage.setItem('business_type', selectedType.value)
    localStorage.setItem('business_name', businessName.value)
    localStorage.setItem('plan', selectedPlan.value)

    currentStep.value = 3
  } catch {
    localStorage.setItem('onboarding_completed', 'true')
    localStorage.setItem('business_type', selectedType.value)
    localStorage.setItem('business_name', businessName.value)
    localStorage.setItem('plan', selectedPlan.value)
    currentStep.value = 3
  }
}

const goToDashboard = () => {
  router.push('/')
}

onMounted(() => {
  loadBusinessTypes()
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
</style>
