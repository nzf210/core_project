<template>
  <div class="bot-management">
    <div class="header-section">
      <div>
        <h1 class="page-title">Automated Trading Bots</h1>
        <p class="text-muted">Create and manage your 24/7 algorithmic trading strategies.</p>
      </div>
      <button class="btn btn-primary create-btn" @click="showCreateModal = true">
        <span class="icon">+</span> Create New Bot
      </button>
    </div>

    <!-- Active Bots Summary -->
    <div class="stats-grid">
      <div class="card stat-card">
        <div class="stat-label">Active Bots</div>
        <div class="stat-value">{{ activeBotsCount }} / {{ bots.length }}</div>
      </div>
      <div class="stat-card card">
        <div class="stat-label">Total Invested</div>
        <div class="stat-value">${{ totalInvested.toFixed(2) }}</div>
      </div>
      <div class="stat-card card">
        <div class="stat-label">Total Profit (All Time)</div>
        <div :class="['stat-value', totalProfit >= 0 ? 'text-green' : 'text-red']">
          {{ totalProfit >= 0 ? '+' : '' }}${{ totalProfit.toFixed(2) }}
        </div>
      </div>
      <div class="stat-card card">
        <div class="stat-label">24h Bot Trades</div>
        <div class="stat-value">{{ totalTrades }}</div>
      </div>
    </div>

    <!-- Active Bots List -->
    <div class="bots-container">
      <h2 class="section-title">Your Bots</h2>
      
      <div v-if="bots.length === 0" class="empty-state card">
        <p class="text-muted">You don't have any bots running yet.</p>
        <button class="btn btn-primary mt-4" @click="showCreateModal = true">Create Your First Bot</button>
      </div>
      
      <div v-else class="bots-grid">
        <div v-for="bot in bots" :key="bot.id" class="card bot-card">
          <div class="bot-header">
            <div class="bot-title-group">
              <span :class="['status-dot', bot.isActive ? 'active' : 'paused']"></span>
              <h3 class="bot-name">{{ bot.name }}</h3>
            </div>
            <span class="bot-type">{{ bot.type }}</span>
          </div>
          
          <div class="bot-details">
            <div class="detail-row">
              <span class="text-muted">Pair</span>
              <strong>{{ bot.pair }}</strong>
            </div>
            <div class="detail-row" v-if="bot.interval">
              <span class="text-muted">Interval</span>
              <strong>{{ bot.interval }}</strong>
            </div>
            <div class="detail-row" v-if="bot.grids">
              <span class="text-muted">Grids</span>
              <strong>{{ bot.grids }}</strong>
            </div>
            <div class="detail-row" v-if="bot.amountPerOrder">
              <span class="text-muted">Amount/Order</span>
              <strong>${{ bot.amountPerOrder.toFixed(2) }}</strong>
            </div>
            <div class="detail-row">
              <span class="text-muted">Total Invested</span>
              <strong>${{ bot.invested.toFixed(2) }}</strong>
            </div>
            <div class="detail-row">
              <span class="text-muted">Profit</span>
              <strong :class="bot.profit >= 0 ? 'text-green' : 'text-red'">
                {{ bot.profit >= 0 ? '+' : '' }}${{ bot.profit.toFixed(2) }}
              </strong>
            </div>
          </div>
          
          <div class="bot-actions mt-auto">
            <button v-if="bot.isActive" class="btn btn-outline" @click="toggleBotStatus(bot.id)">Pause</button>
            <button v-else class="btn btn-primary w-full" @click="toggleBotStatus(bot.id)">Resume Bot</button>
            <button v-if="bot.isActive" class="btn btn-outline" @click="showBotDetails(bot)">Details</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Bot Modal -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="card modal-content">
        <div class="modal-header">
          <h2>Create DCA Bot</h2>
          <button class="close-btn" @click="showCreateModal = false">✕</button>
        </div>
        
        <form @submit.prevent="handleCreateBot" class="modal-body">
          <div class="form-group">
            <label>Bot Name</label>
            <input type="text" v-model="newBotForm.name" placeholder="e.g. BTC Accumulator" class="w-full" required />
          </div>
          
          <div class="form-group">
            <label>Trading Pair</label>
            <select v-model="newBotForm.pair" class="w-full" required>
              <option value="BTCUSDT">BTC/USDT</option>
              <option value="ETHUSDT">ETH/USDT</option>
              <option value="SOLUSDT">SOL/USDT</option>
              <option value="BNBUSDT">BNB/USDT</option>
            </select>
          </div>
          
          <div class="form-group">
            <label>Investment Amount (Per Order)</label>
            <div class="input-with-suffix">
              <input type="number" step="0.01" v-model.number="newBotForm.amount" placeholder="10.00" class="w-full" required />
              <span class="suffix">USDT</span>
            </div>
          </div>
          
          <div class="form-group">
            <label>Trading Mode</label>
            <div style="display: flex; gap: 1rem;">
              <label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer;">
                <input type="radio" v-model="newBotForm.isPaperTrading" :value="true" /> Paper Trading
              </label>
              <label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer;">
                <input type="radio" v-model="newBotForm.isPaperTrading" :value="false" /> Real Trading
              </label>
            </div>
          </div>
          
          <div class="form-group" v-if="!newBotForm.isPaperTrading">
            <label>API Key</label>
            <select v-model="newBotForm.apiKeyId" class="w-full" required>
              <option value="" disabled>Select API Key</option>
              <option v-for="key in apiKeys" :key="key.id" :value="key.id">
                {{ key.exchange }} - {{ key.label }}
              </option>
            </select>
            <p v-if="apiKeys.length === 0" style="color: var(--warning); font-size: 0.8rem; margin-top: 0.5rem;">
              Anda belum memiliki API Key. Silakan tambahkan di menu API Keys.
            </p>
          </div>

          <div class="form-group">
            <label>Frequency</label>
            <select v-model="newBotForm.interval" class="w-full" required>
              <option value="hourly">Every 1 Hour</option>
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
            </select>
          </div>
          
          <button type="submit" class="btn btn-primary w-full mt-4">Start Bot (24/7)</button>
        </form>
      </div>
    </div>

    <!-- Bot Details Modal -->
    <div v-if="selectedBot" class="modal-overlay" @click.self="selectedBot = null">
      <div class="card modal-content" style="max-width: 600px;">
        <div class="modal-header">
          <h2>{{ selectedBot.name }} Details</h2>
          <button class="close-btn" @click="selectedBot = null">✕</button>
        </div>
        <div class="modal-body" style="max-height: 400px; overflow-y: auto;">
          <div v-if="botOrders.length === 0" style="text-align: center; padding: 2rem;" class="text-muted">
            No orders found for this bot.
          </div>
          <table v-else style="width: 100%; text-align: left; border-collapse: collapse;">
            <thead>
              <tr>
                <th style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">Side</th>
                <th style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">Price</th>
                <th style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">Amount</th>
                <th style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">Status</th>
                <th style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">Time</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="order in botOrders" :key="order.id">
                <td style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);" :style="{ color: order.side === 'buy' ? 'var(--trade-green)' : 'var(--trade-red)' }">
                  <strong>{{ order.side.toUpperCase() }}</strong>
                </td>
                <td style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">${{ order.price.toFixed(2) }}</td>
                <td style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">{{ order.quantity }}</td>
                <td style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">{{ order.status }}</td>
                <td style="padding: 0.5rem; border-bottom: 1px solid var(--border-color);">{{ new Date(order.created_at).toLocaleString() }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { toast } from 'vue3-toastify'
import api from '../api'

interface Bot {
  id: string;
  name: string;
  type: string;
  pair: string;
  interval?: string;
  grids?: number;
  amountPerOrder?: number;
  invested: number;
  profit: number;
  totalTrades: number;
  isActive: boolean;
  bot_type?: string;
  status?: string;
  dca_interval?: string;
  dca_amount_per_order?: number;
  total_invested?: number;
  total_profit?: number;
}

const showCreateModal = ref(false)
const isLoading = ref(false)
const bots = ref<Bot[]>([])
const apiKeys = ref<any[]>([])

const selectedBot = ref<Bot | null>(null)
const botOrders = ref<any[]>([])

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
    if (res.data.success && res.data.data) {
      // Map API response to frontend Bot interface
      bots.value = res.data.data.map((b: any) => ({
        id: b.id,
        name: b.name,
        type: b.bot_type === 'dca' ? 'DCA Bot' : (b.bot_type === 'grid' ? 'Grid Bot' : 'Signal Bot'),
        pair: b.pair,
        interval: b.dca_interval,
        amountPerOrder: b.dca_amount_per_order,
        grids: b.grid_count,
        invested: b.total_invested || 0,
        profit: b.total_profit || 0,
        totalTrades: b.total_trades || 0,
        isActive: b.status === 'running'
      }))
    }
  } catch (err) {
    console.error('Failed to fetch bots', err)
  }
}

onMounted(() => {
  fetchBots()
  fetchApiKeys()
})

// Form state
const newBotForm = ref({
  name: '',
  pair: 'BTCUSDT',
  amount: 10,
  interval: 'daily',
  isPaperTrading: true,
  apiKeyId: ''
})

// Computed stats
const activeBotsCount = computed(() => bots.value.filter(b => b.isActive).length)
const totalInvested = computed(() => bots.value.reduce((sum, bot) => sum + (bot.invested || 0), 0))
const totalProfit = computed(() => bots.value.reduce((sum, bot) => sum + (bot.profit || 0), 0))
const totalTrades = computed(() => bots.value.reduce((sum, bot) => sum + (bot.totalTrades || 0), 0))

const toggleBotStatus = async (id: string) => {
  const bot = bots.value.find(b => b.id === id)
  if (bot) {
    try {
      const endpoint = `/api/crypto/bots/${id}/status`
      const newStatus = bot.isActive ? 'stopped' : 'running'
      const res = await api.put(endpoint, { status: newStatus })
      if (res.data.success) {
        bot.isActive = !bot.isActive
        toast.success(`Bot berhasil di${bot.isActive ? 'jalankan' : 'pause'}`)
      }
    } catch (err) {
      toast.error('Gagal mengubah status bot')
    }
  }
}

const showBotDetails = async (bot: Bot) => {
  selectedBot.value = bot
  botOrders.value = []
  try {
    const res = await api.get(`/api/crypto/orders?bot_id=${bot.id}`)
    if (res.data.success) {
      botOrders.value = res.data.data
    }
  } catch (err) {
    console.error('Failed to fetch bot orders', err)
    toast.error('Gagal mengambil data orders bot')
  }
}

const handleCreateBot = async () => {
  if (!newBotForm.value.isPaperTrading && !newBotForm.value.apiKeyId) {
    toast.error('Pilih API Key untuk Real Trading')
    return
  }
  
  isLoading.value = true
  try {
    const res = await api.post('/api/crypto/bots', {
      name: newBotForm.value.name,
      bot_type: 'dca',
      pair: newBotForm.value.pair,
      is_paper_trading: newBotForm.value.isPaperTrading,
      api_key_id: newBotForm.value.isPaperTrading ? undefined : newBotForm.value.apiKeyId,
      dca_interval: newBotForm.value.interval,
      dca_amount: newBotForm.value.amount
    })
    
    if (res.data.success) {
      showCreateModal.value = false
      // Reset form
      newBotForm.value = {
        name: '',
        pair: 'BTCUSDT',
        amount: 10,
        interval: 'daily',
        isPaperTrading: true,
        apiKeyId: ''
      }
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
.bot-management {
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

.create-btn {
  padding: 0.75rem 1.5rem;
  font-size: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1.5rem;
  margin-bottom: 3rem;
}

.stat-label {
  color: var(--text-muted);
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-primary);
}

.section-title {
  margin-bottom: 1.5rem;
  font-size: 1.25rem;
}

.bots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1.5rem;
}

.empty-state {
  text-align: center;
  padding: 3rem 1rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.bot-card {
  display: flex;
  flex-direction: column;
  min-height: 250px;
}

.bot-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.bot-title-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.status-dot.active { background-color: var(--trade-green); box-shadow: 0 0 8px var(--trade-green); }
.status-dot.paused { background-color: var(--text-muted); }

.bot-name {
  font-size: 1.1rem;
  margin: 0;
}

.bot-type {
  font-size: 0.75rem;
  padding: 0.25rem 0.5rem;
  background-color: var(--bg-tertiary);
  border-radius: var(--radius-sm);
  color: var(--accent-primary);
  font-weight: 600;
}

.bot-details {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  font-size: 0.9rem;
}

.bot-actions {
  display: flex;
  gap: 0.75rem;
  margin-top: auto;
}

.btn-outline {
  flex: 1;
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-primary);
}

.btn-outline:hover {
  background: var(--bg-tertiary);
  border-color: var(--text-muted);
}

.mt-auto { margin-top: auto; }
.w-full { width: 100%; }
.mt-4 { margin-top: 1rem; }

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  width: 100%;
  max-width: 450px;
  background-color: var(--bg-secondary);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.25rem;
}

.close-btn:hover { color: var(--text-primary); }

.form-group {
  margin-bottom: 1.25rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.input-with-suffix {
  position: relative;
  display: flex;
  align-items: center;
}

.input-with-suffix input {
  padding-right: 4rem;
}

.input-with-suffix .suffix {
  position: absolute;
  right: 1rem;
  color: var(--text-muted);
  font-weight: 600;
}
</style>
