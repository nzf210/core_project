<template>
  <div class="settings-page">
    <div class="header-actions flex items-center justify-between" style="margin-bottom: 2rem;">
      <h2>Pengaturan</h2>
      <p>Kelola akun, profil toko, dan integrasi</p>
    </div>

    <!-- Pengaturan Akun -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">Pengaturan Akun</h3>
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <input type="text" placeholder="Username" v-model="profileForm.username" class="form-control" />
        <input type="text" placeholder="Nomor HP" v-model="profileForm.phone_number" class="form-control" />
        <div class="divider" style="border-top: 1px solid var(--border-color); margin: 0.5rem 0;"></div>
        <h4 style="margin-bottom: 0; font-size: 1rem;">Ganti Password</h4>
        <p style="font-size: 0.85rem; color: var(--text-secondary); margin-top: -0.5rem;">Kosongkan jika tidak ingin mengubah password</p>
        <input type="password" placeholder="Password Saat Ini" v-model="profileForm.old_password" class="form-control" />
        <input type="password" placeholder="Password Baru" v-model="profileForm.new_password" class="form-control" />
        <button class="btn btn-primary" @click="saveAccount" :disabled="loadingAccount">
          {{ loadingAccount ? 'Menyimpan...' : 'Simpan Akun' }}
        </button>
      </div>
    </div>

    <!-- Profil Toko -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">Profil Toko</h3>

      <div class="profile-logo-section" style="margin-bottom: 1.5rem; text-align: center;">
        <div class="logo-preview"
          style="width: 80px; height: 80px; border-radius: 50%; overflow: hidden; margin: 0 auto 0.5rem; background: var(--bg-tertiary); display: flex; align-items: center; justify-content: center;">
          <img v-if="profileForm.logo_url" :src="profileForm.logo_url" alt="Logo"
            style="width: 100%; height: 100%; object-fit: cover;" />
          <span v-else style="font-size: 2rem; color: var(--text-secondary);">🏪</span>
        </div>
        <label class="btn btn-secondary" style="cursor: pointer; font-size: 0.8rem; padding: 0.3rem 0.8rem;">
          <input type="file" accept="image/png,image/jpeg,image/webp" @change="handleLogoUpload" hidden />
          {{ uploadingLogo ? 'Mengupload...' : 'Upload Logo' }}
        </label>
      </div>

      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <input type="text" placeholder="Nama Usaha" v-model="profileForm.business_name" class="form-control" />
        <textarea placeholder="Alamat Usaha" v-model="profileForm.business_address" class="form-control" rows="2" />
        <select v-model="profileForm.business_type" class="form-control">
          <option value="" disabled>Pilih Jenis Usaha</option>
          <option value="umum">Umum</option>
          <option value="warung">Warung</option>
          <option value="laundry">Laundry</option>
          <option value="industri_kreatif">Industri Kreatif</option>
          <option value="toko_online">Toko Online</option>
          <option value="restoran">Restoran</option>
          <option value="jasa">Jasa</option>
        </select>
        <input type="text" placeholder="Nomor WA (dengan kode negara, contoh: 6281...)" v-model="profileForm.wa_number" class="form-control" />
        <button class="btn btn-primary" @click="saveStore" :disabled="loadingStore">
          {{ loadingStore ? 'Menyimpan...' : 'Simpan Toko' }}
        </button>
      </div>
    </div>

    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">Integrasi WhatsApp (QR Code)</h3>
      <p style="color: var(--text-secondary); margin-bottom: 2rem;">
        Hubungkan nomor WhatsApp utama toko Anda untuk bertindak sebagai Chatbot pintar yang melayani pelanggan secara
        mandiri.
      </p>

      <div class="wa-status-box" :class="waStatus">
        <div class="status-indicator"></div>
        <span class="status-text">
          {{ waStatus === 'checking' ? 'Memeriksa status...' :
            waStatus === 'connected' ? 'WhatsApp Terhubung' : 'WhatsApp Terputus' }}
        </span>
      </div>

      <div v-if="waStatus === 'disconnected'" style="margin-top: 1.5rem; text-align: center;">
        <div v-if="qrCodeData" class="qr-container">
          <p style="margin-bottom: 1rem;">Scan QR code di bawah menggunakan aplikasi WhatsApp Anda (Linked Devices /
            Perangkat Taut)</p>
          <img :src="qrCodeData" alt="WhatsApp QR Code" class="qr-image" />
        </div>
        <button v-else @click="requestQRCode" class="btn btn-primary" :disabled="loadingQr">
          {{ loadingQr ? 'Memuat QR...' : 'Tautkan Perangkat' }}
        </button>
      </div>

      <div v-else-if="waStatus === 'connected'" style="margin-top: 1.5rem;">
        <div class="webhook-info"
          style="padding: 1rem; background: rgba(16, 185, 129, 0.1); border-left: 4px solid #10b981; border-radius: 4px;">
          <h4 style="margin-bottom: 0.5rem; color: #10b981;">Siap Melayani Pelanggan!</h4>
          <p style="color: var(--text-secondary); font-size: 0.9rem;">
            Nomor ini sekarang secara otomatis dijawab oleh AI Chatbot.
          </p>
        </div>
      </div>
    </div>

    <!-- QRIS Settings -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <div class="flex justify-between items-center" style="margin-bottom: 1.5rem;">
        <h3>Pengaturan Pembayaran QRIS</h3>
        <label class="switch">
          <input type="checkbox" v-model="qrisEnabled">
          <span class="slider round"></span>
        </label>
      </div>
      <p style="color: var(--text-secondary); margin-bottom: 1rem;">
        Aktifkan pembayaran QRIS di halaman Kasir secara otomatis via Xendit. 
        Masukkan API Key dan Webhook Token akun Xendit Anda agar sistem bisa meng-generate invoice.
      </p>

      <div v-if="qrisEnabled" style="display: flex; flex-direction: column; gap: 1rem; margin-bottom: 1rem;">
        <input type="password" v-model="xenditApiKey" class="form-control"
          placeholder="Xendit Secret API Key (xnd_...)" />
        <input type="password" v-model="xenditWebhookToken" class="form-control"
          placeholder="Xendit Webhook Verification Token" />
      </div>

      <button class="btn btn-primary" @click="saveQrisSettings" :disabled="loadingQris">
        {{ loadingQris ? 'Menyimpan...' : 'Simpan Pengaturan QRIS' }}
      </button>
    </div>

    <!-- Laporan Harian Otomatis -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1rem;">Pengaturan Automasi & Laporan</h3>
      <p style="color: var(--text-secondary); margin-bottom: 1.5rem;">
        Fitur laporan harian kini lebih canggih. Anda bisa membuat banyak jadwal laporan otomatis (Harian, Mingguan, Bulanan) dan notifikasi stok rendah.
      </p>
      <router-link to="/automations" class="btn btn-primary" style="display: inline-block; text-decoration: none;">
        Kelola Automasi ➡️
      </router-link>
    </div>

    <!-- Staff Management -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">Pengaturan Pegawai (Kasir)</h3>
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <input type="text" placeholder="Username" v-model="staffForm.username" class="form-control" />
        <input type="email" placeholder="Email" v-model="staffForm.email" class="form-control" />
        <input type="password" placeholder="Password Sementara" v-model="staffForm.password" class="form-control" />
        <input type="text" placeholder="Nomor WA (contoh: 0812...)" v-model="staffForm.phoneNumber"
          class="form-control" />
        <select v-model="staffForm.role" class="form-control">
          <option value="kasir">Kasir</option>
          <option value="admin">Admin</option>
        </select>
        <button class="btn btn-primary" @click="handleAddStaff" :disabled="loadingStaff">
          {{ loadingStaff ? 'Menyimpan...' : 'Tambah Pegawai' }}
        </button>
      </div>
    </div>

    <!-- Pengaturan FAQ Bot -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <div class="flex justify-between items-center" style="margin-bottom: 1.5rem;">
        <h3 style="margin-bottom: 0;">Pengaturan FAQ Bot AI</h3>
        <button class="btn btn-secondary btn-sm" @click="generateFAQ" :disabled="loadingFaq">
          ✨ Generate Otomatis
        </button>
      </div>
      <p style="color: var(--text-secondary); margin-bottom: 1rem; font-size: 0.9rem;">
        Tambahkan daftar pertanyaan umum agar AI bisa langsung menjawab pelanggan yang menanyakan hal ini.
      </p>

      <div style="display: flex; flex-direction: column; gap: 1rem; margin-bottom: 1rem;">
        <div v-for="faq in faqs" :key="faq.id" style="background: rgba(255,255,255,0.05); padding: 1rem; border-radius: 8px;">
          <div style="font-weight: bold; margin-bottom: 0.3rem;">Q: {{ faq.question }}</div>
          <div style="color: var(--text-secondary); font-size: 0.9rem; margin-bottom: 0.5rem;">A: {{ faq.answer }}</div>
          <button class="btn btn-secondary btn-sm" style="color: #ef4444; border-color: #ef4444;" @click="deleteFAQ(faq.id)">Hapus</button>
        </div>
      </div>

      <div style="display: flex; flex-direction: column; gap: 0.5rem; border-top: 1px solid var(--border-color); padding-top: 1rem;">
        <input type="text" placeholder="Pertanyaan (Contoh: Jam Buka?)" v-model="newFaq.question" class="form-control" />
        <textarea placeholder="Jawaban (Contoh: Buka dari jam 8 pagi...)" v-model="newFaq.answer" class="form-control" rows="2"></textarea>
        <button class="btn btn-primary" @click="addFAQ" :disabled="!newFaq.question || !newFaq.answer">Tambah FAQ</button>
      </div>
    </div>

    <!-- Pengaturan Forwarder -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">Nomor WA Eskalasi (Forwarder)</h3>
      <p style="color: var(--text-secondary); margin-bottom: 1rem; font-size: 0.9rem;">
        Daftar nomor WhatsApp penerima notifikasi otomatis jika Bot gagal menjawab.
      </p>

      <div style="display: flex; flex-direction: column; gap: 0.5rem; margin-bottom: 1rem;">
        <div v-for="fwd in forwarders" :key="fwd.id" class="flex justify-between items-center" style="background: rgba(255,255,255,0.05); padding: 0.5rem 1rem; border-radius: 8px;">
          <span>{{ fwd.phone_number }}</span>
          <button class="btn btn-secondary btn-sm" style="color: #ef4444; border-color: transparent;" @click="deleteForwarder(fwd.id)">Hapus</button>
        </div>
      </div>

      <div class="flex gap-2">
        <input type="text" placeholder="Nomor WA (contoh: 62812...)" v-model="newForwarder" class="form-control" style="flex: 1;" />
        <button class="btn btn-primary" @click="addForwarder" :disabled="!newForwarder">Tambah</button>
      </div>
    </div>

    <!-- Custom Toast -->
    <div v-if="toast.visible" :class="['toast-notification', `toast-${toast.type}`]">
      {{ toast.message }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { api, API_BASE } from '../api'

const tenantId = ref(localStorage.getItem('tenant_id') || '')
const waStatus = ref<'checking' | 'connected' | 'disconnected'>('checking')
const qrCodeData = ref<string | null>(null)
const loadingQr = ref(false)
let pollingInterval: number | null = null

const qrisEnabled = ref(false)
const xenditApiKey = ref('')
const xenditWebhookToken = ref('')
const loadingQris = ref(false)

const reportEnabled = ref(false)
const reportTime = ref('07:00')


const toast = ref({ visible: false, message: '', type: 'success' })
const showToast = (message: string, type: 'success' | 'error' = 'success') => {
  toast.value = { visible: true, message, type }
  setTimeout(() => { toast.value.visible = false }, 3000)
}

// Profile Form
const profileForm = ref({
  username: '',
  phone_number: '',
  business_name: '',
  business_address: '',
  business_type: '',
  wa_number: '',
  old_password: '',
  new_password: '',
  logo_url: '',
})
const loadingAccount = ref(false)
const loadingStore = ref(false)
const uploadingLogo = ref(false)

const loadProfile = async () => {
  try {
    const data = await api.get('/api/profile')
    if (data.success && data.data) {
      profileForm.value = {
        username: data.data.username || '',
        phone_number: data.data.phone_number || '',
        business_name: data.data.business_name || '',
        business_address: data.data.business_address || '',
        business_type: data.data.business_type || '',
        wa_number: data.data.wa_number || '',
        old_password: '',
        new_password: '',
        logo_url: data.data.logo_url || '',
      }
    }
  } catch (err) {
    console.error("Failed to load profile", err)
  }
}

const saveAccount = async () => {
  loadingAccount.value = true
  try {
    const payload: any = {}
    if (profileForm.value.username) payload.username = profileForm.value.username
    if (profileForm.value.phone_number) payload.phone_number = profileForm.value.phone_number
    if (profileForm.value.new_password) {
      payload.old_password = profileForm.value.old_password
      payload.new_password = profileForm.value.new_password
    }
    const data = await api.put('/api/profile', payload)
    if (data.success) {
      showToast('Akun berhasil disimpan')
      profileForm.value.old_password = ''
      profileForm.value.new_password = ''
    } else {
      showToast(data.message || 'Gagal menyimpan akun', 'error')
    }
  } catch (err) {
    showToast('Kesalahan jaringan', 'error')
  } finally {
    loadingAccount.value = false
  }
}

const saveStore = async () => {
  loadingStore.value = true
  try {
    const payload: any = {}
    if (profileForm.value.business_name) payload.business_name = profileForm.value.business_name
    if (profileForm.value.business_address) payload.business_address = profileForm.value.business_address
    if (profileForm.value.business_type) payload.business_type = profileForm.value.business_type
    if (profileForm.value.wa_number) payload.wa_number = profileForm.value.wa_number
    const data = await api.put('/api/profile', payload)
    if (data.success) {
      if (payload.business_name) {
        localStorage.setItem('business_name', payload.business_name)
      }
      showToast('Profil toko berhasil disimpan')
      setTimeout(() => {
        window.location.reload()
      }, 1000)
    } else {
      showToast(data.message || 'Gagal menyimpan profil toko', 'error')
    }
  } catch (err) {
    showToast('Kesalahan jaringan', 'error')
  } finally {
    loadingStore.value = false
  }
}

const handleLogoUpload = async (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  uploadingLogo.value = true
  try {
    const formData = new FormData()
    formData.append('logo', file)
    const token = localStorage.getItem('access_token')
    const tid = localStorage.getItem('tenant_id')
    const res = await fetch(`${API_BASE}/api/profile/logo`, {
      method: 'POST',
      headers: { 'Authorization': 'Bearer ' + token, 'X-Tenant-ID': tid || '' },
      body: formData,
    })
    const data = await res.json()
    if (data.success) {
      profileForm.value.logo_url = data.data.logo_url
      showToast('Logo berhasil diupload')
    } else {
      showToast(data.message || 'Gagal upload logo', 'error')
    }
  } catch (err) {
    showToast('Kesalahan jaringan', 'error')
  } finally {
    uploadingLogo.value = false
  }
}

// Staff Form State
const staffForm = ref({ username: '', email: '', password: '', phoneNumber: '', role: 'kasir' })
const loadingStaff = ref(false)



const checkWaStatus = async () => {
  if (!tenantId.value) return

  try {
    const data = await api.wa('status', { tenant_id: tenantId.value })

    if (data.status === 'connected') {
      waStatus.value = 'connected'
      qrCodeData.value = null // clear QR if connected
      if (pollingInterval) clearInterval(pollingInterval)
    } else {
      waStatus.value = 'disconnected'
    }
  } catch (e) {
    console.error("Gagal memeriksa status WA:", e)
    waStatus.value = 'disconnected'
  }
}

const requestQRCode = async () => {
  loadingQr.value = true
  try {
    const data = await api.wa('qr', { tenant_id: tenantId.value })

    if (data.status === 'qr' && data.qr_code) {
      qrCodeData.value = data.qr_code
      // Start polling for status
    } else if (data.status === 'connected') {
      waStatus.value = 'connected'
      showToast("Sudah terhubung!")
    } else {
      showToast("Gagal memuat QR Code", "error")
    }
  } catch (e) {
    console.error("Error get QR:", e)
    showToast("Gagal menyambung ke server WA Gateway", "error")
  } finally {
    loadingQr.value = false
  }
}


const handleAddStaff = async () => {
  if (!staffForm.value.username || !staffForm.value.phoneNumber) {
    showToast("Isi data dengan lengkap!", "error")
    return
  }
  loadingStaff.value = true
  setTimeout(() => {
    loadingStaff.value = false
    showToast('Pegawai berhasil ditambahkan!')
    staffForm.value = { username: '', email: '', password: '', phoneNumber: '', role: 'kasir' }
  }, 1000)
}

const loadSettings = async () => {
  try {
    const data = await api.get('/api/umkm/settings')
    if (data.success && data.data) {
      qrisEnabled.value = data.data.qris_enabled || false
      xenditApiKey.value = data.data.xendit_api_key || ''
      xenditWebhookToken.value = data.data.xendit_webhook_token || ''
      reportEnabled.value = data.data.report_enabled || false
      reportTime.value = data.data.report_time || '07:00'
    }
  } catch (err) {
    console.error("Gagal load settings", err)
  }
}

const saveQrisSettings = async () => {
  loadingQris.value = true
  try {
    const payload = {
      qris_enabled: qrisEnabled.value,
      xendit_api_key: xenditApiKey.value,
      xendit_webhook_token: xenditWebhookToken.value,
      report_enabled: reportEnabled.value,
      report_time: reportTime.value
    }
    const data = await api.put('/api/umkm/settings', payload)
    if (data.success) {
      showToast('Pengaturan QRIS berhasil disimpan')
    } else {
      showToast(data.message || 'Gagal menyimpan', 'error')
    }
  } catch (err) {
    showToast('Kesalahan jaringan', 'error')
  } finally {
    loadingQris.value = false
  }
}



// FAQ Management
const faqs = ref<any[]>([])
const newFaq = ref({ question: '', answer: '' })
const loadingFaq = ref(false)

const loadFaqs = async () => {
  try {
    const res = await api.get('/api/umkm/faqs')
    if (res.success) faqs.value = res.data || []
  } catch (e) {
    console.error(e)
  }
}

const addFAQ = async () => {
  try {
    const res = await api.post('/api/umkm/faqs', newFaq.value)
    if (res.success) {
      showToast('FAQ berhasil ditambahkan')
      newFaq.value = { question: '', answer: '' }
      loadFaqs()
    } else {
      showToast('Gagal menambahkan FAQ', 'error')
    }
  } catch (e) {
    showToast('Network error', 'error')
  }
}

const deleteFAQ = async (id: string) => {
  const res = await api.del(`/api/umkm/faqs?id=${id}`)
  if (res.success) loadFaqs()
}

const generateFAQ = async () => {
  loadingFaq.value = true
  showToast('Meminta AI membuatkan draf FAQ...')
  const res = await api.post('/api/umkm/faqs/generate')
  loadingFaq.value = false
  if (res.success) {
    showToast('FAQ berhasil di-generate')
    loadFaqs()
  } else {
    showToast('Gagal generate FAQ', 'error')
  }
}

// Forwarder Management
const forwarders = ref<any[]>([])
const newForwarder = ref('')

const loadForwarders = async () => {
  const res = await api.get('/api/umkm/forwarders')
  if (res.success) forwarders.value = res.data || []
}

const addForwarder = async () => {
  const res = await api.post('/api/umkm/forwarders', { phone_number: newForwarder.value })
  if (res.success) {
    showToast('Nomor forwarder ditambahkan')
    newForwarder.value = ''
    loadForwarders()
  } else {
    showToast('Gagal menambahkan nomor', 'error')
  }
}

const deleteForwarder = async (id: string) => {
  const res = await api.del(`/api/umkm/forwarders?id=${id}`)
  if (res.success) loadForwarders()
}

onMounted(() => {
  checkWaStatus()
  loadSettings()
  loadProfile()
  loadFaqs()
  loadForwarders()
})

onUnmounted(() => {
  if (pollingInterval) clearInterval(pollingInterval)
})
</script>

<style scoped>
.form-control {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
}

.toast-notification {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  padding: 1rem 1.5rem;
  border-radius: 8px;
  color: #fff;
  font-weight: 500;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  z-index: 9999;
  animation: slideIn 0.3s ease-out;
}

.toast-success {
  background-color: #10b981;
}

.toast-error {
  background: #ef4444;
  color: white;
}

/* Switch styling */
.switch {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 24px;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  transition: .4s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 4px;
  bottom: 4px;
  background-color: white;
  transition: .4s;
}

input:checked+.slider {
  background-color: #10b981;
}

input:checked+.slider:before {
  transform: translateX(26px);
}

.slider.round {
  border-radius: 24px;
}

.slider.round:before {
  border-radius: 50%;
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }

  to {
    transform: translateX(0);
    opacity: 1;
  }
}

.wa-status-box {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 9999px;
  background: var(--bg-tertiary);
  font-weight: 500;
  font-size: 0.875rem;
}

.wa-status-box.connected {
  color: #10b981;
  background: rgba(16, 185, 129, 0.1);
}

.wa-status-box.disconnected {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}

.wa-status-box.checking {
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.1);
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
}

.qr-container {
  background: white;
  padding: 1.5rem;
  border-radius: 8px;
  display: inline-block;
}

.qr-image {
  width: 256px;
  height: 256px;
  margin: 0 auto;
}
</style>
