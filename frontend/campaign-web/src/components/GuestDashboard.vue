<template>
  <div class="guest-dashboard">
    <header class="top-header flex justify-between items-center glass-card"
      style="margin: 1rem 2rem; border-radius: var(--radius-md)">
      <div class="logo">
        <h2 style="font-weight: 800; color: var(--text-primary)">Campaign<span class="text-gradient">Manager</span></h2>
      </div>
      <div>
        <button class="btn-outline" @click="$emit('login')" type="button"
          style="border-color: var(--accent-primary); color: var(--accent-primary);">
          Masuk (Login)
        </button>
      </div>
    </header>

    <div class="hero-section">
      <div class="hero-content">
        <h2>Kawal Suara Rakyat Bersama Kami!</h2>
        <p>Lihat perkembangan dukungan secara transparan dan berikan kontribusi Anda sekarang.</p>

        <!-- Dropdown Filter Daerah -->
        <div class="region-filter-wrapper flex gap-2 items-center" style="margin-top: 1rem; margin-bottom: 2rem; justify-content: center;">
          <label for="select-region-type" style="color: var(--text-secondary); font-weight: 500;">Tampilkan Data di:</label>
          <select id="select-region-type" v-model="selectedRegionType" @change="fetchPublicStats" class="form-input" style="width: auto; display: inline-block; padding: 0.5rem 1rem;">
            <option value="nasional">Nasional (Seluruh Indonesia)</option>
            <option value="province">Tingkat Provinsi</option>
            <option value="regency">Tingkat Kabupaten/Kota</option>
          </select>

          <label for="select-province" v-if="selectedRegionType === 'province'" style="color: var(--text-secondary); font-weight: 500;">Provinsi:</label>
          <select id="select-province" v-if="selectedRegionType === 'province'" v-model="selectedRegionId" @change="fetchPublicStats" class="form-input" style="width: auto; display: inline-block; padding: 0.5rem 1rem;">
            <option value="">-- Pilih Provinsi --</option>
            <option v-for="prov in provinces" :key="prov.id" :value="prov.id">{{ prov.name }}</option>
          </select>

          <label for="select-regency" v-if="selectedRegionType === 'regency'" style="color: var(--text-secondary); font-weight: 500;">Kabupaten:</label>
          <select id="select-regency" v-if="selectedRegionType === 'regency'" v-model="selectedRegionId" @change="fetchPublicStats" class="form-input" style="width: auto; display: inline-block; padding: 0.5rem 1rem;">
            <option value="">-- Pilih Kabupaten --</option>
            <!-- Mock data for regencies right now -->
            <option value="1">Kota Jakarta Selatan</option>
            <option value="2">Kabupaten Bogor</option>
          </select>
        </div>

        <div class="dashboard-grid">
          <div class="stats-card glass-card candidates-list">
            <h3 class="stats-title">Kandidat Teratas</h3>

            <div v-if="isLoading" class="loading-state">Mengambil data...</div>

            <div v-else-if="topCandidates.length > 0" class="candidate-stat" v-for="c in topCandidates" :key="c.id">
              <div class="candidate-avatar" :style="{ background: c.color }">{{ c.name.charAt(0) }}</div>
              <div class="candidate-info">
                <h4>{{ c.name }}</h4>
                <div class="progress-container">
                  <div class="progress-bar" :style="{ width: c.electability_percentage + '%', background: c.color }">
                  </div>
                </div>
                <p class="stat-detail">
                  <strong :style="{ color: c.color }">{{ c.electability_percentage }}%</strong> ({{ c.total_votes }}
                  dukungan)
                </p>
              </div>
            </div>

            <div v-else class="error-state">Data belum tersedia</div>
          </div>

          <div class="stats-card glass-card map-container">
            <h3 class="stats-title">Peta Persebaran Dukungan</h3>
            <div class="mock-map">
              <!-- Simple mock SVG map -->
              <svg viewBox="0 0 400 300" class="svg-map">
                <path d="M50,150 Q100,50 200,100 T350,150 Q300,250 200,200 T50,150"
                  :fill="mapData.region_1 || '#cbd5e1'" stroke="#fff" stroke-width="2" />
                <circle cx="150" cy="120" r="30" :fill="mapData.region_2 || '#cbd5e1'" stroke="#fff" stroke-width="2" />
                <rect x="220" y="140" width="80" height="60" rx="10" :fill="mapData.region_3 || '#cbd5e1'" stroke="#fff"
                  stroke-width="2" />
                <polygon points="100,200 150,250 80,260" :fill="mapData.region_4 || '#cbd5e1'" stroke="#fff"
                  stroke-width="2" />
              </svg>
              <div class="map-legend" v-if="topCandidates.length > 0">
                <div class="legend-item" v-for="c in topCandidates" :key="c.id">
                  <span class="color-dot" :style="{ background: c.color }"></span>
                  {{ c.name.split(' ')[0] }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="action-buttons">
          <button type="button" class="btn-primary" @click="$emit('register', 'relawan')">
            <span class="icon">✋</span> Daftar Menjadi Relawan
          </button>
          <button type="button" class="btn-secondary" @click="$emit('register', 'user_biasa')">
            <span class="icon">👤</span> Daftar Menjadi User Biasa
          </button>
          <button type="button" class="btn-outline" @click="$emit('register', 'kandidat')">
            <span class="icon">👑</span> Daftar Menjadi Kandidat
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { publicApi } from '../api'

const emit = defineEmits(['login', 'register'])

const selectedRegionType = ref('nasional')
const selectedRegionId = ref('')
const provinces = ref<any[]>([
  { id: '11', name: 'DKI Jakarta' },
  { id: '12', name: 'Jawa Barat' },
  { id: '13', name: 'Jawa Tengah' },
  { id: '14', name: 'Jawa Timur' }
])

const topCandidates = ref<any[]>([])
const mapData = ref<any>({})
const isLoading = ref(true)

const fetchPublicStats = async () => {
  try {
    isLoading.value = true
    const res = await publicApi.getDashboard(selectedRegionType.value, selectedRegionId.value)
    const data = await res.json()
    if (data.success && data.data) {
      if (data.data.top_candidates) {
        topCandidates.value = data.data.top_candidates
      }
      if (data.data.map_data) {
        mapData.value = data.data.map_data
      }
    }
  } catch (err) {
    console.error("Gagal mengambil data publik", err)
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  fetchPublicStats()
})
</script>

<style scoped>
.guest-dashboard {
  min-height: 100vh;
  background: var(--bg-primary);
  display: flex;
  flex-direction: column;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem 3rem;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.logo-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 800;
}

.btn-outline {
  background: transparent;
  border: 1px solid var(--accent-primary);
  color: var(--accent-primary);
  padding: 0.5rem 1.5rem;
  border-radius: var(--radius-sm);
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-outline:hover {
  background: rgba(220, 38, 38, 0.1);
}

.hero-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 3rem;
}

.hero-content {
  max-width: 1000px;
  width: 100%;
  text-align: center;
}

.hero-content h2 {
  font-size: 2.5rem;
  margin-bottom: 1rem;
  background: var(--accent-gradient);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.hero-content>p {
  font-size: 1.1rem;
  color: var(--text-secondary);
  margin-bottom: 3rem;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
  margin-bottom: 3rem;
}

.stats-card {
  background: var(--bg-secondary);
  padding: 2rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-md);
  text-align: left;
}

.stats-title {
  margin: 0 0 1.5rem 0;
  font-size: 1.2rem;
  color: var(--text-primary);
  text-align: center;
}

.candidates-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.candidate-stat {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.5rem;
  border-radius: 8px;
  transition: background 0.2s;
}

.candidate-stat:hover {
  background: var(--bg-tertiary);
}

.candidate-avatar {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
  font-weight: bold;
}

.candidate-info {
  flex: 1;
}

.candidate-info h4 {
  margin: 0 0 0.25rem 0;
  font-size: 1.1rem;
}

.progress-container {
  height: 8px;
  background: var(--bg-tertiary);
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 0.25rem;
}

.progress-bar {
  height: 100%;
  transition: width 1s ease-out;
}

.stat-detail {
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.map-container {
  display: flex;
  flex-direction: column;
}

.mock-map {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
}

.svg-map {
  width: 100%;
  max-height: 250px;
  filter: drop-shadow(0 4px 6px rgba(0, 0, 0, 0.1));
}

.map-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin-top: 1rem;
  justify-content: center;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.color-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.action-buttons {
  display: flex;
  gap: 1.5rem;
  justify-content: center;
}

.action-buttons button {
  padding: 1rem 2rem;
  border-radius: var(--radius-md);
  font-size: 1.1rem;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border: none;
  transition: transform 0.2s;
}

.action-buttons button:hover {
  transform: translateY(-2px);
}

.btn-primary {
  background: var(--accent-gradient);
  color: white;
  box-shadow: 0 4px 15px rgba(220, 38, 38, 0.3);
}

.btn-secondary {
  background: var(--surface-0);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  box-shadow: var(--shadow-sm);
}

@media (max-width: 768px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .action-buttons {
    flex-direction: column;
  }
}
</style>
