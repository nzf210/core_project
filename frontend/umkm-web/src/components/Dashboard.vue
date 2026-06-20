<template>
  <div class="dashboard">
    <div class="header-actions flex items-center justify-between">
      <div>
        <h2>Ringkasan Keuangan</h2>
        <p>Bulan ini vs Bulan lalu</p>
      </div>
      <button class="btn btn-primary" @click="syncData">
        Sync Data AI
      </button>
    </div>

    <div class="metrics-grid-bento">
      <div class="metric-card">
        <div class="flex items-center justify-between" style="margin-bottom: 0.5rem;">
          <span class="text-secondary" style="font-size: 0.9rem; font-weight: 500;">Total Pemasukan</span>
          <span :class="['badge', revPercent >= 0 ? 'badge-success' : 'badge-danger']">
            {{ revPercent > 0 ? '+' : '' }}{{ revPercent.toFixed(1) }}%
          </span>
        </div>
        <div style="font-size: 2rem; font-weight: 700; color: var(--text-primary); margin: 0.5rem 0;">
          Rp {{ incomeStatement.revenue.toLocaleString('id-ID') }}
        </div>
        <span class="text-muted" style="font-size: 0.85rem;">Laporan Laba Rugi</span>
      </div>

      <div class="metric-card">
        <div class="flex items-center justify-between" style="margin-bottom: 0.5rem;">
          <span class="text-secondary" style="font-size: 0.9rem; font-weight: 500;">Total Pengeluaran</span>
          <span :class="['badge', expPercent > 0 ? 'badge-danger' : 'badge-success']">
            {{ expPercent > 0 ? '+' : '' }}{{ expPercent.toFixed(1) }}%
          </span>
        </div>
        <div style="font-size: 2rem; font-weight: 700; color: var(--text-primary); margin: 0.5rem 0;">
          Rp {{ incomeStatement.expense.toLocaleString('id-ID') }}
        </div>
        <span class="text-muted" style="font-size: 0.85rem;">Laporan Laba Rugi</span>
      </div>

      <div class="metric-card">
        <div class="flex items-center justify-between" style="margin-bottom: 0.5rem;">
          <span class="text-secondary" style="font-size: 0.9rem; font-weight: 500;">Laba Bersih</span>
          <span :class="['badge', netPercent >= 0 ? 'badge-success' : 'badge-danger']">
            {{ netPercent > 0 ? '+' : '' }}{{ netPercent.toFixed(1) }}%
          </span>
        </div>
        <div style="font-size: 2rem; font-weight: 700; color: var(--text-primary); margin: 0.5rem 0;">
          Rp {{ incomeStatement.net_income.toLocaleString('id-ID') }}
        </div>
        <span class="text-muted" style="font-size: 0.85rem;">Laporan Laba Rugi</span>
      </div>

      <div class="metric-card">
        <div class="flex items-center justify-between" style="margin-bottom: 0.5rem;">
          <span class="text-secondary" style="font-size: 0.9rem; font-weight: 500;">Total Aset (Neraca)</span>
        </div>
        <div style="font-size: 2rem; font-weight: 700; color: var(--text-primary); margin: 0.5rem 0;">
          Rp {{ balanceSheet.assets.toLocaleString('id-ID') }}
        </div>
        <span class="text-muted" style="font-size: 0.85rem;">
          Kewajiban: Rp {{ balanceSheet.liabilities.toLocaleString('id-ID') }}
        </span>
      </div>
    </div>

    <div class="chart-container glass-card">
      <h3 style="margin-bottom: 1.5rem;">Grafik Arus Kas (Bulan Ini)</h3>
      <div style="height: 300px; width: 100%;">
        <Bar v-if="chartData.labels" :data="chartData" :options="chartOptions" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Bar } from 'vue-chartjs'
import { Chart as ChartJS, Title, Tooltip, Legend, BarElement, CategoryScale, LinearScale } from 'chart.js'
import { api } from '../api'

ChartJS.register(Title, Tooltip, Legend, BarElement, CategoryScale, LinearScale)

const incomeStatement = ref({ revenue: 0, expense: 0, net_income: 0 })
const incomeStatementPrev = ref({ revenue: 0, expense: 0, net_income: 0 })
const balanceSheet = ref({ assets: 0, liabilities: 0, equity: 0 })

const revPercent = ref(0)
const expPercent = ref(0)
const netPercent = ref(0)

const calculatePercent = (current: number, prev: number) => {
  if (prev === 0) return current > 0 ? 100 : 0
  return ((current - prev) / prev) * 100
}

const chartData = ref({
  labels: ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'],
  datasets: [
    {
      label: 'Pemasukan',
      backgroundColor: '#3b82f6',
      data: [0, 0, 0, 0, 0, 0, 0]
    },
    {
      label: 'Pengeluaran',
      backgroundColor: '#ef4444',
      data: [0, 0, 0, 0, 0, 0, 0]
    }
  ]
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      labels: { color: '#e2e8f0' }
    }
  },
  scales: {
    y: {
      grid: { color: 'rgba(255, 255, 255, 0.1)' },
      ticks: { color: '#94a3b8' }
    },
    x: {
      grid: { display: false },
      ticks: { color: '#94a3b8' }
    }
  }
}

const syncData = async () => {
  try {
    const tenantID = localStorage.getItem('tenant_id')
    if (!tenantID) {
      alert("Session tenant_id tidak ditemukan, silakan login ulang.")
      return
    }

    // Fetch Income Statement
    const today = new Date().toISOString().split('T')[0]
    const lastMonth = new Date(new Date().setDate(new Date().getDate() - 30)).toISOString().split('T')[0]
    const twoMonthsAgo = new Date(new Date().setDate(new Date().getDate() - 60)).toISOString().split('T')[0]
    
    // Previous period
    const isPrevData = await api.get(`/api/umkm/reports/income-statement?from=${twoMonthsAgo}&to=${lastMonth}`)
    if (isPrevData.success && isPrevData.data) {
      incomeStatementPrev.value = {
        revenue: isPrevData.data.revenue || 0,
        expense: isPrevData.data.expense || 0,
        net_income: isPrevData.data.net_income || 0
      }
    }

    // Current period
    const isData = await api.get(`/api/umkm/reports/income-statement?from=${lastMonth}&to=${today}`)
    if (isData.success && isData.data) {
      incomeStatement.value = {
        revenue: isData.data.revenue || 0,
        expense: isData.data.expense || 0,
        net_income: isData.data.net_income || 0
      }
      
      revPercent.value = calculatePercent(incomeStatement.value.revenue, incomeStatementPrev.value.revenue)
      expPercent.value = calculatePercent(incomeStatement.value.expense, incomeStatementPrev.value.expense)
      netPercent.value = calculatePercent(incomeStatement.value.net_income, incomeStatementPrev.value.net_income)

      const dailyRev = Math.round((isData.data.revenue || 0) / 7)
      const dailyExp = Math.round((isData.data.expense || 0) / 7)
      chartData.value = {
        labels: ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu'],
        datasets: [
          { label: 'Pemasukan', backgroundColor: '#3b82f6', data: Array(7).fill(dailyRev) },
          { label: 'Pengeluaran', backgroundColor: '#ef4444', data: Array(7).fill(dailyExp) }
        ]
      }
    }

    // 3. Fetch Balance Sheet
    const bsData = await api.get(`/api/umkm/reports/balance-sheet?date=${today}`)
    if (bsData.success && bsData.data) {
      balanceSheet.value = {
        assets: bsData.data.assets || 0,
        liabilities: bsData.data.liabilities || 0,
        equity: bsData.data.equity || 0
      }
    }
  } catch (e) {
    console.error("Gagal sinkronisasi data:", e)
  }
}

onMounted(() => {
  syncData()
})
</script>

<style scoped>
.header-actions {
  margin-bottom: 2rem;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2.5rem;
}

.metric-card {
  display: flex;
  flex-direction: column;
}

.metric-title {
  color: var(--text-secondary);
  font-weight: 500;
  font-size: 1.1rem;
}

.metric-value {
  font-size: 2.5rem;
  font-weight: 700;
  margin: 1rem 0 0.5rem;
}

.metric-subtitle {
  color: var(--text-muted);
  font-size: 0.875rem;
}

.chart-container {
  padding: 2rem;
}

.placeholder-chart {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  height: 200px;
  gap: 1rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.bar {
  flex: 1;
  background-color: var(--bg-tertiary);
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  transition: height 1s ease-out;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .header-actions.flex.items-center.justify-between {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }
  
  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .chart-container {
    padding: 1rem;
  }

  .metric-value {
    font-size: 2rem;
  }
}
</style>
