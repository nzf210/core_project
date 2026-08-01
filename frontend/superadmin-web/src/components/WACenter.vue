<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api/client'

const status = ref<any>(null)
const qrData = ref<any>(null)
const loading = ref(true)
const loadingQR = ref(false)
const error = ref('')
const pollInterval = ref<ReturnType<typeof setInterval>>()

// F064: platform WA provider
const platformProvider = ref<any>(null)
const loadingProvider = ref(false)
const savingProvider = ref(false)

async function fetchStatus() {
  try {
    const res = await api.getWAStatus()
    status.value = res
    error.value = ''
    if (res.status === 'connected') {
      stopPoll()
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function requestQR() {
  loadingQR.value = true
  qrData.value = null
  error.value = ''
  try {
    const res = await api.getWAQR()
    qrData.value = res
    if (res.status === 'connected') {
      status.value = res
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loadingQR.value = false
  }
}

async function disconnectWA() {
  if (!confirm('Disconnect WA Center? User tidak bisa register/login via WA sampai reconnect.')) return
  try {
    const apiBase = import.meta.env.VITE_API_URL || ''
    const res = await fetch(`${apiBase}/admin/wa/logout`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`,
        'X-Tenant-ID': 'system'
      }
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Disconnect failed')
    status.value = { status: 'disconnected' }
    qrData.value = null
  } catch (e: any) {
    error.value = e.message
  }
}

function startPoll() {
  pollInterval.value = setInterval(fetchStatus, 5000)
}

function stopPoll() {
  if (pollInterval.value) {
    clearInterval(pollInterval.value)
    pollInterval.value = undefined
  }
}

async function fetchPlatformProvider() {
  try {
    platformProvider.value = await api.getPlatformProvider()
  } catch (e: any) {
    // non-critical — silently fail
  }
}

async function savePlatformProvider(wa_provider: string) {
  savingProvider.value = true
  try {
    await api.setPlatformProvider(wa_provider)
    await fetchPlatformProvider()
    await fetchStatus()
  } catch (e: any) {
    error.value = e.message
  } finally {
    savingProvider.value = false
  }
}

onMounted(() => {
  fetchStatus()
  fetchPlatformProvider()
  startPoll()
})

onUnmounted(() => stopPoll())

function extractPhone(jid: string) {
  if (!jid) return ''
  const num = jid.split('@')[0]
  return num.startsWith('62') ? '0' + num.slice(2) : num
}
</script>

<template>
  <div class="wa-center">
    <div class="header">
      <div class="title-row">
        <h2>📱 WA Center</h2>
        <span v-if="!loading" :class="['badge', status?.status === 'connected' ? 'connected' : 'disconnected']">
          {{ status?.status === 'connected' ? '● Connected' : '○ Disconnected' }}
        </span>
      </div>
      <p class="desc">WhatsApp platform untuk REG/OTP/VERIF semua tenant SaaS UMKM</p>
    </div>

    <!-- F064: Platform WA Provider Selector -->
    <div class="provider-selector">
      <div class="provider-header">
        <label for="platform-wa-provider" class="provider-label">📡 WA Provider</label>
        <select
          id="platform-wa-provider"
          class="provider-select"
          :value="platformProvider?.data?.wa_provider || 'auto'"
          :disabled="savingProvider"
          @change="savePlatformProvider(($event.target as HTMLSelectElement).value)"
        >
          <option value="auto">🔄 Auto Detect</option>
          <option value="whatsmeow">📱 Paksa Whatsmeow</option>
          <option value="cloud_api">⚡ Paksa Cloud API (Meta)</option>
        </select>
      </div>
      <div class="provider-info" v-if="platformProvider?.data">
        <div class="provider-status-row">
          <span class="provider-mode">
            Mode: <strong>{{ platformProvider.data.effective_provider === 'cloud_api' ? '⚡ Cloud API (Meta Official)' : '📱 Whatsmeow (WA Center)' }}</strong>
          </span>
          <span class="provider-reason">{{ platformProvider.data.reason }}</span>
        </div>
        <div class="connection-badges">
          <span :class="['conn-badge', platformProvider.data.connections?.whatsmeow?.connected ? 'ok' : 'no']">
            📱 Whatsmeow {{ platformProvider.data.connections?.whatsmeow?.connected ? '✅' : '❌' }}
          </span>
          <span :class="['conn-badge', platformProvider.data.connections?.cloud_api?.active ? 'ok' : 'no']">
            ⚡ Cloud API {{ platformProvider.data.connections?.cloud_api?.active ? '✅' : '❌' }}
          </span>
        </div>
        <div class="provider-hint" v-if="platformProvider.data.effective_provider === 'whatsmeow'">
          <em>User harus chat REG/OTP/VERIF ke WA Center — sistem tidak kirim OTP otomatis.</em>
        </div>
        <div class="provider-hint" v-else>
          <em>Sistem kirim OTP otomatis via Cloud API — user tidak perlu chat duluan.</em>
        </div>
      </div>
      <div v-else-if="!loadingProvider" class="provider-loading">Memuat provider...</div>
    </div>

    <div v-if="loading" class="loading">Memuat status WA...</div>
    <div v-else-if="error && !status" class="error">{{ error }}</div>

    <div v-else class="content">
      <!-- Connected state -->
      <div v-if="status?.status === 'connected'" class="connected-state">
        <div class="wa-info">
          <div class="info-row">
            <span class="label">Nomor WA</span>
            <span class="value">{{ extractPhone(status?.jid) }}</span>
          </div>
          <div class="info-row">
            <span class="label">JID</span>
            <span class="value jid">{{ status?.jid }}</span>
          </div>
          <div v-if="status?.owner" class="info-row">
            <span class="label">Instance</span>
            <span class="value">{{ status.owner }}</span>
          </div>
        </div>
        <div class="actions">
          <button class="btn disconnect" @click="disconnectWA">🔌 Disconnect</button>
          <button class="btn refresh" @click="fetchStatus">🔄 Refresh</button>
        </div>
      </div>

      <!-- Disconnected / QR state -->
      <div v-else class="disconnected-state">
        <div class="qr-section" v-if="qrData?.status === 'qr'">
          <p class="qr-hint">Scan QR ini dengan WhatsApp номер:</p>
          <img :src="qrData.qr_code" alt="WA QR Code" class="qr-img" />
          <p class="qr-expire">QR Code expires in 10 menit</p>
          <button class="btn refresh" @click="requestQR">🔄 Refresh QR</button>
        </div>
        <div v-else-if="qrData?.message === 'Already connected'" class="already-connected">
          <p>✅ WA Center sudah terhubung. Refresh untuk update status.</p>
          <button class="btn refresh" @click="fetchStatus">🔄 Refresh</button>
        </div>
        <div v-else class="connect-action">
          <p class="desc-connect">WA Center belum terhubung. Koneksikan untuk mengaktifkan REG/OTP/VERIF.</p>
          <button
            class="btn connect"
            @click="requestQR"
            :disabled="loadingQR"
          >
            {{ loadingQR ? '⏳ Memuat...' : '📱 Connect WA Center' }}
          </button>
          <p v-if="error" class="error-inline">{{ error }}</p>
        </div>
      </div>

      <!-- Usage guide -->
      <div class="usage-guide">
        <h3>Cara Kerja</h3>
        <div class="guide-grid">
          <div class="guide-item">
            <span class="guide-cmd">REG / DAFTAR</span>
            <span>User kirim pesan ke WA Center → pendaftaran penuh via WA</span>
          </div>
          <div class="guide-item">
            <span class="guide-cmd">OTP</span>
            <span>User kirim OTP → dapat kode login via WA</span>
          </div>
          <div class="guide-item">
            <span class="guide-cmd">VERIF [kode]</span>
            <span>Verifikasi pendaftaran dari web (tanpa OTP otomatis)</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wa-center {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
}

.header { margin-bottom: 20px; }
.title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 4px;
}
h2 { font-size: 18px; font-weight: 600; margin: 0; }
.desc { font-size: 13px; color: var(--muted); margin: 0; }

.badge {
  font-size: 12px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 20px;
}
.badge.connected { background: rgba(16,185,129,0.15); color: var(--success); }
.badge.disconnected { background: rgba(239,68,68,0.15); color: var(--danger); }

.loading, .error { text-align: center; padding: 20px; color: var(--muted); }
.error { color: var(--danger); }

.content { display: flex; flex-direction: column; gap: 20px; }

/* Connected */
.wa-info { display: flex; flex-direction: column; gap: 8px; }
.info-row { display: flex; justify-content: space-between; font-size: 14px; }
.label { color: var(--muted); }
.value { font-weight: 600; font-family: monospace; }
.value.jid { font-size: 12px; color: var(--text); }

.actions { display: flex; gap: 8px; flex-wrap: wrap; }

/* Buttons */
.btn {
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid var(--border);
  transition: all 0.2s;
}
.btn.connect {
  background: var(--accent);
  color: white;
  border-color: var(--accent);
}
.btn.connect:hover:not(:disabled) { opacity: 0.85; }
.btn.connect:disabled { opacity: 0.5; cursor: not-allowed; }
.btn.disconnect {
  background: rgba(239,68,68,0.1);
  color: var(--danger);
  border-color: rgba(239,68,68,0.3);
}
.btn.disconnect:hover { background: rgba(239,68,68,0.2); }
.btn.refresh {
  background: var(--bg);
  color: var(--text);
}
.btn.refresh:hover { background: var(--border); }

/* QR */
.qr-section { text-align: center; }
.qr-hint { font-size: 14px; color: var(--muted); margin-bottom: 12px; }
.qr-img {
  max-width: 220px;
  border-radius: 12px;
  border: 2px solid var(--border);
  margin: 0 auto;
  display: block;
}
.qr-expire { font-size: 12px; color: var(--muted); margin: 8px 0; }

.connect-action { text-align: center; }
.desc-connect { font-size: 14px; color: var(--muted); margin-bottom: 16px; }
.error-inline { color: var(--danger); font-size: 13px; margin-top: 8px; }
.already-connected { text-align: center; font-size: 14px; color: var(--success); }

/* Usage guide */
.usage-guide {
  border-top: 1px solid var(--border);
  padding-top: 16px;
}
.usage-guide h3 { font-size: 14px; font-weight: 600; margin: 0 0 12px 0; }
.guide-grid { display: flex; flex-direction: column; gap: 8px; }
.guide-item {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  font-size: 13px;
}
.guide-cmd {
  background: var(--bg);
  border: 1px solid var(--border);
  padding: 2px 8px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  min-width: 100px;
  text-align: center;
  color: var(--accent);
}

/* F064: Platform WA Provider Selector */
.provider-selector {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.provider-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.provider-label { font-size: 13px; font-weight: 600; color: var(--text); }
.provider-select {
  padding: 5px 10px;
  border-radius: 6px;
  border: 1px solid var(--border);
  font-size: 13px;
  background: var(--card);
  color: var(--text);
  cursor: pointer;
}
.provider-select:disabled { opacity: 0.5; cursor: not-allowed; }
.provider-info { display: flex; flex-direction: column; gap: 6px; }
.provider-status-row { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; font-size: 13px; }
.provider-mode { font-weight: 500; }
.provider-reason { color: var(--muted); font-size: 12px; font-family: monospace; }
.connection-badges { display: flex; gap: 8px; flex-wrap: wrap; }
.conn-badge {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 20px;
  border: 1px solid var(--border);
}
.conn-badge.ok { color: var(--success); border-color: rgba(16,185,129,0.3); background: rgba(16,185,129,0.05); }
.conn-badge.no { color: var(--muted); }
.provider-hint { font-size: 12px; color: var(--muted); }
.provider-loading { font-size: 13px; color: var(--muted); text-align: center; }
</style>