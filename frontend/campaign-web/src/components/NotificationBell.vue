<template>
  <div class="notification-container">
    <button class="bell-btn" @click="isOpen = !isOpen">
      🔔
      <span class="badge-count" v-if="unreadCount > 0">{{ unreadCount }}</span>
    </button>
    <div class="dropdown" v-if="isOpen">
      <div class="dropdown-header">Notifications</div>
      <ul class="notification-list">
        <li v-for="n in notifications" :key="n.id" :class="{ unread: n.status === 'unread' }">
          <strong>{{ n.title }}</strong>
          <p>{{ n.message }}</p>
        </li>
        <li v-if="notifications.length === 0" class="empty">No notifications</li>
      </ul>
      <div class="dropdown-footer">
        <button @click="sendBroadcast">Test Broadcast</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'

import { ref, onMounted, computed } from 'vue'

const isOpen = ref(false)
const notifications = ref<any[]>([])
const unreadCount = computed(() => notifications.value.filter(n => n.status === 'unread').length)

const fetchNotifications = async () => {
  try {
    const res = await apiClient('/notifications', { headers: { 'X-Tenant-ID': 'default' } })
    const data = await res.json()
    if (data.success) notifications.value = data.data
  } catch (err) { console.error(err) }
}

const sendBroadcast = async () => {
  try {
    await apiClient('/notifications', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default' },
      body: JSON.stringify({ title: 'New Broadcast', message: 'Hello all volunteers!', type: 'broadcast' })
    })
    fetchNotifications()
  } catch (err) { console.error(err) }
}

onMounted(fetchNotifications)
</script>

<style scoped>
.notification-container { position: relative; }
.bell-btn {
  background: none; border: none; font-size: 1.5rem; cursor: pointer; position: relative;
}
.badge-count {
  position: absolute; top: -5px; right: -5px; background: var(--accent-primary);
  color: white; font-size: 0.75rem; font-weight: bold; padding: 2px 6px; border-radius: 999px;
}
.dropdown {
  position: absolute; top: 100%; right: 0; width: 300px; background: var(--bg-secondary);
  border: 1px solid var(--border-color); border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg); z-index: 100; margin-top: 0.5rem;
}
.dropdown-header {
  padding: 1rem; border-bottom: 1px solid var(--border-color); font-weight: 600;
}
.notification-list { list-style: none; max-height: 300px; overflow-y: auto; }
.notification-list li { padding: 1rem; border-bottom: 1px solid var(--border-color); font-size: 0.875rem; }
.notification-list li.unread { background: rgba(220,38,38,0.05); }
.notification-list p { color: var(--text-muted); margin-top: 0.25rem; }
.empty { text-align: center; color: var(--text-muted); }
.dropdown-footer { padding: 0.5rem; text-align: center; border-top: 1px solid var(--border-color); }
.dropdown-footer button {
  background: none; border: none; color: var(--accent-primary); cursor: pointer; font-weight: 600;
}
</style>
