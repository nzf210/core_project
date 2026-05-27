<template>
  <div class="dashboard">
    <div class="summary-cards">
      <div class="glass-card">
        <div class="label">Total Portfolio Balance</div>
        <div class="value">${{ (stats.totalPortfolio || 0).toLocaleString('en-US', {minimumFractionDigits: 2}) }}</div>
        <div class="change" style="color: var(--text-muted)">Real-time Value</div>
      </div>
      <div class="glass-card">
        <div class="label">Active Bots</div>
        <div class="value">{{ stats.activeBots || 0 }} <span style="font-size: 1rem; color: var(--text-muted)">/ 10</span></div>
        <div class="change" style="color: var(--text-muted)">Pro Tier Active</div>
      </div>
      <div class="glass-card">
        <div class="label">Total Profit (All Time)</div>
        <div :class="['value', (stats.totalProfit || 0) >= 0 ? 'text-gradient' : 'text-danger']">
          {{ (stats.totalProfit || 0) >= 0 ? '+' : '' }}${{ Math.abs(stats.totalProfit || 0).toLocaleString('en-US', {minimumFractionDigits: 2}) }}
        </div>
        <div :class="['change', (stats.winRate || 0) >= 50 ? 'badge-up' : 'badge-down']">Win Rate: {{ (stats.winRate || 0).toFixed(1) }}%</div>
      </div>
    </div>

    <div class="chart-section glass-card" style="margin-bottom: 3rem;">
      <div class="flex items-center justify-between" style="margin-bottom: 1rem;">
        <h2>BTC/USDT Real-time Chart (Technical Analysis)</h2>
        <div>
          <span class="badge badge-success">SMA 20</span>
          <span class="badge badge-danger" style="margin-left: 0.5rem">EMA 50</span>
        </div>
      </div>
      <div ref="chartContainer" class="lw-chart"></div>
    </div>

    <div class="bot-list">
      <div class="flex items-center justify-between" style="margin-bottom: 1.5rem">
        <h2>Running Bots</h2>
        <button class="btn btn-primary" @click="showModal = true">Create Bot</button>
      </div>
      <div class="table-container glass-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>Pair</th>
              <th>Strategy</th>
              <th>Status</th>
              <th>Uptime</th>
              <th>PnL</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="bot in bots" :key="bot.id">
              <td><strong>{{ bot.pair }}</strong></td>
              <td>{{ bot.bot_type }}</td>
              <td>
                <span class="status-badge" :class="bot.status === 'running' ? 'running' : 'paused'">
                  {{ bot.status }}
                </span>
              </td>
              <td>${{ (bot.total_invested || 0).toFixed(2) }}</td>
              <td :class="(bot.total_profit || 0) >= 0 ? 'badge-up' : 'badge-down'">
                {{ (bot.total_profit || 0) >= 0 ? '+' : '' }}${{ (bot.total_profit || 0).toFixed(2) }}
              </td>
              <td>
                <button 
                  class="action-btn" 
                  :class="bot.status === 'running' ? 'text-danger' : 'text-success'"
                  @click="toggleBotStatus(bot)"
                >
                  {{ bot.status === 'running' ? 'Stop' : 'Start' }}
                </button>
              </td>
            </tr>
            <tr v-if="bots.length === 0">
              <td colspan="6" style="text-align: center; color: var(--text-muted)">Belum ada bot.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Create Bot Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-content glass-card">
        <h3>Configure New Bot</h3>
        <div class="form-group">
          <label>Trading Pair</label>
          <select class="form-control" v-model="botConfig.pair">
            <option value="BTCUSDT">BTC/USDT</option>
            <option value="ETHUSDT">ETH/USDT</option>
            <option value="SOLUSDT">SOL/USDT</option>
          </select>
        </div>
        <div class="form-group">
          <label>Strategy</label>
          <select class="form-control" v-model="botConfig.strategy">
            <option value="dca">DCA (Daily)</option>
            <option value="grid">Grid Trading</option>
            <option value="signal">AI Oracle Signal</option>
          </select>
        </div>
        <div class="form-group">
          <label>Investment Amount (USDT)</label>
          <input type="number" class="form-control" v-model="botConfig.amount" placeholder="e.g. 1000">
        </div>
        <div class="form-group">
          <label>Trading Mode</label>
          <div style="display: flex; gap: 1rem;">
            <label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer; color: var(--text-primary);">
              <input type="radio" v-model="botConfig.isPaperTrading" :value="true" /> Paper Trading
            </label>
            <label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer; color: var(--text-primary);">
              <input type="radio" v-model="botConfig.isPaperTrading" :value="false" /> Real Trading
            </label>
          </div>
        </div>
        <div class="form-group" v-if="!botConfig.isPaperTrading">
          <label>API Key</label>
          <select v-model="botConfig.apiKeyId" class="form-control" required>
            <option value="" disabled>Select API Key</option>
            <option v-for="key in apiKeys" :key="key.id" :value="key.id">
              {{ key.exchange }} - {{ key.label }}
            </option>
          </select>
          <p v-if="apiKeys.length === 0" style="color: var(--warning); font-size: 0.8rem; margin-top: 0.5rem;">
            Anda belum memiliki API Key. Silakan tambahkan di menu API Keys.
          </p>
        </div>
        <div class="modal-actions">
          <button class="btn" style="background: transparent; color: white; border: 1px solid var(--border-color);" @click="showModal = false">Cancel</button>
          <button class="btn btn-primary" @click="saveBot" :disabled="isLoading">
            {{ isLoading ? 'Loading...' : 'Start Bot' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { createChart, CandlestickSeries, type IChartApi } from 'lightweight-charts'
import { toast } from 'vue3-toastify'
import api from '../api'

const showModal = ref(false)
const isLoading = ref(false)
const botConfig = ref({
  pair: 'BTCUSDT',
  strategy: 'dca',
  amount: 1000,
  isPaperTrading: true,
  apiKeyId: ''
})

const bots = ref<any[]>([])
const apiKeys = ref<any[]>([])

const fetchApiKeys = async () => {
  try {
    const res = await api.get('/api/crypto/api-keys')
    if (res.data.success) {
      apiKeys.value = res.data.data || []
    }
  } catch (err) {
    console.error('Failed to fetch API keys', err)
  }
}

const fetchBots = async () => {
  try {
    const res = await api.get('/api/crypto/bots')
    if (res.data.success) {
      bots.value = res.data.data || []
    }
  } catch (err) {
    toast.error('Failed to fetch bots')
  }
}

const chartContainer = ref<HTMLElement | null>(null)
let chart: IChartApi | null = null

const stats = ref({
  totalPortfolio: 0,
  totalProfit: 0,
  activeBots: 0,
  winRate: 0
})

const fetchDashboardStats = async () => {
  try {
    const res = await api.get('/api/crypto/dashboard')
    if (res.data.success) {
      const data = res.data.data
      stats.value = {
        totalPortfolio: data.total_portfolio_value || 0,
        totalProfit: data.total_profit || 0,
        activeBots: data.active_bots || 0,
        winRate: data.win_rate || 0
      }
    }
  } catch (err) {
    console.error('Failed to fetch dashboard stats', err)
  }
}

onMounted(async () => {
  await fetchDashboardStats()
  await fetchBots()
  await fetchApiKeys()

  if (chartContainer.value) {
    chart = createChart(chartContainer.value, {
      layout: {
        background: { color: 'transparent' },
        textColor: '#9ca3af',
      },
      grid: {
        vertLines: { color: '#2a2e39' },
        horzLines: { color: '#2a2e39' },
      },
      width: chartContainer.value.clientWidth,
    })

    // Realtime Binance Data
    const candlestickSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderVisible: false,
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444'
    })

    try {
      const response = await fetch('https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1d&limit=100')
      const data = await response.json()
      
      const chartData = data.map((d: any) => ({
        time: d[0] / 1000, // Unix timestamp in seconds
        open: parseFloat(d[1]),
        high: parseFloat(d[2]),
        low: parseFloat(d[3]),
        close: parseFloat(d[4])
      }))
      
      candlestickSeries.setData(chartData)
    } catch (e) {
      console.error('Failed to fetch chart data', e)
    }
  }
})

onUnmounted(() => {
  if (chart) chart.remove()
})

const toggleBotStatus = async (bot: any) => {
  try {
    const endpoint = `/api/crypto/bots/${bot.id}/status`
    const newStatus = bot.status === 'running' ? 'stopped' : 'running'
    const res = await api.put(endpoint, { status: newStatus })
    if (res.data.success) {
      bot.status = newStatus
      toast.success(`Bot berhasil di${bot.status === 'running' ? 'jalankan' : 'berhentikan'}`)
    }
  } catch (err) {
    toast.error('Gagal mengubah status bot')
  }
}

const saveBot = async () => {
  if (!botConfig.value.isPaperTrading && !botConfig.value.apiKeyId) {
    toast.error('Pilih API Key untuk Real Trading')
    return
  }

  isLoading.value = true
  try {
    const res = await api.post('/api/crypto/bots', {
      name: `New ${botConfig.value.strategy} Bot`,
      bot_type: botConfig.value.strategy,
      pair: botConfig.value.pair,
      is_paper_trading: botConfig.value.isPaperTrading,
      api_key_id: botConfig.value.isPaperTrading ? undefined : botConfig.value.apiKeyId,
      dca_interval: botConfig.value.strategy === 'dca' ? 'daily' : undefined,
      dca_amount: Number(botConfig.value.amount),
      grid_count: botConfig.value.strategy === 'grid' ? 10 : undefined,
      grid_lower_price: botConfig.value.strategy === 'grid' ? 30000 : undefined,
      grid_upper_price: botConfig.value.strategy === 'grid' ? 40000 : undefined,
      grid_investment: botConfig.value.strategy === 'grid' ? Number(botConfig.value.amount) : undefined,
    })
    
    if (res.data.success) {
      showModal.value = false
      toast.success('Bot berhasil dibuat dan dijalankan!')
      await fetchBots()
    }
  } catch (err: any) {
    toast.error(err.response?.data?.message || 'Gagal membuat bot')
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.summary-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-bottom: 3rem;
}
.label {
  color: var(--text-secondary);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.5rem;
}
.value {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 0.25rem;
}
.change {
  font-size: 0.875rem;
  font-weight: 500;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}
.data-table th, .data-table td {
  padding: 1.25rem 1rem;
  border-bottom: 1px solid var(--border-color);
}
.data-table th {
  color: var(--text-secondary);
  font-weight: 500;
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.05em;
}
.data-table tbody tr:last-child td { border-bottom: none; }
.status-badge {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}
.status-badge.running { background: rgba(34, 197, 94, 0.2); color: var(--success); }
.status-badge.paused { background: rgba(245, 158, 11, 0.2); color: var(--warning); }
.action-btn {
  background: transparent;
  border: none;
  color: var(--accent-primary);
  font-family: inherit;
  font-weight: 600;
  cursor: pointer;
}
.action-btn:hover { text-decoration: underline; }
.text-danger { color: var(--danger) !important; }

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal-content {
  width: 100%;
  max-width: 450px;
  padding: 2rem;
}
.modal-content h3 {
  margin-bottom: 1.5rem;
}
.form-group {
  margin-bottom: 1.25rem;
}
.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}
.form-control {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  margin-top: 2rem;
}
</style>
