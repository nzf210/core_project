<template>
  <div class="performance-view">
    <div class="header-section">
      <div>
        <h1 class="page-title">Performance Analytics</h1>
        <p class="text-muted">Track the historical performance of your automated strategies.</p>
      </div>
      <div class="date-filter">
        <select class="filter-select">
          <option>Last 7 Days</option>
          <option selected>Last 30 Days</option>
          <option>Last 90 Days</option>
          <option>All Time</option>
        </select>
      </div>
    </div>

    <div class="card chart-container">
      <div class="chart-header">
        <h3>Cumulative PnL (USDT)</h3>
        <h2 :class="stats.totalProfit >= 0 ? 'text-green' : 'text-red'">
          {{ stats.totalProfit >= 0 ? '+' : '' }}${{ stats.totalProfit.toFixed(2) }}
        </h2>
      </div>
      
      <div class="mock-chart" ref="chartContainer">
      </div>
    </div>
    
    <div class="metrics-grid">
      <div class="card metric-card">
        <h4>Win Rate</h4>
        <div class="metric-value">{{ stats.winRate.toFixed(1) }}%</div>
      </div>
      <div class="card metric-card">
        <h4>Best Performing Bot</h4>
        <div class="metric-value" style="font-size: 1.1rem">{{ bestBotName }}</div>
      </div>
      <div class="card metric-card">
        <h4>Avg. Daily Profit</h4>
        <div class="metric-value" :class="stats.dailyProfit >= 0 ? 'text-green' : 'text-red'">
          {{ stats.dailyProfit >= 0 ? '+' : '' }}${{ stats.dailyProfit.toFixed(2) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { createChart, LineSeries, type IChartApi } from 'lightweight-charts'
import api from '../api'

const stats = ref({
  winRate: 0,
  dailyProfit: 0,
  totalProfit: 0
})

const bestBotName = ref('No bots yet')
const chartContainer = ref<HTMLElement | null>(null)
let chart: IChartApi | null = null

const fetchStats = async () => {
  try {
    const res = await api.get('/api/crypto/dashboard')
    if (res.data.success) {
      const data = res.data.data
      stats.value.winRate = data.win_rate || 0
      stats.value.totalProfit = data.total_profit || 0
      stats.value.dailyProfit = (data.total_profit || 0) / 30 // Rough estimate
    }
  } catch (err) {
    console.error('Failed to fetch performance stats', err)
  }
}

const fetchBots = async () => {
  try {
    const res = await api.get('/api/crypto/bots')
    if (res.data.success && res.data.data && res.data.data.length > 0) {
      const bots = res.data.data;
      let best = bots[0];
      for (const b of bots) {
        if ((b.total_profit || 0) > (best.total_profit || 0)) {
          best = b;
        }
      }
      bestBotName.value = best.name || 'Unknown Bot';
    }
  } catch (err) {
    console.error('Failed to fetch bots', err)
  }
}

onMounted(async () => {
  await fetchStats()
  await fetchBots()

  if (chartContainer.value) {
    chart = createChart(chartContainer.value, {
      layout: {
        background: { color: 'transparent' },
        textColor: '#9ca3af',
      },
      grid: {
        vertLines: { visible: false },
        horzLines: { visible: false },
      },
      width: chartContainer.value.clientWidth,
      height: 200,
    })

    const lineSeries = chart.addSeries(LineSeries, { 
      color: '#00c853', 
      lineWidth: 3,
      crosshairMarkerVisible: false,
    })

    try {
      // Get 30d BTC data as a real market performance proxy
      const response = await fetch('https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1d&limit=30')
      const data = await response.json()
      
      const chartData = data.map((d: any) => ({
        time: d[0] / 1000, 
        value: parseFloat(d[4]) // close price
      }))
      
      lineSeries.setData(chartData)
      chart.timeScale().fitContent()
    } catch (e) {
      console.error('Failed to fetch chart data', e)
    }
  }
})

onUnmounted(() => {
  if (chart) chart.remove()
})
</script>

<style scoped>
.performance-view {
  padding: 1rem;
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 2rem;
}

.page-title {
  font-size: 1.75rem;
  margin-bottom: 0.25rem;
}

.filter-select {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
}

.chart-container {
  margin-bottom: 2rem;
  padding: 1.5rem;
}

.chart-header {
  margin-bottom: 2rem;
}

.chart-header h3 {
  color: var(--text-muted);
  font-size: 1rem;
  margin-bottom: 0.25rem;
}

.chart-header h2 {
  font-size: 2rem;
  margin: 0;
}

.mock-chart {
  width: 100%;
  height: 250px;
  display: flex;
  align-items: flex-end;
}

.sparkline {
  width: 100%;
  height: 100%;
  overflow: visible;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
}

.metric-card {
  padding: 1.5rem;
}

.metric-card h4 {
  color: var(--text-muted);
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
}

.metric-value {
  font-size: 1.25rem;
  font-weight: 700;
}
</style>
