<template>
  <div class="dashboard flex flex-col gap-4">
    <div class="header">
      <h2 style="font-size: 1.5rem; color: var(--accent-primary);">Campaign Dashboard</h2>
      <p style="color: var(--text-secondary);">Pantau performa kampanye dan pergerakan pemilih.</p>
    </div>

    <div class="stats-grid">
      <div class="stat-card glass-card">
        <h3>Pemilih Dihubungi</h3>
        <p class="stat-number text-success">{{ voterStats.by_status?.contacted || 0 }}</p>
      </div>
      <div class="stat-card glass-card">
        <h3>Belum Dihubungi</h3>
        <p class="stat-number">{{ voterStats.by_status?.uncontacted || 0 }}</p>
      </div>
      <div class="stat-card glass-card border-accent">
        <h3>Potensi Suara (Pasti)</h3>
        <p class="stat-number text-accent">{{ voterStats.potential?.high || 0 }}</p>
      </div>
      <div class="stat-card glass-card">
        <h3>Potensi Suara (Ragu)</h3>
        <p class="stat-number" style="color: #f59e0b;">{{ voterStats.potential?.medium || 0 }}</p>
      </div>
      <div class="stat-card glass-card" style="border-color: #ef4444;">
        <h3>Pendukung Calon Lain</h3>
        <p class="stat-number" style="color: #ef4444;">{{ totalCompetitors }}</p>
      </div>
      <div class="stat-card glass-card">
        <h3>Total Relawan Aktif</h3>
        <p class="stat-number">{{ stats.total_volunteers || 0 }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'
import { ref, onMounted, computed } from 'vue'

const stats = ref<any>({})
const voterStats = ref<any>({})

const totalCompetitors = computed(() => {
  if (!voterStats.value.competitors) return 0;
  return Object.values(voterStats.value.competitors).reduce((a: any, b: any) => a + b, 0);
})

onMounted(async () => {
  try {
    const res = await apiClient('/volunteers/stats')
    const data = await res.json()
    if (data.success) {
      stats.value = data.data
    }

    const resVoter = await apiClient('/voters/stats')
    const dataVoter = await resVoter.json()
    if (dataVoter.success) {
      voterStats.value = dataVoter.data
    }
  } catch (err) {
    console.error('Failed to fetch stats', err)
  }
})
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1.5rem;
  margin-top: 1rem;
}
.stat-card {
  padding: 1.5rem;
  text-align: center;
  border-radius: var(--radius-md);
  background: var(--surface-0);
  border: 1px solid var(--border-color);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}
.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.2);
}
.stat-card h3 {
  font-size: 0.9rem;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
  font-weight: 500;
}
.stat-number {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
}
.border-accent {
  border-color: var(--accent-primary);
  background: rgba(59, 130, 246, 0.05);
}
.text-accent {
  color: var(--accent-primary);
}
.text-success {
  color: #10b981;
}
.flex { display: flex; }
.flex-col { flex-direction: column; }
.gap-4 { gap: 1rem; }
</style>
