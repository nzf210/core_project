<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api/client'

const health = ref<any>(null)
const loading = ref(true)
const error = ref('')
const lastRefresh = ref<Date | null>(null)
let interval: ReturnType<typeof setInterval>

const statusColor = (status: string) => {
  switch (status) {
    case 'up': return 'var(--success)'
    case 'down': return 'var(--danger)'
    default: return 'var(--warning)'
  }
}

const statusLabel = (status: string) => {
  switch (status) {
    case 'up': return 'Connected'
    case 'down': return 'Offline'
    case 'degraded': return 'Degraded'
    default: return 'Unknown'
  }
}



function extractMetric(raw: string, metricName: string): string {
  const lines = raw.split('\n')
  for (const line of lines) {
    if (line.startsWith(metricName)) {
      // Extract value after last space
      const parts = line.trim().split(/\s+/)
      return parts[parts.length - 1]
    }
  }
  return '-'
}

onMounted(async () => {
  const fetchHealth = async () => {
    try {
      const res = await api.getHealthStatus()
      health.value = res.data
      lastRefresh.value = new Date()
      error.value = ''
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  await fetchHealth()
  interval = setInterval(fetchHealth, 30000) // Refresh every 30s
})

onUnmounted(() => {
  if (interval) clearInterval(interval)
})
</script>

<template>
  <div class="monitor">
    <div class="header">
      <h2>HA Monitoring</h2>
      <div class="meta">
        <span v-if="lastRefresh" class="refresh">
          Last refresh: {{ lastRefresh.toLocaleTimeString('id-ID') }}
        </span>
        <span :class="['status-badge', health?.status]">
          {{ health?.status || 'loading' }}
        </span>
      </div>
    </div>

    <div v-if="loading" class="loading">Loading...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="health">
      <div class="services-grid">
        <div
          v-for="svc in health.services"
          :key="svc.name"
          class="service-card"
          :class="svc.status"
        >
          <div class="svc-header">
            <span class="svc-name">{{ svc.name }}</span>
            <span class="svc-status" :style="{ color: statusColor(svc.status) }">
              {{ statusLabel(svc.status) }}
            </span>
          </div>

          <!-- WA Gateway specific metrics -->
          <div v-if="svc.name === 'wa-gateway' && svc.metrics" class="metrics">
            <div class="metric">
              <span class="metric-label">Connected Sessions</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'wa_gateway_connected_sessions') }}
              </span>
            </div>
            <div class="metric">
              <span class="metric-label">Active Instances</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'wa_gateway_active_instances') }}
              </span>
            </div>
            <div class="metric">
              <span class="metric-label">Messages Sent</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'wa_gateway_messages_sent_total') }}
              </span>
            </div>
            <div class="metric">
              <span class="metric-label">Messages Received</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'wa_gateway_messages_received_total') }}
              </span>
            </div>
          </div>

          <!-- Chatbot specific metrics -->
          <div v-if="svc.name === 'umkm-chatbot' && svc.metrics" class="metrics">
            <div class="metric">
              <span class="metric-label">Messages Processed</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'chatbot_messages_processed_total') }}
              </span>
            </div>
            <div class="metric">
              <span class="metric-label">LLM Calls</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'chatbot_llm_calls_total') }}
              </span>
            </div>
            <div class="metric">
              <span class="metric-label">Queue Depth</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'chatbot_queue_depth') }}
              </span>
            </div>
            <div class="metric">
              <span class="metric-label">Errors</span>
              <span class="metric-value">
                {{ extractMetric(svc.metrics, 'chatbot_errors_total') }}
              </span>
            </div>
          </div>

          <!-- Generic health for other services -->
          <div v-if="!svc.metrics" class="simple-status">
            Port {{ svc.port }}
          </div>
        </div>
      </div>

      <div class="footer">
        <span class="env">Env: <code>{{ health.env }}</code></span>
        <span class="checked">Checked: {{ health.checked_at }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.monitor {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

h2 {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
}

.meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.refresh {
  font-size: 12px;
  color: var(--muted);
}

.status-badge {
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 20px;
  text-transform: uppercase;
}

.status-badge.healthy { background: rgba(16, 185, 129, 0.15); color: var(--success); }
.status-badge.degraded { background: rgba(245, 158, 11, 0.15); color: var(--warning); }
.status-badge.down { background: rgba(239, 68, 68, 0.15); color: var(--danger); }
.status-badge.loading { background: var(--bg); color: var(--muted); }

.services-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.service-card {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 16px;
  transition: border-color 0.2s;
}

.service-card.up { border-left: 3px solid var(--success); }
.service-card.down { border-left: 3px solid var(--danger); }
.service-card.degraded { border-left: 3px solid var(--warning); }

.svc-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.svc-name {
  font-weight: 600;
  font-size: 14px;
}

.svc-status {
  font-size: 12px;
  font-weight: 600;
}

.metrics {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.metric {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.metric-label { color: var(--muted); }
.metric-value { font-weight: 600; font-family: monospace; }

.simple-status {
  font-size: 12px;
  color: var(--muted);
}

.footer {
  display: flex;
  justify-content: space-between;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border);
  font-size: 12px;
  color: var(--muted);
}

code { background: var(--bg); padding: 2px 6px; border-radius: 4px; }

.loading, .error { text-align: center; padding: 40px; color: var(--muted); }
.error { color: var(--danger); }
</style>