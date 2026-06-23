<template>
  <div class="dynamic-dashboard">
    <div class="db-header">
      <div>
        <h2>{{ businessName || 'Dashboard' }}</h2>
        <p class="db-subtitle">{{ typeLabel }} · {{ planLabel }} Plan</p>
      </div>
      <div class="db-actions">
        <span v-if="plan === 'lite'" class="quota-badge">{{ quotaUsed }}/{{ quotaLimit }} transaksi</span>
        <button v-if="plan === 'lite'" class="btn btn-upgrade" @click="showUpgrade = true">
          Upgrade →
        </button>
        <button class="btn btn-primary" @click="syncData" :disabled="loading">
          {{ loading ? 'Memuat...' : 'Sync Data' }}
        </button>
      </div>
    </div>

    <!-- WA Setup warning banner -->
    <div v-if="waWarning" class="glass-card" style="margin-bottom: 1.5rem; padding: 1.25rem; border-left: 4px solid #f59e0b; background: rgba(245, 158, 11, 0.1);">
      <div style="display: flex; justify-content: space-between; align-items: center;">
        <div>
          <strong style="color: #f59e0b;">⚠️ WhatsApp Tidak Terhubung</strong>
          <p style="margin: 0.25rem 0 0; font-size: 0.85rem;">Pelanggan tidak bisa chat dengan AI Anda. Silakan hubungkan WhatsApp.</p>
        </div>
        <button class="btn btn-secondary" @click="router.push('/wa-setup')">Setup WA</button>
      </div>
    </div>

    <div v-if="showUpgrade" class="upgrade-banner">
      <span>Upgrade ke Pro (Rp 450K/bln) untuk fitur advanced & AI unlimited.</span>
      <button class="btn btn-success" @click="upgradePlan('pro')">Upgrade Sekarang</button>
      <button class="btn-close" @click="showUpgrade = false">×</button>
    </div>

    <div class="widgets-grid">
      <div
        v-for="widget in activeWidgets"
        :key="widget.id"
        :class="['widget', 'glass-card', `widget-${widget.size}`]"
      >
        <div class="widget-header">
          <span class="widget-title">{{ widget.title }}</span>
        </div>

        <!-- Metric widget -->
        <div v-if="widget.type === 'metric'" class="widget-metric">
          <div class="metric-value text-accent-primary">{{ formatCurrency(widgetData[widget.id]?.value || 0) }}</div>
          <div v-if="widgetData[widget.id]?.change !== undefined" class="metric-change" :class="widgetData[widget.id]?.change >= 0 ? 'positive' : 'negative'">
            {{ widgetData[widget.id]?.change > 0 ? '+' : '' }}{{ widgetData[widget.id]?.change?.toFixed(1) }}% vs bulan lalu
          </div>
        </div>

        <!-- F060: Real Chart.js chart -->
        <div v-else-if="widget.type === 'chart'" class="widget-chart">
          <!-- Period switcher -->
          <div class="chart-header-row">
            <div class="period-switcher">
              <button v-for="p in periods" :key="p" :class="{ active: period === p }" @click="setPeriod(p)">
                {{ periodLabels[p] }}
              </button>
            </div>
          </div>
          <!-- Loading -->
          <div v-if="chartLoading" class="chart-loading">
            <div class="loading-spinner"></div>
          </div>
          <!-- Chart -->
          <div v-else-if="chartReady" class="chart-wrap">
            <Bar :data="chartData" :options="chartOptions" />
          </div>
          <!-- Empty -->
          <div v-else class="empty-state">Belum ada data penjualan</div>
        </div>

        <!-- F060: Real List widget (recent transactions, top products) -->
        <div v-else-if="widget.type === 'list'" class="widget-list">
          <!-- Recent transactions -->
          <div v-if="widget.id === 'recent_transactions'">
            <div v-if="recentTxLoading" class="list-loading">
              <span class="loading-dot" v-for="n in 5" :key="n"></span>
            </div>
            <div v-else-if="recentTransactions.length" class="list-items">
              <div v-for="tx in recentTransactions" :key="tx.id" class="list-item">
                <span class="item-label">{{ tx.description || 'Transaksi' }}</span>
                <span class="item-value">{{ formatCurrency(tx.amount_rupiah) }}</span>
              </div>
            </div>
            <div v-else class="empty-state">Belum ada transaksi</div>
          </div>
          <!-- Top products -->
          <div v-else-if="widget.id === 'top_products' || widget.id === 'best_selling' || widget.id === 'popular_items'">
            <div v-if="topProductsLoading" class="list-loading">
              <span class="loading-dot" v-for="n in 5" :key="n"></span>
            </div>
            <div v-else-if="topProducts.length" class="list-items">
              <div v-for="(prod, idx) in topProducts" :key="idx" class="list-item">
                <span class="item-label">{{ prod.name }}</span>
                <span class="item-value">{{ formatCurrency(prod.revenue_rupiah) }}</span>
              </div>
            </div>
            <div v-else class="empty-state">Belum ada data produk</div>
          </div>
          <!-- Generic list fallback -->
          <div v-else>
            <div v-if="widgetData[widget.id]?.items?.length" class="list-items">
              <div v-for="(item, idx) in widgetData[widget.id].items.slice(0, 5)" :key="idx" class="list-item">
                <span class="item-label">{{ item.label }}</span>
                <span class="item-value">{{ item.value }}</span>
              </div>
            </div>
            <div v-else class="empty-state">Belum ada data</div>
          </div>
        </div>

        <!-- Actions widget -->
        <div v-else-if="widget.type === 'actions'" class="widget-actions">
          <button @click="router.push('/pos')" class="action-btn">➕ Transaksi Baru</button>
          <button @click="router.push('/catalog')" class="action-btn">📦 Kelola Produk</button>
          <button @click="router.push('/journal')" class="action-btn">📊 Lihat Laporan</button>
        </div>

        <!-- Timeline widget -->
        <div v-else-if="widget.type === 'timeline'" class="widget-timeline">
          <div v-for="(_, idx) in 4" :key="idx" class="timeline-item">
            <div class="timeline-dot"></div>
            <div class="timeline-content">
              <span class="timeline-text">Pesanan #{{ 1000 + idx }}</span>
              <span class="timeline-time">{{ idx * 2 }} jam lalu</span>
            </div>
          </div>
        </div>

        <!-- Kanban / alert / fallback -->
        <div v-else class="widget-generic">
          <div class="empty-state">Widget {{ widget.type }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Bar } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js'
import { api, reportsApi } from '../api'
import type { TopProduct, RecentTransaction } from '../api'

ChartJS.register(CategoryScale, LinearScale, BarElement, Title, Tooltip, Legend)

const router = useRouter()

// F060: Chart state
const period = ref<'week' | 'month' | 'year'>('week')
const periods = ['week', 'month', 'year'] as const
const periodLabels: Record<string, string> = { week: '7H', month: '30H', year: '12B' }
const chartLoading = ref(false)
const chartReady = ref(false)
const chartData = ref({ labels: [] as string[], datasets: [] as any[] })
const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: any) => ' Rp ' + (ctx.raw / 1000).toFixed(0) + 'K',
      },
    },
  },
  scales: {
    x: {
      grid: { color: 'rgba(255,255,255,0.04)' },
      ticks: { color: '#64748b', font: { size: 10 } },
    },
    y: {
      grid: { color: 'rgba(255,255,255,0.04)' },
      ticks: {
        color: '#64748b',
        font: { size: 10 },
        callback: (v: any) => 'Rp ' + (v / 1000).toFixed(0) + 'K',
      },
    },
  },
}

// F060: Real list data
const topProducts = ref<TopProduct[]>([])
const recentTransactions = ref<RecentTransaction[]>([])
const topProductsLoading = ref(false)
const recentTxLoading = ref(false)

const businessName = ref('')
const businessType = ref('umum')
const plan = ref('lite')
const loading = ref(false)
const showUpgrade = ref(false)
const waWarning = ref(false)
const quotaUsed = ref(0)
const quotaLimit = ref(100)

const typeLabels: Record<string, string> = {
  umum: 'Umum',
  warung: 'Warung / Toko',
  laundry: 'Laundry',
  industri_kreatif: 'Industri Kreatif',
  toko_online: 'Toko Online',
  restoran: 'Restoran / F&B',
  jasa: 'Jasa / Service'
}

const planLabels: Record<string, string> = {
  lite: 'Lite',
  pro: 'Pro',
  ultimate: 'Ultimate'
}

const typeLabel = computed(() => typeLabels[businessType.value] || 'Umum')
const planLabel = computed(() => planLabels[plan.value] || 'Lite')

interface Widget {
  id: string
  type: string
  title: string
  module: string
  config: Record<string, any>
  position: number
  size: string
}

const widgetData = ref<Record<string, any>>({})

const widgetTemplates: Record<string, Widget[]> = {
  umum: [
    { id: 'income_summary', type: 'metric', title: 'Pendapatan Bulan Ini', module: 'reports', config: {}, position: 0, size: 'medium' },
    { id: 'expense_summary', type: 'metric', title: 'Pengeluaran Bulan Ini', module: 'reports', config: {}, position: 1, size: 'medium' },
    { id: 'profit_summary', type: 'metric', title: 'Laba Bersih', module: 'reports', config: {}, position: 2, size: 'medium' },
    { id: 'recent_transactions', type: 'list', title: 'Transaksi Terbaru', module: 'transactions', config: {}, position: 3, size: 'large' },
    { id: 'quick_actions', type: 'actions', title: 'Aksi Cepat', module: 'pos', config: {}, position: 4, size: 'small' }
  ],
  warung: [
    { id: 'daily_sales', type: 'chart', title: 'Penjualan Minggu Ini', module: 'transactions', config: {}, position: 0, size: 'large' },
    { id: 'best_selling', type: 'list', title: 'Item Terlaris', module: 'inventory', config: {}, position: 1, size: 'medium' },
    { id: 'stock_alert_widget', type: 'list', title: 'Stok Menipis', module: 'inventory', config: {}, position: 2, size: 'medium' },
    { id: 'income_today', type: 'metric', title: 'Pendapatan Hari Ini', module: 'reports', config: {}, position: 3, size: 'medium' },
    { id: 'profit_margin', type: 'metric', title: 'Margin Untung', module: 'reports', config: {}, position: 4, size: 'medium' }
  ],
  laundry: [
    { id: 'active_orders', type: 'list', title: 'Pesanan Aktif', module: 'order_tracking', config: {}, position: 0, size: 'large' },
    { id: 'income_today', type: 'metric', title: 'Pendapatan Hari Ini', module: 'reports', config: {}, position: 1, size: 'medium' },
    { id: 'order_status', type: 'timeline', title: 'Status Pesanan', module: 'order_tracking', config: {}, position: 2, size: 'medium' },
    { id: 'total_customers', type: 'metric', title: 'Total Pelanggan', module: 'customers', config: {}, position: 3, size: 'medium' },
    { id: 'quick_actions', type: 'actions', title: 'Aksi Cepat', module: 'pos', config: {}, position: 4, size: 'small' }
  ],
  industri_kreatif: [
    { id: 'active_projects', type: 'list', title: 'Proyek Aktif', module: 'project_tracking', config: {}, position: 0, size: 'large' },
    { id: 'income_summary', type: 'metric', title: 'Pendapatan Bulanan', module: 'reports', config: {}, position: 1, size: 'medium' },
    { id: 'project_margin', type: 'metric', title: 'Margin Proyek', module: 'reports', config: {}, position: 2, size: 'medium' },
    { id: 'invoice_status', type: 'list', title: 'Invoice Pending', module: 'invoice_generator', config: {}, position: 3, size: 'medium' },
    { id: 'quick_actions', type: 'actions', title: 'Aksi Cepat', module: 'pos', config: {}, position: 4, size: 'small' }
  ],
  toko_online: [
    { id: 'order_volume', type: 'chart', title: 'Volume Pesanan', module: 'transactions', config: {}, position: 0, size: 'large' },
    { id: 'top_products', type: 'list', title: 'Produk Teratas', module: 'inventory', config: {}, position: 1, size: 'medium' },
    { id: 'income_today', type: 'metric', title: 'Pendapatan Hari Ini', module: 'reports', config: {}, position: 2, size: 'medium' },
    { id: 'pending_shipments', type: 'list', title: 'Pengiriman Pending', module: 'shipment_tracking', config: {}, position: 3, size: 'medium' },
    { id: 'quick_actions', type: 'actions', title: 'Aksi Cepat', module: 'pos', config: {}, position: 4, size: 'small' }
  ],
  restoran: [
    { id: 'daily_sales', type: 'chart', title: 'Penjualan Hari Ini', module: 'transactions', config: {}, position: 0, size: 'large' },
    { id: 'income_today', type: 'metric', title: 'Pendapatan Hari Ini', module: 'reports', config: {}, position: 1, size: 'medium' },
    { id: 'popular_items', type: 'list', title: 'Menu Terpopuler', module: 'menu_management', config: {}, position: 2, size: 'medium' },
    { id: 'cost_ratio', type: 'metric', title: 'Rasio Biaya', module: 'reports', config: {}, position: 3, size: 'medium' },
    { id: 'quick_actions', type: 'actions', title: 'Aksi Cepat', module: 'pos', config: {}, position: 4, size: 'small' }
  ],
  jasa: [
    { id: 'appointments_today', type: 'list', title: 'Janji Hari Ini', module: 'appointment_scheduling', config: {}, position: 0, size: 'large' },
    { id: 'income_today', type: 'metric', title: 'Pendapatan Hari Ini', module: 'reports', config: {}, position: 1, size: 'medium' },
    { id: 'top_services', type: 'list', title: 'Layanan Populer', module: 'service_catalog', config: {}, position: 2, size: 'medium' },
    { id: 'total_customers', type: 'metric', title: 'Total Pelanggan', module: 'customers', config: {}, position: 3, size: 'medium' },
    { id: 'quick_actions', type: 'actions', title: 'Aksi Cepat', module: 'pos', config: {}, position: 4, size: 'small' }
  ]
}

const activeWidgets = computed(() => {
  return widgetTemplates[businessType.value] || widgetTemplates['umum']
})

const formatCurrency = (val: number) => {
  return 'Rp ' + Math.abs(val).toLocaleString('id-ID')
}

const syncData = async () => {
  loading.value = true
  try {
    const today = new Date().toISOString().split('T')[0]
    const lastMonth = new Date(new Date().setDate(new Date().getDate() - 30)).toISOString().split('T')[0]
    const twoMonthsAgo = new Date(new Date().setDate(new Date().getDate() - 60)).toISOString().split('T')[0]

    const [isData, isPrevData] = await Promise.all([
      api.get(`/api/umkm/reports/income-statement?from=${lastMonth}&to=${today}`),
      api.get(`/api/umkm/reports/income-statement?from=${twoMonthsAgo}&to=${lastMonth}`),
      api.get(`/api/umkm/reports/balance-sheet?date=${today}`)
    ])

    if (isData.success && isData.data) {
      const curr = isData.data
      const prev = isPrevData.success ? isPrevData.data : { revenue: 0, expense: 0 }

      const calcChange = (c: number, p: number) => p > 0 ? ((c - p) / p) * 100 : (c > 0 ? 100 : 0)

      widgetData.value = {
        income_summary: { value: curr.revenue || 0, change: calcChange(curr.revenue || 0, prev.revenue || 0) },
        income_today: { value: Math.round((curr.revenue || 0) / 30), change: calcChange(curr.revenue || 0, prev.revenue || 0) },
        expense_summary: { value: curr.expense || 0, change: calcChange(curr.expense || 0, prev.expense || 0) },
        profit_summary: { value: (curr.revenue || 0) - (curr.expense || 0), change: calcChange((curr.revenue || 0) - (curr.expense || 0), (prev.revenue || 0) - (prev.expense || 0)) },
        profit_margin: { value: curr.revenue > 0 ? Math.round(((curr.revenue - curr.expense) / curr.revenue) * 100) : 0, change: 0 },
        cost_ratio: { value: curr.revenue > 0 ? Math.round((curr.expense / curr.revenue) * 100) : 0, change: 0 },
        total_customers: { value: 0, change: 0 },
        quick_actions: {}
      }
    }
  } catch (e) {
    console.error('Sync data gagal:', e)
  } finally {
    loading.value = false
  }

  // F060: Fetch sales chart + top products + recent transactions
  fetchSalesChart()
  fetchTopProducts()
  fetchRecentTransactions()

  // Check WA Setup Status
  try {
    const waRes = await api.getWASetup()
    if (waRes.success && waRes.data) {
      const waProvider = waRes.data.wa_provider_preference
      const wmConnected = waRes.data.whatsmeow?.connected
      const cloudActive = waRes.data.cloud_api?.active

      if (waProvider === 'whatsmeow' && !wmConnected) {
        waWarning.value = true
      } else if (waProvider === 'cloud_api' && !cloudActive) {
        waWarning.value = true
      } else if (waProvider === 'auto' && !wmConnected && !cloudActive) {
        waWarning.value = true
      } else {
        waWarning.value = false
      }
    }
  } catch (e) {
    console.warn('Failed to check WA status', e)
  }
}

const upgradePlan = (targetPlan: string) => {
  plan.value = targetPlan
  showUpgrade.value = false
  localStorage.setItem('plan', targetPlan)
}

// F060: Sales Chart
const fetchSalesChart = async () => {
  chartLoading.value = true
  try {
    const res = await reportsApi.getSalesChart(period.value)
    if (res.success && res.data) {
      const d = res.data
      chartData.value = {
        labels: d.labels,
        datasets: [
          {
            label: 'Pendapatan',
            data: d.revenue,
            backgroundColor: 'rgba(59, 130, 246, 0.7)',
            borderRadius: 4,
          },
          {
            label: 'Pengeluaran',
            data: d.expense,
            backgroundColor: 'rgba(239, 68, 68, 0.6)',
            borderRadius: 4,
          },
        ],
      }
      chartReady.value = true
    } else {
      chartReady.value = false
    }
  } catch (e) {
    console.error('Sales chart error:', e)
    chartReady.value = false
  } finally {
    chartLoading.value = false
  }
}

const setPeriod = (p: 'week' | 'month' | 'year') => {
  period.value = p
  fetchSalesChart()
}

// F060: Top Products
const fetchTopProducts = async () => {
  topProductsLoading.value = true
  try {
    const res = await reportsApi.getTopProducts(5)
    if (res.success && Array.isArray(res.data)) {
      topProducts.value = res.data
    }
  } catch (e) {
    console.warn('Top products error:', e)
  } finally {
    topProductsLoading.value = false
  }
}

// F060: Recent Transactions
const fetchRecentTransactions = async () => {
  recentTxLoading.value = true
  try {
    const res = await reportsApi.getRecentTransactions(5)
    if (res.success && Array.isArray(res.data)) {
      recentTransactions.value = res.data
    }
  } catch (e) {
    console.warn('Recent transactions error:', e)
  } finally {
    recentTxLoading.value = false
  }
}

onMounted(() => {
  businessType.value = localStorage.getItem('business_type') || 'umum'
  businessName.value = localStorage.getItem('business_name') || ''
  plan.value = localStorage.getItem('plan') || 'lite'
  quotaLimit.value = plan.value === 'lite' ? 100 : plan.value === 'pro' ? 1000 : 10000
  syncData()
})
</script>

<style scoped>
.dynamic-dashboard {
  min-height: calc(100vh - 80px);
}

.db-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
  flex-wrap: wrap;
  gap: 1rem;
}

.db-header h2 {
  font-size: 1.75rem;
  color: #f1f5f9;
  margin-bottom: 0.25rem;
}

.db-subtitle {
  color: #64748b;
  font-size: 0.875rem;
}

.db-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.quota-badge {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
}

.btn {
  padding: 0.5rem 1.25rem;
  border: none;
  border-radius: 8px;
  font-size: 0.875rem;
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

.btn-upgrade {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #fff;
}

.btn-upgrade:hover {
  transform: translateY(-1px);
}

.btn-success {
  background: #22c55e;
  color: #fff;
}

.upgrade-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.15), rgba(217, 119, 6, 0.1));
  border: 1px solid rgba(245, 158, 11, 0.3);
  padding: 0.75rem 1.5rem;
  border-radius: 12px;
  margin-bottom: 2rem;
  color: #fbbf24;
  font-size: 0.9rem;
  position: relative;
}

.btn-close {
  position: absolute;
  top: 8px;
  right: 12px;
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 1.2rem;
  cursor: pointer;
}

.widgets-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.25rem;
}

.widget-small { grid-column: span 1; }
.widget-medium { grid-column: span 1; }
.widget-large { grid-column: span 2; }

@media (max-width: 1200px) {
  .widgets-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .widgets-grid {
    grid-template-columns: 1fr;
  }
  .widget-large { grid-column: span 1; }
}

.widget {
  padding: 1.25rem;
  min-height: 120px;
}

.widget-header {
  margin-bottom: 1rem;
}

.widget-title {
  color: #94a3b8;
  font-size: 0.875rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.widget-metric .metric-value {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
}

.metric-change {
  font-size: 0.85rem;
  font-weight: 500;
}

.metric-change.positive { color: #22c55e; }
.metric-change.negative { color: #ef4444; }

.widget-chart {
  height: 200px;
  display: flex;
  flex-direction: column;
}

.chart-header-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-bottom: 0.5rem;
  flex-shrink: 0;
}

.period-switcher {
  display: flex;
  gap: 0.25rem;
  background: rgba(255,255,255,0.04);
  border-radius: 8px;
  padding: 3px;
}

.period-switcher button {
  background: none;
  border: none;
  color: #64748b;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 0.6rem;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
}

.period-switcher button.active {
  background: #3b82f6;
  color: #fff;
}

.period-switcher button:not(.active):hover {
  color: #e2e8f0;
  background: rgba(255,255,255,0.06);
}

.chart-wrap {
  flex: 1;
  min-height: 0;
  position: relative;
}

.chart-loading,
.list-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 6px;
}

.loading-spinner {
  width: 28px;
  height: 28px;
  border: 3px solid rgba(59,130,246,0.15);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-dot {
  width: 8px;
  height: 8px;
  background: #3b82f6;
  border-radius: 50%;
  opacity: 0.4;
  animation: pulse-dot 1s ease infinite;
}

.loading-dot:nth-child(2) { animation-delay: 0.15s; }
.loading-dot:nth-child(3) { animation-delay: 0.3s; }
.loading-dot:nth-child(4) { animation-delay: 0.45s; }
.loading-dot:nth-child(5) { animation-delay: 0.6s; }

.chart-placeholder {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 140px;
  padding-bottom: 4px;
}

.chart-bar {
  flex: 1;
  background: linear-gradient(to top, #3b82f6, #60a5fa);
  border-radius: 4px 4px 0 0;
  opacity: 0.8;
  min-height: 10px;
  transition: height 0.5s ease;
}

.chart-labels {
  display: flex;
  justify-content: space-between;
  color: #64748b;
  font-size: 0.7rem;
  margin-top: 4px;
}

.widget-list .list-items {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.list-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 6px;
}

.item-label { color: #e2e8f0; font-size: 0.9rem; }
.item-value { color: #3b82f6; font-weight: 600; font-size: 0.9rem; }

.widget-actions {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.action-btn {
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  padding: 0.6rem 1rem;
  border-radius: 8px;
  text-align: left;
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.2s;
  font-family: inherit;
}

.action-btn:hover {
  background: rgba(59, 130, 246, 0.2);
}

.widget-timeline {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.timeline-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.timeline-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #3b82f6;
  flex-shrink: 0;
}

.timeline-content {
  display: flex;
  justify-content: space-between;
  flex: 1;
  gap: 0.5rem;
}

.timeline-text { color: #e2e8f0; font-size: 0.85rem; }
.timeline-time { color: #64748b; font-size: 0.75rem; }

.empty-state {
  color: #475569;
  text-align: center;
  padding: 2rem;
  font-size: 0.9rem;
}
</style>
