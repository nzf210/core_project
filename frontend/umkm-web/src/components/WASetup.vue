<template>
  <div class="wa-setup-page">
    <div class="header-actions" style="margin-bottom: 1.5rem;">
      <h2>📱 Setup WhatsApp & Provider</h2>
      <p>Pilih jalur pengiriman WhatsApp: Gratis (Whatsmeow) atau Resmi (Meta Cloud API).</p>
    </div>

    <div v-if="loading" class="glass-card" style="padding: 2rem; text-align: center;">
      <p>Memuat status...</p>
    </div>
    
    <div v-else class="setup-layout" style="display: grid; grid-template-columns: 2fr 1fr; gap: 1.5rem;">
      <div class="glass-card" style="padding: 1.5rem;">
        <h3 style="margin-bottom: 1rem;">Provider Utama</h3>
        <p style="font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 1rem;">
          Pilih provider yang ingin Anda gunakan. Mode Auto akan memprioritaskan Cloud API untuk notifikasi penting (jika aktif) dan Whatsmeow untuk chatbot.
        </p>

        <div style="display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 2rem;">
          <label class="provider-card" :class="{ active: provider === 'auto' }">
            <input type="radio" value="auto" v-model="provider" @change="saveProvider" />
            <div class="provider-info">
              <strong>⚡ Auto (Hybrid) — Rekomendasi</strong>
              <span class="desc">Notifikasi tagihan/OTP via Cloud API, Chatbot via Whatsmeow.</span>
            </div>
          </label>
          
          <label class="provider-card" :class="{ active: provider === 'whatsmeow' }">
            <input type="radio" value="whatsmeow" v-model="provider" @change="saveProvider" />
            <div class="provider-info">
              <strong>📱 Whatsmeow Only</strong>
              <span class="desc">Gratis, tanpa kuota. Risiko blokir Meta (banned) lebih tinggi untuk broadcast.</span>
            </div>
          </label>
          
          <label class="provider-card" :class="{ active: provider === 'cloud_api', locked: !waSetupState.can_use_cloud_api }">
            <input type="radio" value="cloud_api" v-model="provider" :disabled="!waSetupState.can_use_cloud_api" @change="saveProvider" />
            <div class="provider-info">
              <strong>☁️ Cloud API (Meta Official)</strong>
              <span class="desc">Resmi, bebas blokir. Butuh add-on dan credit (saldo berbayar).</span>
              <span v-if="!waSetupState.can_use_cloud_api" class="badge locked-badge">Butuh Add-on</span>
            </div>
          </label>
        </div>

        <h3 style="margin-bottom: 1rem;">Status Whatsmeow</h3>
        <div class="status-box" :class="waSetupState.whatsmeow.status">
          <div class="status-header">
            <span class="status-dot"></span>
            <strong>{{ waSetupState.whatsmeow.connected ? 'Terhubung' : (waSetupState.whatsmeow.status === 'qr_pending' ? 'Menunggu Scan QR' : 'Terputus') }}</strong>
          </div>
          <p class="status-desc">
            Koneksi pihak ketiga ke WhatsApp Web. Harus terhubung agar chatbot bisa membalas.
          </p>
          <div v-if="!waSetupState.whatsmeow.connected" class="action-row">
            <button class="btn btn-primary" @click="requestQR">Generate QR Code</button>
          </div>
        </div>

        <h3 style="margin-bottom: 1rem; margin-top: 2rem;">Status Cloud API</h3>
        <div class="status-box" :class="waSetupState.cloud_api.active ? 'connected' : 'disconnected'">
           <div class="status-header">
            <span class="status-dot"></span>
            <strong>{{ waSetupState.cloud_api.active ? 'Aktif' : 'Belum Dikonfigurasi' }}</strong>
          </div>
          <div v-if="waSetupState.cloud_api.active" style="margin-top: 1rem; background: var(--bg-primary); padding: 1rem; border-radius: 0.5rem;">
            <div style="display: flex; justify-content: space-between; margin-bottom: 0.5rem;">
              <span>Kredit Tersedia:</span>
              <strong style="color: var(--success);">Rp {{ formatPrice(waSetupState.cloud_api.credit_balance_cents) }}</strong>
            </div>
            <div style="display: flex; justify-content: space-between;">
              <span>Pemakaian Bulan Ini:</span>
              <strong style="color: var(--warning);">Rp {{ formatPrice(waSetupState.cloud_api.credit_used_cents) }}</strong>
            </div>
            <button class="btn btn-secondary" style="margin-top: 1rem; width: 100%;">Top Up Kredit</button>
          </div>
          <div v-else class="action-row">
            <button class="btn btn-primary" :disabled="!waSetupState.can_use_cloud_api">Hubungkan ke Meta</button>
          </div>
        </div>
      </div>

      <div class="glass-card preview-card" style="padding: 1.25rem;">
        <h4 style="margin-bottom: 1rem; color: #f59e0b;">⚠️ Peringatan Whatsmeow</h4>
        <p style="font-size: 0.85rem; color: var(--text-secondary); line-height: 1.5; margin-bottom: 1rem;">
          Whatsmeow adalah koneksi tidak resmi (unofficial). Menggunakan Whatsmeow untuk <strong>broadcast promosi massal</strong> sangat rawan menyebabkan nomor WhatsApp Anda diblokir (banned) secara permanen oleh pihak Meta.
        </p>
        <p style="font-size: 0.85rem; color: var(--text-secondary); line-height: 1.5;">
          Sangat disarankan menggunakan nomor cadangan (bukan nomor utama bisnis) jika menggunakan Whatsmeow. Untuk keamanan maksimal, upgrade ke Cloud API.
        </p>

        <h4 style="margin-bottom: 0.75rem; margin-top: 2rem; color: #3b82f6;">💳 Tarif Cloud API</h4>
        <ul style="font-size: 0.85rem; color: var(--text-secondary); padding-left: 1.2rem; line-height: 1.6;">
          <li>Pesan Percakapan (Chatbot): ~Rp 450 / pesan</li>
          <li>Notifikasi Otentikasi (OTP): ~Rp 300 / pesan</li>
          <li>Broadcast Marketing: ~Rp 600 / pesan</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

const loading = ref(true)
const provider = ref('auto')
const waSetupState = ref({
  wa_provider_preference: 'auto',
  can_use_cloud_api: false,
  whatsmeow: { connected: false, status: 'disconnected' },
  cloud_api: { active: false, credit_balance_cents: 0, credit_used_cents: 0 }
})

const formatPrice = (sen: number) => {
  return (sen / 100).toLocaleString('id-ID')
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await api.getWASetup()
    if (res.success) {
      waSetupState.value = res.data
      provider.value = res.data.wa_provider_preference
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const saveProvider = async () => {
  try {
    const res = await api.updateWAProvider(provider.value as 'auto' | 'whatsmeow' | 'cloud_api')
    if (res.success) {
      alert('Preferensi provider berhasil disimpan')
    }
  } catch (e) {
    console.error(e)
    alert('Gagal menyimpan preferensi')
  }
}

const requestQR = () => {
  alert('Fitur scan QR akan memanggil WA Gateway endpoint...')
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.provider-card {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  padding: 1rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-tertiary);
  cursor: pointer;
  transition: all 0.2s;
}
.provider-card:hover:not(.locked) {
  border-color: #4f46e5;
}
.provider-card.active {
  border-color: #4f46e5;
  background: rgba(79, 70, 229, 0.1);
}
.provider-card.locked {
  opacity: 0.6;
  cursor: not-allowed;
}
.provider-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.provider-info .desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
}
.locked-badge {
  background: #dc2626;
  color: white;
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  width: fit-content;
  margin-top: 0.25rem;
}
.status-box {
  padding: 1rem;
  border: 1px solid var(--border-color);
  border-radius: 0.5rem;
  background: var(--bg-tertiary);
}
.status-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #6b7280;
}
.status-box.connected .status-dot { background: #10b981; }
.status-box.qr_pending .status-dot { background: #f59e0b; }
.status-box.disconnected .status-dot { background: #dc2626; }
.status-desc {
  font-size: 0.85rem;
  color: var(--text-secondary);
  margin-bottom: 1rem;
}
</style>
