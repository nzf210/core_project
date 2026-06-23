<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api/client'
import { Chart as ChartJS, ArcElement, Tooltip, Legend, CategoryScale, LinearScale, BarElement, Title } from 'chart.js'
import { Pie, Bar } from 'vue-chartjs'

ChartJS.register(ArcElement, Tooltip, Legend, CategoryScale, LinearScale, BarElement, Title)

const analytics = ref<any>(null)
const loading = ref(true)
const error = ref('')
const selectedProgram = ref('')
const programs = ref<any[]>([])

onMounted(async () => {
  await load()
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    programs.value = (await api.listVoucherPrograms()).data || []
    analytics.value = await api.getVoucherAnalytics(selectedProgram.value || undefined)
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function filterByProgram() {
  await load()
}

const redemptionChartData = () => ({
  labels: ['Redeemed', 'Unredeemed'],
  datasets: [{
    data: [
      analytics.value?.total_redeemed || 0,
      (analytics.value?.total_generated || 0) - (analytics.value?.total_redeemed || 0)
    ],
    backgroundColor: ['#10b981', '#e5e7eb'],
  }],
})

const usageChartData = () => ({
  labels: analytics.value?.usage_by_tenant?.map((u: any) => u.tenant_name) || [],
  datasets: [{
    label: 'Redemptions',
    data: analytics.value?.usage_by_tenant?.map((u: any) => u.count) || [],
    backgroundColor: '#3b82f6',
  }],
})
</script>

<template>
  <div>
    <div class="header">
      <h1>Voucher Analytics</h1>
      <div class="filter">
        <select v-model="selectedProgram" @change="filterByProgram">
          <option value="">All Programs</option>
          <option v-for="p in programs" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>
    </div>

    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="analytics">
      <div class="metrics">
        <div class="metric-card">
          <div class="metric-label">Total Generated</div>
          <div class="metric-value">{{ analytics.total_generated || 0 }}</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">Total Redeemed</div>
          <div class="metric-value">{{ analytics.total_redeemed || 0 }}</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">Redemption Rate</div>
          <div class="metric-value">{{ ((analytics.total_redeemed || 0) / (analytics.total_generated || 1)).toFixed(1) }}%</div>
        </div>
        <div class="metric-card">
          <div class="metric-label">Revenue Impact</div>
          <div class="metric-value">Rp {{ (analytics.revenue_impact || 0).toLocaleString('id-ID') }}</div>
        </div>
      </div>

      <div class="charts">
        <div class="chart-container">
          <h3>Redemption Status</h3>
          <Pie :data="redemptionChartData()" :options="{ responsive: true, maintainAspectRatio: true }" />
        </div>
        <div class="chart-container">
          <h3>Usage by Tenant</h3>
          <Bar :data="usageChartData()" :options="{ responsive: true, maintainAspectRatio: true, indexAxis: 'y' }" />
        </div>
      </div>

      <div class="details">
        <h3>Top Tenants</h3>
        <table v-if="analytics.usage_by_tenant?.length">
          <thead>
            <tr><th>Tenant</th><th>Redemptions</th><th>Revenue</th></tr>
          </thead>
          <tbody>
            <tr v-for="u in analytics.usage_by_tenant" :key="u.tenant_id">
              <td>{{ u.tenant_name }}</td>
              <td>{{ u.count }}</td>
              <td>Rp {{ (u.revenue || 0).toLocaleString('id-ID') }}</td>
            </tr>
          </tbody>
        </table>
        <div v-else class="empty">No redemptions yet</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px; }
h1 { font-size: 24px; margin: 0; }
.filter { display: flex; gap: 12px; }
.filter select { padding: 8px 12px; border: 1px solid var(--border); border-radius: 6px; }
.loading, .error { padding: 40px; text-align: center; }
.error { color: var(--danger); }
.metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 32px; }
.metric-card { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 20px; }
.metric-label { font-size: 13px; color: var(--muted); margin-bottom: 8px; }
.metric-value { font-size: 28px; font-weight: 700; }
.charts { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 24px; margin-bottom: 32px; }
.chart-container { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 20px; }
.chart-container h3 { margin-top: 0; }
.details { background: var(--card); border: 1px solid var(--border); border-radius: 10px; padding: 20px; }
table { width: 100%; border-collapse: collapse; }
th, td { padding: 12px; text-align: left; border-bottom: 1px solid var(--border); }
th { font-weight: 600; color: var(--muted); }
.empty { padding: 20px; text-align: center; color: var(--muted); }
</style>
