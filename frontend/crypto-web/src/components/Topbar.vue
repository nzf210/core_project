<template>
  <header class="topbar">
    <div class="topbar-left">
      <button class="mobile-menu-btn" @click="$emit('toggleMobileMenu')">
        ☰
      </button>
      <div class="search-container">
        <span class="search-icon">🔍</span>
        <input type="text" placeholder="Search markets..." class="search-input" />
      </div>
    </div>
    
    <div class="topbar-right">
      <button class="btn btn-trade-buy quick-trade-btn" @click="showQuickTrade = true">
        <span class="btn-text">Quick Trade</span>
        <span class="btn-icon">⚡</span>
      </button>
      
      <div class="status-indicator desktop-only" title="System Operational">
        <span class="pulse-success"></span>
        <span class="status-text">Connected</span>
        <span class="latency-text">12ms</span>
      </div>
      
      <div class="notifications">
        <button class="icon-btn" @click="toggleNotifications">
          <span>🔔</span>
          <span v-if="unreadCount > 0" class="badge">{{ unreadCount }}</span>
        </button>
        <div v-if="showNotifications" class="dropdown-menu notifications-dropdown">
          <div class="dropdown-header">
            <h3>Notifications</h3>
          </div>
          <div class="dropdown-body">
            <div v-if="notifications.length === 0" class="empty-notif text-muted">
              No notifications yet.
            </div>
            <div v-for="n in notifications" :key="n.id" 
                 :class="['notif-item', { unread: !n.is_read }]"
                 @click="!n.is_read && markAsRead(n.id)">
              <div class="notif-content">
                <strong>{{ n.title }}</strong>
                <p>{{ n.message }}</p>
                <small class="text-muted">{{ new Date(n.created_at).toLocaleString() }}</small>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <div class="profile-dropdown desktop-only">
        <button class="icon-btn" @click="$emit('logout')" title="Logout">
          <span>🚪</span>
        </button>
      </div>
    </div>
    <!-- Quick Trade Modal -->
    <div v-if="showQuickTrade" class="modal-overlay" @click.self="showQuickTrade = false">
      <div class="card modal-content">
        <div class="modal-header">
          <h2>Quick Trade</h2>
          <button class="close-btn" @click="showQuickTrade = false">✕</button>
        </div>
        
        <form @submit.prevent="handleQuickTrade" class="modal-body">
          <div class="form-group">
            <label>Trading Pair</label>
            <select v-model="quickTradeForm.pair" class="w-full" required>
              <option value="BTCUSDT">BTC/USDT</option>
              <option value="ETHUSDT">ETH/USDT</option>
              <option value="SOLUSDT">SOL/USDT</option>
            </select>
          </div>
          
          <div class="form-group">
            <label>Side</label>
            <div style="display: flex; gap: 1rem;">
              <label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer; color: var(--trade-green);">
                <input type="radio" v-model="quickTradeForm.side" value="buy" /> Buy
              </label>
              <label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer; color: var(--trade-red);">
                <input type="radio" v-model="quickTradeForm.side" value="sell" /> Sell
              </label>
            </div>
          </div>
          
          <div class="form-group">
            <label>Amount (USDT)</label>
            <input type="number" step="0.01" v-model.number="quickTradeForm.amount" class="w-full" required />
          </div>
          
          <button type="submit" :class="['btn w-full mt-4', quickTradeForm.side === 'buy' ? 'btn-trade-buy' : 'btn-trade-sell']">
            Execute Market {{ quickTradeForm.side.toUpperCase() }}
          </button>
        </form>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { toast } from 'vue3-toastify'
import api from '../api'

defineEmits(['toggleMobileMenu', 'logout'])

const notifications = ref<any[]>([])
const showNotifications = ref(false)
const showQuickTrade = ref(false)

const quickTradeForm = ref({
  pair: 'BTCUSDT',
  side: 'buy',
  amount: 100
})

const unreadCount = computed(() => notifications.value.filter(n => !n.is_read).length)

const fetchNotifications = async () => {
  try {
    const res = await api.get('/api/crypto/notifications')
    if (res.data && res.data.success) {
      notifications.value = res.data.data
    }
  } catch (e) {
    console.error('Failed to fetch notifications', e)
  }
}

const markAsRead = async (id: string) => {
  try {
    await api.put(`/api/crypto/notifications/${id}/read`)
    const n = notifications.value.find(x => x.id === id)
    if (n) n.is_read = true
  } catch (e) {
    console.error('Failed to mark notification read', e)
  }
}

const toggleNotifications = () => {
  showNotifications.value = !showNotifications.value
}

const handleQuickTrade = async () => {
  try {
    const res = await api.post('/api/crypto/trade', quickTradeForm.value)
    if (res.data.success) {
      toast.success('Quick trade executed successfully')
      showQuickTrade.value = false
      fetchNotifications()
    }
  } catch (e: any) {
    toast.error(e.response?.data?.message || 'Quick trade failed')
  }
}

let notifInterval: number | undefined
onMounted(() => {
  fetchNotifications()
  notifInterval = setInterval(fetchNotifications, 15000) as any
})

onUnmounted(() => {
  if (notifInterval) clearInterval(notifInterval)
})
</script>

<style scoped>
.topbar {
  height: var(--topbar-height);
  background-color: var(--bg-primary);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.5rem;
  position: sticky;
  top: 0;
  z-index: 90;
}

.topbar-left {
  flex: 1;
  max-width: 400px;
  display: flex;
  align-items: center;
  gap: 1rem;
}

.mobile-menu-btn {
  display: none;
  background: transparent;
  border: none;
  color: var(--text-primary);
  font-size: 1.5rem;
  cursor: pointer;
  padding: 0;
}

.search-container {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
}

.search-icon {
  position: absolute;
  left: 1rem;
  color: var(--text-muted);
  font-size: 0.875rem;
}

.search-input {
  width: 100%;
  padding-left: 2.5rem;
  background-color: var(--bg-secondary);
  border-color: var(--bg-secondary);
  border-radius: 20px;
}

.search-input:focus {
  background-color: var(--bg-primary);
  border-color: var(--accent-primary);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.quick-trade-btn {
  font-size: 0.875rem;
  padding: 0.4rem 1rem;
  border-radius: 20px;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-icon {
  display: none;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.75rem;
  background-color: rgba(0, 200, 83, 0.05);
  border: 1px solid rgba(0, 200, 83, 0.2);
  border-radius: 20px;
  font-size: 0.75rem;
}

.pulse-success {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--trade-green);
  animation: pulse-success 2s infinite;
}

.status-text {
  color: var(--trade-green);
  font-weight: 500;
}

.latency-text {
  color: var(--text-muted);
}

.icon-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 1.25rem;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  transition: background-color 0.2s;
}

.icon-btn:hover {
  background-color: var(--bg-secondary);
  color: var(--text-primary);
}

.badge {
  position: absolute;
  top: -2px;
  right: -2px;
  background-color: var(--trade-red);
  color: white;
  font-size: 0.6rem;
  font-weight: bold;
  padding: 2px 5px;
  border-radius: 10px;
  border: 2px solid var(--bg-primary);
}

.notifications {
  position: relative;
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 0.5rem;
  background-color: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 100;
  min-width: 300px;
}

.notifications-dropdown {
  width: 320px;
  max-height: 400px;
  display: flex;
  flex-direction: column;
}

.dropdown-header {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.dropdown-header h3 {
  margin: 0;
  font-size: 1rem;
}

.dropdown-body {
  overflow-y: auto;
  max-height: 350px;
}

.empty-notif {
  padding: 1.5rem;
  text-align: center;
}

.notif-item {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background-color 0.2s;
}

.notif-item:hover {
  background-color: var(--bg-primary);
}

.notif-item.unread {
  background-color: rgba(0, 200, 83, 0.05);
  border-left: 3px solid var(--trade-green);
}

.notif-content strong {
  display: block;
  margin-bottom: 0.25rem;
  font-size: 0.9rem;
}

.notif-content p {
  margin: 0 0 0.5rem 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

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
  cursor: pointer;
}

.form-group {
  margin-bottom: 1.25rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.w-full { width: 100%; }
.mt-4 { margin-top: 1rem; }

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .topbar {
    padding: 0 1rem;
  }
  
  .mobile-menu-btn {
    display: block;
  }
  
  .desktop-only {
    display: none;
  }
  
  .search-input {
    display: none;
  }
  .search-icon {
    display: none;
  }
  
  .topbar-right {
    gap: 0.75rem;
  }
  
  .quick-trade-btn {
    padding: 0.4rem 0.8rem;
  }
  
  .btn-text {
    display: none;
  }
  
  .btn-icon {
    display: inline;
  }
}
</style>
