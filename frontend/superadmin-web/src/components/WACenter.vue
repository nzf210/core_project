<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api/client'

const status = ref<any>(null)
const qrData = ref<any>(null)
const loading = ref(true)
const loadingQR = ref(false)
const error = ref('')
const pollInterval = ref<ReturnType<typeof setInterval>>()

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
    const res = await fetch('/admin/wa/logout', {
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

onMounted(() => {
  fetchStatus()
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
</style>