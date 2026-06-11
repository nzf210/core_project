<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api/client'

interface N8NStatus {
  status: 'connected' | 'disconnected' | 'unknown'
  version: string
  active_workflows: number
  queue_mode: boolean
  last_health_check: string
}

const n8nStatus = ref<N8NStatus | null>(null)
const loading = ref(true)
const error = ref('')
const showExecutions = ref(false)
const executions = ref<any[]>([])
const executionsLoading = ref(false)

let pollInterval: ReturnType<typeof setInterval> | null = null

async function fetchStatus() {
  try {
    const res = await api.getN8NStatus()
    if (res.success && res.data) {
      n8nStatus.value = res.data
    }
  } catch (e: any) {
    error.value = e.message
    n8nStatus.value = { status: 'disconnected', version: '-', active_workflows: 0, queue_mode: false, last_health_check: '-' }
  } finally {
    loading.value = false
  }
}

async function fetchExecutions() {
  executionsLoading.value = true
  try {
    const res = await api.getN8NExecutions()
    if (res.success && res.data) {
      executions.value = (Array.isArray(res.data) ? res.data : []).slice(0, 10)
    }
  } catch (e) {
    console.error('Failed to fetch executions', e)
  } finally {
    executionsLoading.value = false
  }
}

function toggleExecutions() {
  showExecutions.value = !showExecutions.value
  if (showExecutions.value && executions.value.length === 0) {
    fetchExecutions()
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function getStatusClass(status: string) {
  switch (status) {
    case 'connected': return 'status-connected'
    case 'disconnected': return 'status-disconnected'
    default: return 'status-unknown'
  }
}

onMounted(() => {
  fetchStatus()
  // Poll every 30 seconds
  pollInterval = setInterval(fetchStatus, 30000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <div class="n8n-hub">
    <!-- AC-1: N8N Status Indicator -->
    <div class="status-card" :class="getStatusClass(n8nStatus?.status || 'unknown')">
      <div class="status-header">
        <div class="status-title">
          <span class="status-dot"></span>
          <h3>N8N Automation Engine</h3>
        </div>
        <div class="status-actions">
          <a
            href="http://localhost:5678"
            target="_blank"
            rel="noopener noreferrer"
            class="btn-open-editor"
          >
            ⚡ Buka Workflow Editor
          </a>
        </div>
      </div>

      <div v-if="loading" class="loading">Checking N8N status...</div>
      <div v-else-if="error" class="error">{{ error }}</div>
      <div v-else class="status-info">
        <div class="info-grid">
          <div class="info-item">
            <span class="label">Status</span>
            <span class="value status-badge" :class="getStatusClass(n8nStatus?.status || 'unknown')">
              {{ n8nStatus?.status?.toUpperCase() || 'UNKNOWN' }}
            </span>
          </div>
          <div class="info-item">
            <span class="label">Version</span>
            <span class="value">{{ n8nStatus?.version || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="label">Active Workflows</span>
            <span class="value">{{ n8nStatus?.active_workflows || 0 }}</span>
          </div>
          <div class="info-item">
            <span class="label">Queue Mode</span>
            <span class="value">{{ n8nStatus?.queue_mode ? '✅ Enabled' : '❌ Disabled' }}</span>
          </div>
          <div class="info-item">
            <span class="label">Last Check</span>
            <span class="value">{{ formatDate(n8nStatus?.last_health_check || '') }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- AC-3: Recent Executions Widget -->
    <div class="executions-card">
      <button class="executions-toggle" @click="toggleExecutions">
        <span>📜 Recent Executions</span>
        <span class="chevron" :class="{ open: showExecutions }">▼</span>
      </button>

      <div v-show="showExecutions" class="executions-content">
        <div v-if="executionsLoading" class="loading">Loading executions...</div>
        <div v-else-if="executions.length === 0" class="empty">
          No recent executions found.
        </div>
        <table v-else class="executions-table">
          <thead>
            <tr>
              <th>Workflow</th>
              <th>Status</th>
              <th>Started</th>
              <th>Duration</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="exec in executions" :key="exec.id">
              <td>{{ exec.workflowName || exec.workflow_id || 'Unknown' }}</td>
              <td>
                <span
                  class="exec-status"
                  :class="{
                    'exec-success': exec.status === 'success',
                    'exec-error': exec.status === 'error',
                    'exec-running': exec.status === 'running'
                  }"
                >
                  {{ exec.status }}
                </span>
              </td>
              <td>{{ formatDate(exec.startedAt) }}</td>
              <td>{{ exec.stoppedAt ? `${((new Date(exec.stoppedAt).getTime() - new Date(exec.startedAt).getTime()) / 1000).toFixed(1)}s` : '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.n8n-hub {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.status-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}

.status-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.status-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--muted);
}

.status-connected .status-dot {
  background: var(--success);
  box-shadow: 0 0 8px var(--success);
}

.status-disconnected .status-dot {
  background: var(--danger);
  box-shadow: 0 0 8px var(--danger);
}

.btn-open-editor {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: var(--accent);
  color: white;
  border-radius: 6px;
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  transition: background 0.2s;
}

.btn-open-editor:hover {
  background: var(--accent-hover, #1d4ed8);
}

.status-info {
  padding: 20px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item .label {
  font-size: 11px;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-item .value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.status-badge {
  display: inline-block;
  padding: 3px 8px;
  border-radius: 4px;
  font-size: 11px;
}

.status-connected .status-badge {
  background: rgba(16, 185, 129, 0.15);
  color: var(--success);
}

.status-disconnected .status-badge {
  background: rgba(239, 68, 68, 0.15);
  color: var(--danger);
}

.status-unknown .status-badge {
  background: rgba(107, 114, 128, 0.15);
  color: var(--muted);
}

/* Executions */
.executions-card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 10px;
  overflow: hidden;
}

.executions-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.executions-toggle:hover {
  background: var(--bg);
}

.chevron {
  font-size: 10px;
  transition: transform 0.2s;
}

.chevron.open {
  transform: rotate(180deg);
}

.executions-content {
  padding: 0 20px 20px;
}

.executions-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.executions-table th {
  text-align: left;
  padding: 8px 12px;
  color: var(--muted);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-bottom: 1px solid var(--border);
}

.executions-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
}

.exec-status {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
}

.exec-success {
  background: rgba(16, 185, 129, 0.15);
  color: var(--success);
}

.exec-error {
  background: rgba(239, 68, 68, 0.15);
  color: var(--danger);
}

.exec-running {
  background: rgba(59, 130, 246, 0.15);
  color: var(--accent);
}

.loading, .empty {
  text-align: center;
  color: var(--muted);
  padding: 20px;
}
</style>