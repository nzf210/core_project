<template>
  <div class="automations-page">
    <div class="header-actions" style="margin-bottom: 2rem;">
      <div>
        <h2>Automasi</h2>
        <p style="color: var(--text-secondary);">Buat jadwal otomatis untuk laporan, notifikasi stok, dan lainnya.</p>
      </div>
      <div class="flex items-center gap-3">
        <span class="plan-badge" :class="`plan-${planInfo.plan}`">{{ (planInfo.plan || 'lite').toUpperCase() }}</span>
        <span class="usage-counter" v-if="planInfo.limit > 0">{{ planInfo.count }}/{{ planInfo.limit }} digunakan</span>
      </div>
    </div>

    <!-- Free Plan Banner -->
    <div v-if="planInfo.plan === 'lite'" class="upgrade-banner glass-card animate-fade-in" style="max-width: 700px;">
      <div class="upgrade-icon">🔒</div>
      <div>
        <h3 style="margin-bottom: 0.5rem;">Fitur Automasi Terkunci</h3>
        <p style="color: var(--text-secondary); margin-bottom: 1rem;">
          Paket gratis tidak mendukung fitur automasi. Upgrade ke paket <strong>Lite</strong> atau lebih tinggi
          untuk membuat jadwal laporan otomatis, notifikasi stok rendah, dan lainnya.
        </p>
        <div class="plan-comparison">
          <div class="plan-item">
            <span class="plan-badge plan-lite">LITE</span>
            <span>3 automasi</span>
          </div>
          <div class="plan-item">
            <span class="plan-badge plan-pro">PRO</span>
            <span>10 automasi</span>
          </div>
          <div class="plan-item">
            <span class="plan-badge plan-enterprise">ENTERPRISE</span>
            <span>Unlimited</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Automations List -->
    <div v-if="planInfo.plan !== 'lite'" style="max-width: 700px;">

      <!-- Add Button -->
      <button class="btn btn-primary add-automation-btn" @click="showForm = true" v-if="!showForm && planInfo.count < planInfo.limit">
        ＋ Tambah Automasi
      </button>
      <p v-if="!showForm && planInfo.count >= planInfo.limit && planInfo.limit > 0" class="limit-warning">
        ⚠️ Batas automasi ({{ planInfo.limit }}) telah tercapai. Hapus automasi lama atau upgrade paket.
      </p>

      <!-- Add Form -->
      <div v-if="showForm" class="glass-card animate-fade-in" style="padding: 2rem; margin-bottom: 1.5rem;">
        <h3 style="margin-bottom: 1.5rem;">{{ editingId ? 'Edit Automasi' : 'Buat Automasi Baru' }}</h3>
        <div style="display: flex; flex-direction: column; gap: 1rem;">
          <div>
            <label class="form-label" for="auto-type">Jenis Automasi</label>
            <select id="auto-type" v-model="form.type" class="form-control" @change="onTypeChange">
              <option value="" disabled>Pilih jenis automasi</option>
              <option value="daily_report">📊 Laporan Harian</option>
              <option value="weekly_report">📈 Laporan Mingguan</option>
              <option value="monthly_report">📋 Laporan Bulanan</option>
              <option value="low_stock_alert">⚠️ Alert Stok Rendah</option>
            </select>
          </div>
          <div>
            <label class="form-label" for="auto-name">Nama (custom)</label>
            <input id="auto-name" type="text" v-model="form.name" class="form-control" placeholder="Contoh: Laporan Pagi" />
          </div>

          <div class="schedule-picker">
            <label class="form-label" for="auto-schedule">Jadwal</label>
            <div class="flex gap-3 items-center flex-wrap">
              <select id="auto-schedule" v-model="scheduleFreq" class="form-control" style="flex: 1; min-width: 150px;">
                <option value="daily">Setiap Hari</option>
                <option value="weekly">Setiap Minggu</option>
                <option value="monthly">Setiap Bulan (Tgl 1)</option>
              </select>
              <div style="display: flex; align-items: center; gap: 0.5rem;">
                <label for="auto-time" style="color: var(--text-secondary);">Jam:</label>
                <input id="auto-time" type="time" v-model="scheduleTime" class="form-control" style="width: 130px;" />
              </div>
              <div v-if="scheduleFreq === 'weekly'" style="display: flex; align-items: center; gap: 0.5rem;">
                <label for="auto-day" style="color: var(--text-secondary);">Hari:</label>
                <select id="auto-day" v-model="scheduleDay" class="form-control" style="width: 130px;">
                  <option value="1">Senin</option>
                  <option value="2">Selasa</option>
                  <option value="3">Rabu</option>
                  <option value="4">Kamis</option>
                  <option value="5">Jumat</option>
                  <option value="6">Sabtu</option>
                  <option value="0">Minggu</option>
                </select>
              </div>
            </div>
            <p class="cron-preview">Cron: <code>{{ computedCron }}</code></p>
          </div>

          <div v-if="form.type === 'low_stock_alert'">
            <label class="form-label" for="auto-threshold">Threshold Stok (alert jika ≤)</label>
            <input id="auto-threshold" type="number" v-model.number="form.threshold" class="form-control" placeholder="5" min="1" />
          </div>

          <div>
            <label class="form-label" for="auto-wa">Nomor WA Tujuan (opsional, default: nomor toko)</label>
            <input id="auto-wa" type="text" v-model="form.target_wa" class="form-control" placeholder="6281234567890" />
          </div>

          <div class="flex gap-2" style="margin-top: 0.5rem;">
            <button class="btn btn-primary" @click="saveAutomation" :disabled="saving || !form.type || !form.name">
              {{ saving ? 'Menyimpan...' : (editingId ? 'Simpan Perubahan' : 'Buat Automasi') }}
            </button>
            <button class="btn btn-secondary" @click="cancelForm">Batal</button>
          </div>
        </div>
      </div>

      <!-- Automations Cards -->
      <div v-if="automations.length === 0 && !showForm" class="empty-state glass-card animate-fade-in">
        <div class="empty-icon">⚡</div>
        <h3>Belum Ada Automasi</h3>
        <p>Buat automasi pertama Anda untuk mengirim laporan otomatis via WhatsApp.</p>
      </div>

      <div v-for="auto in automations" :key="auto.id" class="automation-card glass-card animate-fade-in">
        <div class="card-header">
          <div class="card-info">
            <span class="type-icon">{{ getTypeIcon(auto.type) }}</span>
            <div>
              <h4 class="card-title">{{ auto.name }}</h4>
              <span class="type-label">{{ getTypeLabel(auto.type) }}</span>
            </div>
          </div>
          <label class="switch">
            <input type="checkbox" :checked="auto.enabled" @change="toggleEnabled(auto)">
            <span class="slider round"></span>
          </label>
        </div>

        <div class="card-details">
          <div class="detail-row">
            <span class="detail-label">🕐 Jadwal</span>
            <span>{{ describeCron(auto.cron_expression) }}</span>
          </div>
          <div class="detail-row" v-if="auto.target_wa">
            <span class="detail-label">📱 WA Tujuan</span>
            <span>{{ auto.target_wa }}</span>
          </div>
          <div class="detail-row" v-if="auto.last_run_at">
            <span class="detail-label">🔄 Terakhir Jalan</span>
            <span>{{ formatDate(auto.last_run_at) }}</span>
          </div>
          <div class="detail-row" v-if="auto.type === 'low_stock_alert' && auto.config?.threshold">
            <span class="detail-label">📦 Threshold</span>
            <span>≤ {{ auto.config.threshold }} item</span>
          </div>
        </div>

        <div class="card-actions">
          <button class="btn btn-secondary btn-sm" @click="editAutomation(auto)">✏️ Edit</button>
          <button class="btn btn-secondary btn-sm btn-danger" @click="deleteAutomation(auto.id)">🗑️ Hapus</button>
        </div>
      </div>
    </div>

    <!-- Toast -->
    <Teleport to="body">
      <div v-if="toast.visible" :class="['toast-notification', `toast-${toast.type}`]">
        {{ toast.message }}
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api } from '../api'

const automations = ref<any[]>([])
const planInfo = ref({ plan: 'lite', limit: 0, count: 0 })
const showForm = ref(false)
const editingId = ref<string | null>(null)
const saving = ref(false)

const form = ref({
  type: '',
  name: '',
  target_wa: '',
  threshold: 5,
})

const scheduleFreq = ref('daily')
const scheduleTime = ref('07:00')
const scheduleDay = ref('1')

const toast = ref({ visible: false, message: '', type: 'success' })
const showToast = (message: string, type: 'success' | 'error' = 'success') => {
  toast.value = { visible: true, message, type }
  setTimeout(() => { toast.value.visible = false }, 3000)
}

const typeDefaults: Record<string, { name: string; freq: string }> = {
  daily_report: { name: 'Laporan Harian', freq: 'daily' },
  weekly_report: { name: 'Laporan Mingguan', freq: 'weekly' },
  monthly_report: { name: 'Laporan Bulanan', freq: 'monthly' },
  low_stock_alert: { name: 'Alert Stok Rendah', freq: 'daily' },
}

const onTypeChange = () => {
  const d = typeDefaults[form.value.type]
  if (d) {
    if (!editingId.value) form.value.name = d.name
    scheduleFreq.value = d.freq
  }
}

const computedCron = computed(() => {
  const [h, m] = scheduleTime.value.split(':').map(Number)
  const minute = isNaN(m) ? 0 : m
  const hour = isNaN(h) ? 7 : h
  if (scheduleFreq.value === 'daily') return `${minute} ${hour} * * *`
  if (scheduleFreq.value === 'weekly') return `${minute} ${hour} * * ${scheduleDay.value}`
  if (scheduleFreq.value === 'monthly') return `${minute} ${hour} 1 * *`
  return `${minute} ${hour} * * *`
})

const getTypeIcon = (type: string) => {
  const icons: Record<string, string> = { daily_report: '📊', weekly_report: '📈', monthly_report: '📋', low_stock_alert: '⚠️' }
  return icons[type] || '⚡'
}
const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = { daily_report: 'Laporan Harian', weekly_report: 'Laporan Mingguan', monthly_report: 'Laporan Bulanan', low_stock_alert: 'Alert Stok Rendah' }
  return labels[type] || type
}

const describeCron = (cron: string) => {
  const parts = cron.split(' ')
  if (parts.length < 5) return cron
  const [min, hour, dom, , dow] = parts
  const time = `${hour.padStart(2, '0')}:${min.padStart(2, '0')}`
  const days = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu']
  if (dom === '1' && dow === '*') return `Setiap tanggal 1, jam ${time}`
  if (dow !== '*') return `Setiap ${days[parseInt(dow)] || dow}, jam ${time}`
  return `Setiap hari, jam ${time}`
}

const formatDate = (d: string) => {
  if (!d) return '-'
  return new Date(d).toLocaleString('id-ID', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

const loadAutomations = async () => {
  try {
    const res = await api.get('/api/umkm/automations')
    if (res.success && res.data) {
      automations.value = res.data.automations || []
      planInfo.value = {
        plan: res.data.plan || 'lite',
        limit: res.data.limit || 0,
        count: res.data.count || 0,
      }
    }
  } catch (e) {
    console.error('Failed to load automations', e)
  }
}

const saveAutomation = async () => {
  saving.value = true
  try {
    const config: any = {}
    if (form.value.type === 'low_stock_alert') {
      config.threshold = form.value.threshold || 5
    }
    const payload = {
      type: form.value.type,
      name: form.value.name,
      cron_expression: computedCron.value,
      target_wa: form.value.target_wa,
      config,
    }

    let res
    if (editingId.value) {
      res = await api.put(`/api/umkm/automations?id=${editingId.value}`, payload)
    } else {
      res = await api.post('/api/umkm/automations', payload)
    }

    if (res.success) {
      showToast(editingId.value ? 'Automasi berhasil diupdate' : 'Automasi berhasil dibuat')
      cancelForm()
      loadAutomations()
    } else {
      showToast(res.message || 'Gagal menyimpan', 'error')
    }
  } catch (e) {
    showToast('Kesalahan jaringan', 'error')
  } finally {
    saving.value = false
  }
}

const editAutomation = (auto: any) => {
  editingId.value = auto.id
  form.value.type = auto.type
  form.value.name = auto.name
  form.value.target_wa = auto.target_wa || ''
  form.value.threshold = auto.config?.threshold || 5

  // Parse cron back to UI
  const parts = auto.cron_expression.split(' ')
  if (parts.length >= 5) {
    const [min, hour, dom, , dow] = parts
    scheduleTime.value = `${hour.padStart(2, '0')}:${min.padStart(2, '0')}`
    if (dom === '1' && dow === '*') {
      scheduleFreq.value = 'monthly'
    } else if (dow !== '*') {
      scheduleFreq.value = 'weekly'
      scheduleDay.value = dow
    } else {
      scheduleFreq.value = 'daily'
    }
  }
  showForm.value = true
}

const cancelForm = () => {
  showForm.value = false
  editingId.value = null
  form.value = { type: '', name: '', target_wa: '', threshold: 5 }
  scheduleFreq.value = 'daily'
  scheduleTime.value = '07:00'
  scheduleDay.value = '1'
}

const toggleEnabled = async (auto: any) => {
  const newState = !auto.enabled
  const res = await api.put(`/api/umkm/automations?id=${auto.id}`, { enabled: newState })
  if (res.success) {
    auto.enabled = newState
    showToast(newState ? 'Automasi diaktifkan' : 'Automasi dinonaktifkan')
  }
}

const deleteAutomation = async (id: string) => {
  if (!confirm('Yakin ingin menghapus automasi ini?')) return
  const res = await api.del(`/api/umkm/automations?id=${id}`)
  if (res.success) {
    showToast('Automasi berhasil dihapus')
    loadAutomations()
  } else {
    showToast('Gagal menghapus', 'error')
  }
}

onMounted(() => {
  loadAutomations()
})
</script>

<style scoped>
.automations-page {
  max-width: 800px;
}

.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 1rem;
}

.plan-badge {
  display: inline-block;
  font-size: 0.7rem;
  padding: 0.2rem 0.6rem;
  border-radius: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.plan-lite { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
.plan-pro { background: rgba(59, 130, 246, 0.15); color: #60a5fa; }
.plan-ultimate { background: rgba(139, 92, 246, 0.2); color: #5b21b6; }
.plan-enterprise { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
.plan-superadmin { background: rgba(239, 68, 68, 0.15); color: #f87171; }

.usage-counter {
  font-size: 0.85rem;
  color: var(--text-secondary);
  background: var(--bg-tertiary);
  padding: 0.25rem 0.75rem;
  border-radius: 10px;
}

.upgrade-banner {
  display: flex;
  gap: 1.5rem;
  padding: 2rem;
  align-items: flex-start;
}
.upgrade-icon {
  font-size: 2.5rem;
  flex-shrink: 0;
}
.plan-comparison {
  display: flex;
  gap: 1.5rem;
  flex-wrap: wrap;
}
.plan-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9rem;
}

.add-automation-btn {
  margin-bottom: 1.5rem;
  width: 100%;
  padding: 1rem;
  font-size: 1rem;
  border: 2px dashed var(--border-color);
  background: transparent;
  color: var(--accent-primary);
  transition: all 0.2s;
}
.add-automation-btn:hover {
  background: rgba(59, 130, 246, 0.05);
  border-color: var(--accent-primary);
}

.limit-warning {
  text-align: center;
  color: #92400e;
  padding: 1rem;
  background: rgba(245, 158, 11, 0.15);
  border-radius: 8px;
  margin-bottom: 1.5rem;
}

.form-label {
  display: block;
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 0.4rem;
  font-weight: 500;
}

.form-control {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.95rem;
}

.cron-preview {
  margin-top: 0.5rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}
.cron-preview code {
  background: var(--bg-tertiary);
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  font-size: 0.8rem;
}

.empty-state {
  text-align: center;
  padding: 3rem 2rem;
}
.empty-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.automation-card {
  padding: 1.5rem;
  margin-bottom: 1rem;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}
.automation-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
.card-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.type-icon {
  font-size: 1.5rem;
}
.card-title {
  margin: 0;
  font-size: 1.05rem;
}
.type-label {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.card-details {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  margin-bottom: 1rem;
  padding: 0.75rem;
  background: rgba(255,255,255,0.02);
  border-radius: 6px;
}
.detail-row {
  display: flex;
  justify-content: space-between;
  font-size: 0.85rem;
}
.detail-label {
  color: var(--text-secondary);
}

.card-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-danger {
  color: #ef4444 !important;
  border-color: rgba(239,68,68,0.3) !important;
}
.btn-danger:hover {
  background: rgba(239,68,68,0.1) !important;
}

.btn-sm {
  padding: 0.3rem 0.8rem;
  font-size: 0.8rem;
}

/* Switch styling */
.switch {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 24px;
  flex-shrink: 0;
}
.switch input { opacity: 0; width: 0; height: 0; }
.slider {
  position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0;
  background-color: #ccc; transition: .4s;
}
.slider:before {
  position: absolute; content: ""; height: 16px; width: 16px; left: 4px; bottom: 4px;
  background-color: white; transition: .4s;
}
input:checked+.slider { background-color: #10b981; }
input:checked+.slider:before { transform: translateX(26px); }
.slider.round { border-radius: 24px; }
.slider.round:before { border-radius: 50%; }

/* Toast positioning handled by global main.css (top-right viewport) */
.toast-notification {
  padding: 0.875rem 1.25rem; border-radius: var(--radius-md);
  font-weight: 500; box-shadow: var(--shadow-lg);
}
.toast-success { background-color: #10b981; }
.toast-error { background: #ef4444; }

@media (max-width: 768px) {
  .upgrade-banner { flex-direction: column; }
  .plan-comparison { flex-direction: column; gap: 0.5rem; }
  .schedule-picker .flex { flex-direction: column; }
}
</style>
