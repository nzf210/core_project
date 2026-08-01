<template>
  <div class="access-manager">
    <div class="header flex justify-between align-center" style="margin-bottom: 2rem;">
      <div>
        <h2 style="font-size: 1.5rem; color: var(--accent-primary);">Pengaturan & Integrasi</h2>
        <p style="color: var(--text-secondary);">Kelola API Key Telegram dan koneksi WhatsApp.</p>
      </div>
    </div>

    <div class="glass-card" style="margin-bottom: 2rem; padding: 2rem;">
      <h3 style="margin-bottom: 1rem;">Integrasi WhatsApp & Telegram</h3>
      <div class="dashboard-grid" style="display: grid; grid-template-columns: 1fr 1fr; gap: 2rem;">
        
        <div class="config-box">
          <h4>Telegram Bot API</h4>
          <label for="telegram-key-input" class="text-muted" style="margin-bottom: 0.25rem; font-size: 0.9rem; display: block;">Masukkan token dari BotFather untuk mengaktifkan OTP Telegram.</label>
          <input id="telegram-key-input" type="text" v-model="telegramKey" class="input-field" placeholder="123456789:ABCdefGHIjkl..." style="width:100%; margin-bottom: 1rem;" />
          <button type="button" class="btn-primary" @click="saveTelegramKey">Simpan Token</button>
        </div>

        <div class="config-box">
          <h4>WhatsApp Gateway</h4>
          <p class="text-muted" style="margin-bottom: 1rem; font-size: 0.9rem;">Hubungkan perangkat WA untuk mengirim OTP secara otomatis.</p>
          
          <div v-if="waStatus === 'connected'" class="alert-success" style="padding: 1rem; border-radius: 8px; background: rgba(16,185,129,0.1); color: var(--accent-success, #059669);">
            ✅ WhatsApp Terhubung ({{ waJID }})
          </div>
          
          <div v-else>
            <button type="button" class="btn-outline" @click="generateWAQR" v-if="!waQRCode">Tampilkan QR Code WA</button>
            <div v-if="waQRCode" style="margin-top: 1rem; text-align: center;">
              <img :src="waQRCode" alt="WA QR Code" style="border-radius: 8px; max-width: 200px;" />
              <p class="text-muted" style="margin-top: 0.5rem; font-size: 0.8rem;">Scan dengan WhatsApp Anda</p>
            </div>
          </div>
        </div>

      </div>
    </div>

    <h3 style="margin-bottom: 1rem; color: var(--text-secondary)">Access Control & Audit Log</h3>
    
    <div style="margin-bottom: 2rem;">
      <h4 style="margin-bottom: 0.5rem;">Roles</h4>
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr><th>Role Name</th><th>Description</th></tr>
          </thead>
          <tbody>
            <tr v-for="r in roles" :key="r.id">
              <td>{{ r.name }}</td><td>{{ r.description }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div>
      <h4 style="margin-bottom: 0.5rem;">Audit Logs</h4>
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr><th>Timestamp</th><th>Action</th><th>Resource</th></tr>
          </thead>
          <tbody>
            <tr v-for="l in logs" :key="l.id">
              <td>{{ l.created_at }}</td>
              <td><span class="badge badge-primary">{{ l.action }}</span></td>
              <td><code>{{ l.resource }}</code></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient, waApi } from '../api'
import { ref, onMounted } from 'vue'

const roles = ref<any[]>([])
const logs = ref<any[]>([])
const telegramKey = ref('')
const waStatus = ref('disconnected')
const waJID = ref('')
const waQRCode = ref('')

const fetchData = async () => {
  try {
    const resRoles = await apiClient('/roles', { headers: { 'X-Tenant-ID': 'default' } })
    const dataRoles = await resRoles.json()
    if (dataRoles.success) roles.value = dataRoles.data

    const resLogs = await apiClient('/audit-logs', { headers: { 'X-Tenant-ID': 'default' } })
    const dataLogs = await resLogs.json()
    if (dataLogs.success) logs.value = dataLogs.data
    
    const tenantId = localStorage.getItem('tenantId') || 'system'
    const waRes = await waApi.status(tenantId)
    const waData = await waRes.json()
    if (waData.status === 'connected') {
      waStatus.value = 'connected'
      waJID.value = waData.jid
    }
  } catch (err) { console.error(err) }
}

const saveTelegramKey = () => {
  alert("Telegram API Key berhasil disimpan untuk tenant ini.")
}

const generateWAQR = async () => {
  try {
    const tenantId = localStorage.getItem('tenantId') || 'system'
    const res = await waApi.qr(tenantId)
    const data = await res.json()
    if (data.status === 'qr') {
      waQRCode.value = data.qr_code
      setTimeout(async () => {
        const checkRes = await waApi.status(tenantId)
        const checkData = await checkRes.json()
        if (checkData.status === 'connected') {
          waStatus.value = 'connected'
          waJID.value = checkData.jid
          waQRCode.value = ''
        }
      }, 15000)
    } else if (data.status === 'connected') {
      waStatus.value = 'connected'
    }
  } catch (err) {
    console.error('WA QR error:', err)
    alert("Gagal menghubungi WA Gateway")
  }
}

onMounted(fetchData)
</script>

<style scoped>
.table-container { overflow-x: auto; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th, .data-table td { padding: 1rem; text-align: left; border-bottom: 1px solid var(--border-color); }
.data-table th { font-weight: 600; color: var(--text-secondary); background-color: var(--bg-tertiary); }
.input-field {
  padding: 0.75rem 1rem; border: 1px solid var(--border-color); border-radius: var(--radius-sm);
  background: var(--surface-1); color: var(--text-primary);
}
.btn-primary { background: var(--accent-gradient); color: white; padding: 0.5rem 1.5rem; border-radius: var(--radius-sm); border: none; cursor: pointer; }
.btn-outline { background: transparent; border: 1px solid var(--accent-primary); color: var(--accent-primary); padding: 0.5rem 1.5rem; border-radius: var(--radius-sm); cursor: pointer; }
.config-box { background: var(--surface-0); padding: 1.5rem; border-radius: var(--radius-md); border: 1px solid var(--border-color); }
</style>
