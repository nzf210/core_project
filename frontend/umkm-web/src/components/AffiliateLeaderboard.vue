<template>
  <div class="leaderboard-page">
    <div class="hero">
      <h1>🏆 Papan Peringkat Agen</h1>
      <p>10 agen afiliasi terbaik WCH Platform — diurutkan berdasarkan total pendapatan</p>
    </div>

    <div v-if="loading" class="loading-block">
      <div class="spinner"></div>
      <p>Memuat...</p>
    </div>

    <div v-else-if="errorMsg" class="error-block">
      <p>{{ errorMsg }}</p>
      <button class="btn btn-outline" @click="loadLeaderboard">🔄 Coba Lagi</button>
    </div>

    <div v-else-if="leaders.length === 0" class="empty-block">
      <p>🔔 Belum ada agen terdaftar. Jadilah yang pertama!</p>
    </div>

    <div v-else class="leader-list">
      <div 
        v-for="(leader, idx) in leaders" 
        :key="idx" 
        class="leader-card"
        :class="`rank-${idx + 1}`"
      >
        <div class="rank">
          <span v-if="idx === 0">🥇</span>
          <span v-else-if="idx === 1">🥈</span>
          <span v-else-if="idx === 2">🥉</span>
          <span v-else class="rank-number">#{{ idx + 1 }}</span>
        </div>

        <div class="info">
          <div class="name">{{ leader.name }}</div>
          <div class="meta">
            <span>{{ leader.total_closing }} closing</span>
            <span class="dot">·</span>
            <span class="revenue">Rp {{ formatRupiah(leader.total_revenue_rupiah) }}</span>
          </div>
        </div>

        <div class="bar-container">
          <div class="bar" :style="{ width: barWidth(leader) }"></div>
        </div>
      </div>
    </div>

    <div v-if="leaders.length > 0" class="footer-cta">
      <router-link to="/login" class="btn btn-primary">🔗 Jadi Agen Sekarang</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

interface Leader {
  name: string
  total_closing: number
  total_revenue_rupiah: number
}

const loading = ref(true)
const errorMsg = ref('')
const leaders = ref<Leader[]>([])

function formatRupiah(rupiah: number): string {
  const rp = Math.floor(rupiah)
  return rp.toLocaleString('id-ID')
}

function barWidth(leader: Leader): string {
  if (leaders.value.length === 0) return '0%'
  const max = leaders.value[0].total_revenue_rupiah
  if (max === 0) return '0%'
  const pct = (leader.total_revenue_rupiah / max) * 100
  return Math.max(pct, 5) + '%'  // min 5% biar keliatan
}

async function loadLeaderboard() {
  loading.value = true
  errorMsg.value = ''
  try {
    const res = await api.getAffiliateLeaderboard()
    // Backend returns {status, message, data}; leaderboard array is in res.data
    if (res && res.data && Array.isArray(res.data)) {
      leaders.value = res.data as Leader[]
    } else {
      errorMsg.value = res?.message || 'Gagal memuat leaderboard'
    }
  } catch (e) {
    console.warn('Leaderboard load failed:', e)
    errorMsg.value = 'Kesalahan jaringan'
  } finally {
    loading.value = false
  }
}

onMounted(loadLeaderboard)
</script>

<style scoped>
.leaderboard-page {
  max-width: 640px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.hero {
  text-align: center;
  margin-bottom: 2rem;
}
.hero h1 {
  font-size: 1.75rem;
  margin-bottom: 0.5rem;
}
.hero p {
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.loading-block, .error-block, .empty-block {
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

/* --- Leader Cards --- */
.leader-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.leader-card {
  background: var(--surface-0);
  border-radius: var(--radius-lg);
  padding: 1rem 1.25rem;
  display: flex;
  align-items: center;
  gap: 1rem;
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
  position: relative;
  overflow: hidden;
}

.leader-card.rank-1 { border-left: 3px solid #f59e0b; }
.leader-card.rank-2 { border-left: 3px solid #94a3b8; }
.leader-card.rank-3 { border-left: 3px solid #d97706; }

.rank {
  font-size: 1.5rem;
  flex-shrink: 0;
  width: 40px;
  text-align: center;
}

.rank-number {
  font-weight: 800;
  color: var(--text-secondary);
  font-size: 1rem;
}

.info {
  flex: 1;
  min-width: 0;
}

.name {
  font-weight: 700;
  font-size: 1.05rem;
  margin-bottom: 0.25rem;
}

.meta {
  font-size: 0.8rem;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.dot { color: var(--text-secondary); }

.revenue {
  color: #f59e0b;
  font-weight: 600;
}

/* Bar */
.bar-container {
  flex-shrink: 0;
  width: 80px;
  height: 6px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
}

.bar {
  height: 100%;
  background: linear-gradient(90deg, var(--accent-primary), var(--accent-secondary));
  border-radius: 3px;
  transition: width 0.5s ease;
}

/* Footer */
.footer-cta {
  text-align: center;
  margin-top: 2rem;
}

.btn {
  display: inline-block;
  padding: 0.75rem 2rem;
  border: none;
  border-radius: var(--radius-sm);
  font-family: inherit;
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  text-decoration: none;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--accent-primary);
  color: #fff;
}
.btn-primary:hover { filter: brightness(1.1); }

.btn-outline {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-primary);
}
.btn-outline:hover { background: var(--bg-tertiary); }
</style>
