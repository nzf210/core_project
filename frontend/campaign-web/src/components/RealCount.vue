<template>
  <div class="real-count-dashboard flex flex-col gap-4">
    <!-- Header & Summary -->
    <div class="header flex justify-between items-center">
      <div>
        <h2 style="font-size: 1.5rem; color: var(--accent-primary);">Real Count C1 & Saksi</h2>
        <p style="color: var(--text-secondary);">Pantau hasil suara dan kehadiran saksi secara Real-Time.</p>
      </div>
      <div>
        <button class="btn btn-primary" @click="fetchDashboard" :disabled="loading">
          {{ loading ? 'Memuat...' : 'Refresh Data' }}
        </button>
      </div>
    </div>

    <!-- Top KPI Cards -->
    <div class="stats-grid">
      <div class="stat-card glass-card border-accent">
        <h3>Suara Paslon Kita</h3>
        <p class="stat-number text-accent">{{ stats.candidate_votes || 0 }}</p>
      </div>
      <div class="stat-card glass-card" style="border-color: #ef4444;">
        <h3>Suara Lawan</h3>
        <p class="stat-number" style="color: #ef4444;">{{ stats.opponent_votes || 0 }}</p>
      </div>
      <div class="stat-card glass-card">
        <h3>Suara Tidak Sah / Batal</h3>
        <p class="stat-number" style="color: #f59e0b;">{{ stats.invalid_votes || 0 }}</p>
      </div>
      <div class="stat-card glass-card">
        <h3>Data TPS Masuk</h3>
        <p class="stat-number text-success">
          {{ stats.tps_reported || 0 }} / {{ stats.total_tps || 0 }}
        </p>
        <p style="font-size: 0.8rem; color: var(--text-secondary); margin-top: 0.5rem;">
          {{ tpsPercentage }}% Selesai
        </p>
      </div>
    </div>

    <!-- Tabs Navigation -->
    <div class="tabs flex gap-4" style="margin-top: 1rem; border-bottom: 1px solid var(--border-color); padding-bottom: 0.5rem;">
      <button 
        :class="['tab-btn', activeTab === 'dashboard' ? 'active' : '']"
        @click="activeTab = 'dashboard'">
        📊 Dashboard Hasil
      </button>
      <button 
        :class="['tab-btn', activeTab === 'saksi' ? 'active' : '']"
        @click="activeTab = 'saksi'">
        👥 Absensi Saksi
      </button>
      <button 
        :class="['tab-btn', activeTab === 'review' ? 'active' : '']"
        @click="activeTab = 'review'">
        ⚠️ Antrian Verifikasi C1 (AI Mismatch)
      </button>
    </div>

    <!-- TAB 1: Dashboard -->
    <div v-if="activeTab === 'dashboard'" class="tab-content glass-card" style="padding: 2rem; text-align: center;">
      <h3 style="margin-bottom: 1rem;">Progress Data Masuk</h3>
      <!-- Simple Progress Bar -->
      <div style="width: 100%; background: var(--surface-1); height: 24px; border-radius: 12px; overflow: hidden; margin-bottom: 2rem;">
        <div 
          :style="{ width: tpsPercentage + '%', background: 'var(--accent-primary)', height: '100%', transition: 'width 1s ease' }">
        </div>
      </div>
      <p style="color: var(--text-secondary);">Fitur pemetaan grafis/chart detail akan ditambahkan pada rilis berikutnya.</p>
    </div>

    <!-- TAB 2: Absensi Saksi -->
    <div v-if="activeTab === 'saksi'" class="tab-content">
      <div class="table-container glass-card">
        <table>
          <thead>
            <tr>
              <th>TPS</th>
              <th>Nama Saksi</th>
              <th>Status Hadir (Jam 07:00)</th>
              <th>Waktu Lapor</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="attendances.length === 0">
              <td colspan="4" class="text-center" style="padding: 2rem; color: var(--text-secondary);">
                Belum ada data saksi yang melapor.
              </td>
            </tr>
            <tr v-for="att in attendances" :key="att.id">
              <td>{{ att.tps_name }}</td>
              <td>{{ att.volunteer_name }}</td>
              <td>
                <span :class="['badge', att.status === 'present' ? 'badge-success' : 'badge-danger']">
                  {{ att.status === 'present' ? 'Hadir' : 'Bolos/Telat' }}
                </span>
              </td>
              <td>{{ att.verified_at }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- TAB 3: Human Review -->
    <div v-if="activeTab === 'review'" class="tab-content">
      <div v-if="pendingReviews.length === 0" class="glass-card text-center" style="padding: 3rem;">
        <h3 style="color: #10b981;">✅ Bersih!</h3>
        <p style="color: var(--text-secondary); margin-top: 0.5rem;">Tidak ada dokumen C1 yang perlu divalidasi manual. AI Vision membaca semua data dengan akurat.</p>
      </div>

      <div class="grid-cards" v-else>
        <div v-for="rev in pendingReviews" :key="rev.id" class="glass-card review-card flex flex-col gap-4">
          <div style="background: #000; width: 100%; height: 200px; display: flex; align-items: center; justify-content: center; border-radius: 8px; overflow: hidden;">
            <img :src="rev.c1_image_url" alt="C1 Plano" style="max-height: 100%; max-width: 100%; object-fit: contain;" />
          </div>
          <div>
            <h4>TPS: {{ rev.tps_name }}</h4>
            <p style="color: #ef4444; font-size: 0.85rem; margin-bottom: 1rem;">⚠️ AI Mismatch: {{ rev.notes }}</p>
            
            <div class="comparison-grid">
              <div class="col">
                <strong>Input Saksi</strong>
                <div>Paslon: {{ rev.reported_candidate_votes }}</div>
                <div>Lawan: {{ rev.reported_opponent_votes }}</div>
                <div>Batal: {{ rev.reported_invalid_votes }}</div>
              </div>
              <div class="col">
                <strong>BACAAN AI</strong>
                <div>Paslon: {{ rev.ai_candidate_votes }}</div>
                <div>Lawan: {{ rev.ai_opponent_votes }}</div>
                <div>Batal: {{ rev.ai_invalid_votes }}</div>
              </div>
            </div>
          </div>
          <div class="actions flex gap-2" style="margin-top: auto;">
            <button class="btn btn-success flex-1">Terima (Input Saksi)</button>
            <button class="btn btn-primary flex-1">Terima (Bacaan AI)</button>
            <button class="btn btn-danger flex-1">Tolak</button>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { apiClient } from '../api'

const activeTab = ref('dashboard')
const loading = ref(false)
const stats = ref<any>({})
const attendances = ref<any[]>([])
const pendingReviews = ref<any[]>([])

// Placeholder Campaign ID, in production this comes from Vuex/Pinia or Dropdown
const ACTIVE_CAMPAIGN_ID = "00000000-0000-0000-0000-000000000000" 

const tpsPercentage = computed(() => {
  if (!stats.value.total_tps || stats.value.total_tps === 0) return 0
  return Math.round((stats.value.tps_reported / stats.value.total_tps) * 100)
})

const fetchDashboard = async () => {
  loading.value = true
  try {
    const res = await apiClient(`/real-count?campaign_id=${ACTIVE_CAMPAIGN_ID}`)
    const data = await res.json()
    if (data.success) {
      stats.value = data.data
    }
    // MOCK DATA for tabs 2 & 3 since APIs are pending
    attendances.value = []
    pendingReviews.value = []
  } catch (e) {
    console.error(e)
  }
  loading.value = false
}

onMounted(() => {
  fetchDashboard()
})
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1.5rem;
  margin-bottom: 1rem;
}
.stat-card {
  padding: 1.5rem;
  text-align: center;
  border-radius: var(--radius-md);
  background: var(--surface-0);
  border: 1px solid var(--border-color);
}
.stat-card h3 {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
  font-weight: 500;
}
.stat-number {
  font-size: 2.2rem;
  font-weight: 700;
  color: var(--text-primary);
}
.border-accent { border-color: var(--accent-primary); background: rgba(59, 130, 246, 0.05); }
.text-accent { color: var(--accent-primary); }
.text-success { color: #10b981; }

.tab-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  padding: 0.5rem 1rem;
  font-weight: 500;
  cursor: pointer;
  border-bottom: 2px solid transparent;
}
.tab-btn.active {
  color: var(--accent-primary);
  border-bottom: 2px solid var(--accent-primary);
}
.tab-btn:hover { color: var(--text-primary); }

.grid-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}
.review-card { padding: 1rem; }
.comparison-grid {
  display: flex;
  gap: 1rem;
  background: var(--surface-1);
  padding: 0.75rem;
  border-radius: 8px;
  font-size: 0.85rem;
}
.col { flex: 1; }
.col div { margin-top: 0.25rem; }

table { width: 100%; border-collapse: collapse; }
th, td { padding: 1rem; text-align: left; border-bottom: 1px solid var(--border-color); }
th { color: var(--text-secondary); font-weight: 500; }
</style>