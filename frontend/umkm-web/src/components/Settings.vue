<template>
  <div class="settings-page">
    <div class="header-actions flex items-center justify-between" style="margin-bottom: 2rem;">
      <div>
        <h2>Pengaturan</h2>
        <p>Kelola akun, profil toko, dan integrasi</p>
      </div>
    </div>

    <!-- Pengaturan Akun -->
    <div class="surface-card animate-fade-in" style="max-width: 600px; padding: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">Pengaturan Akun</h3>
      <div style="display: flex; flex-direction: column; gap: 1rem;">
        <div>
          <label class="form-label">Username</label>
          <input type="text" placeholder="Username" v-model="profileForm.username" class="form-control" />
        </div>
        <div>
          <label class="form-label">Nomor HP</label>
          <input type="text" placeholder="Nomor HP" v-model="profileForm.phone_number" class="form-control" />
        </div>
        <div class="divider" style="border-top: 1px solid var(--border-color); margin: 0.5rem 0;"></div>
        <h4 style="margin-bottom: 0.25rem; font-size: 1rem; color: var(--text-primary);">Ganti Password</h4>
        <p style="font-size: 0.85rem; color: var(--text-secondary); margin-top: 0;">Kosongkan jika tidak ingin mengubah password</p>
        <div>
          <label class="form-label">Password Saat Ini</label>
          <input type="password" placeholder="Password Saat Ini" v-model="profileForm.old_password" class="form-control" />
        </div>
        <div>
          <label class="form-label">Password Baru</label>
          <input type="password" placeholder="Password Baru" v-model="profileForm.new_password" class="form-control" />
        </div>
        <button class="btn btn-primary" @click="saveAccount" :disabled="loadingAccount">
          {{ loadingAccount ? 'Menyimpan...' : 'Simpan Akun' }}
        </button>
      </div>
    </div>

    <!-- Profil Toko -->
    <div class="surface-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">Profil Toko</h3>

      <div class="profile-logo-section" style="margin-bottom: 1.5rem; text-align: center;">
        <div class="logo-preview"
          style="width: 80px; height: 80px; border-radius: 50%; overflow: hidden; margin: 0 auto 0.5rem; border: 2px solid var(--border-color);">
          <img v-if="profileForm.logo_url" :src="profileForm.logo_url" alt="Logo"
            style="width: 100%; height: 100%; object-fit: cover;" />
          <span v-else style="font-size: 2rem; color: var(--text-secondary);">S</span>
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
        <input type="text" v-model="xenditMerchantID" class="form-control"
          placeholder="Xendit Merchant ID (opsional, untuk routing pembayaran ke akun Anda)" />
        <input type="password" v-model="xenditWebhookToken" class="form-control"
          placeholder="Xendit Webhook Verification Token" />
      </div>

      <button class="btn btn-primary" @click="saveQrisSettings" :disabled="loadingQris">
        {{ loadingQris ? 'Menyimpan...' : 'Simpan Pengaturan QRIS' }}
      </button>
    </div>

    <!-- Quota Usage (F025, Task 2.9) — superadmin only -->
    <div v-if="isSuperadmin" class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1.5rem;">📊 Quota Usage (Superadmin)</h3>
      <p style="color: var(--text-secondary); margin-bottom: 1rem; font-size: 0.9rem;">
        Inspect quota counters and plan limits for any tenant. Endpoint: <code>/api/superadmin/billing/admin/quota/{tenant_id}</code>
      </p>

      <div class="flex gap-2" style="margin-bottom: 1rem;">
        <input
          v-model="quotaTenantInput"
          placeholder="Tenant ID (UUID)"
          class="form-control"
          style="flex: 1;"
        />
        <button class="btn btn-primary" @click="loadQuota" :disabled="loadingQuota || !quotaTenantInput">
          {{ loadingQuota ? 'Memuat...' : 'Muat' }}
        </button>
      </div>

      <div v-if="quotaError" style="background: rgba(239,68,68,0.1); color: #ef4444; padding: 0.75rem 1rem; border-radius: 6px; margin-bottom: 1rem; font-size: 0.9rem;">
        {{ quotaError }}
      </div>

      <div v-if="quota" class="quota-section">
        <p style="margin-bottom: 0.25rem;">
          Plan: <strong>{{ quota.tier }}</strong> ({{ quota.plan_name }})
        </p>
        <p style="font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 1rem;">
          Tenant: <code>{{ quota.tenant_id }}</code> · Period: {{ quota.period }}
        </p>

        <div v-for="row in quotaRows" :key="row.key" class="quota-bar" style="margin-bottom: 0.75rem;">
          <div class="quota-bar-label" style="display: flex; justify-content: space-between; font-size: 0.85rem; margin-bottom: 0.25rem;">
            <span>{{ row.label }}</span>
            <span>{{ row.used }} / {{ row.limitText }}</span>
          </div>
          <div class="quota-bar-track" style="background: var(--bg-tertiary); height: 8px; border-radius: 4px; overflow: hidden;">
            <div
              class="quota-bar-fill"
              :class="row.percent >= 80 ? 'quota-bar-warn' : ''"
              :style="{ width: row.percent + '%' }"
            ></div>
          </div>
        </div>
      </div>
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

    <!-- Staff List -->
    <div class="glass-card animate-fade-in" style="max-width: 600px; padding: 2rem; margin-top: 2rem;">
      <h3 style="margin-bottom: 1.5rem; display: flex; justify-content: space-between; align-items: center;">
        Daftar Pegawai
        <button class="btn btn-secondary" style="padding: 0.5rem 1rem; font-size: 0.8rem;" @click="fetchStaffList" :disabled="loadingStaffList">
          {{ loadingStaffList ? '...' : 'Refresh' }}
        </button>
      </h3>

      <div v-if="loadingStaffList" style="text-align: center; padding: 2rem;">Loading...</div>
      <div v-else-if="staffList.length === 0" style="text-align: center; padding: 2rem; opacity: 0.7;">Belum ada data pegawai.</div>

      <div v-else style="display: flex; flex-direction: column; gap: 1rem;">
        <div v-for="staff in staffList" :key="staff.id" style="border: 1px solid rgba(255,255,255,0.1); border-radius: 8px; padding: 1rem;">
          <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0.5rem;">
            <div>
              <strong>{{ staff.username }}</strong>
              <span style="font-size: 0.8rem; padding: 0.2rem 0.5rem; background: rgba(var(--primary-color-rgb), 0.2); border-radius: 4px; margin-left: 0.5rem;">
                {{ staff.role }}
              </span>
            </div>
            <div>
              <button class="btn btn-secondary" style="padding: 0.3rem 0.6rem; font-size: 0.8rem; margin-right: 0.5rem;" @click="openEditStaffModal(staff)">Edit</button>
              <button class="btn btn-danger" style="padding: 0.3rem 0.6rem; font-size: 0.8rem;" @click="handleDeleteStaff(staff.id)">Hapus</button>
            </div>
          </div>
          <div style="font-size: 0.9rem; opacity: 0.8;">
            <div v-if="staff.email">Email: {{ staff.email }}</div>
            <div v-if="staff.phone_number">WA: {{ staff.phone_number }}</div>
          </div>
        </div>
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
          <template v-if="editingFaqId === faq.id">
            <input type="text" v-model="editFaqForm.question" class="form-control" style="margin-bottom: 0.5rem;" />
            <textarea v-model="editFaqForm.answer" class="form-control" rows="2" style="margin-bottom: 0.5rem;"></textarea>
            <div style="display: flex; gap: 0.5rem;">
              <button class="btn btn-primary btn-sm" @click="saveEditFAQ(faq.id)" :disabled="!editFaqForm.question || !editFaqForm.answer">Simpan</button>
              <button class="btn btn-secondary btn-sm" @click="cancelEditFAQ">Batal</button>
            </div>
          </template>
          <template v-else>
            <div style="font-weight: bold; margin-bottom: 0.3rem;">Q: {{ faq.question }}</div>
            <div style="color: var(--text-secondary); font-size: 0.9rem; margin-bottom: 0.5rem;">A: {{ faq.answer }}</div>
            <div style="display: flex; gap: 0.5rem;">
              <button class="btn btn-secondary btn-sm" @click="startEditFAQ(faq)">✏️ Edit</button>
              <button class="btn btn-secondary btn-sm" style="color: #ef4444; border-color: #ef4444;" @click="deleteFAQ(faq.id)">Hapus</button>
            </div>
          </template>
        </div>
      </div>

      <div style="display: flex; flex-direction: column; gap: 0.5rem; border-top: 1px solid var(--border-color); padding-top: 1rem;">
        <input type="text" placeholder="Pertanyaan (Contoh: Jam Buka?)" v-model="newFaq.question" class="form-control" />
        <textarea placeholder="Jawaban (Contoh: Buka dari jam 8 pagi...)" v-model="newFaq.answer" class="form-control" rows="2"></textarea>
        <button class="btn btn-primary" @click="addFAQ" :disabled="!newFaq.question || !newFaq.answer">Tambah FAQ</button>
      </div>
    </div>

    <!-- Edit Staff Modal -->
    <div v-if="showEditStaffModal" class="modal-overlay" @click.self="showEditStaffModal = false">
      <div class="modal-content glass-card animate-fade-in" style="max-width: 400px; width: 100%;">
        <h3>Edit Pegawai</h3>
        <div style="display: flex; flex-direction: column; gap: 1rem; margin-top: 1.5rem;">
          <div>
            <label style="font-size: 0.85rem; opacity: 0.8; margin-bottom: 0.3rem; display: block;">Username</label>
            <input type="text" v-model="editStaffForm.username" class="form-control" />
          </div>
          <div>
            <label style="font-size: 0.85rem; opacity: 0.8; margin-bottom: 0.3rem; display: block;">No. WA</label>
            <input type="text" v-model="editStaffForm.phone_number" class="form-control" />
          </div>
          <div>
            <label style="font-size: 0.85rem; opacity: 0.8; margin-bottom: 0.3rem; display: block;">Reset Password (opsional)</label>
            <input type="password" placeholder="Biarkan kosong jika tidak diubah" v-model="editStaffForm.password" class="form-control" />
          </div>

          <div style="display: flex; justify-content: flex-end; gap: 1rem; margin-top: 1rem;">
            <button class="btn btn-secondary" @click="showEditStaffModal = false">Batal</button>
            <button class="btn btn-primary" @click="handleUpdateStaff" :disabled="loadingUpdateStaff">
              {{ loadingUpdateStaff ? 'Menyimpan...' : 'Simpan' }}
            </button>
          </div>
        </div>
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
    <Teleport to="body">
      <div v-if="toast.visible" :class="['toast-notification', `toast-${toast.type}`, { fading: toast.fading }]">
        {{ toast.message }}
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { api, API_BASE, getQuotaUsage, type QuotaUsage } from '../api'
import { authApi } from '../api'

const qrisEnabled = ref(false)
const xenditApiKey = ref('')
const xenditWebhookToken = ref('')
const xenditMerchantID = ref('')
const loadingQris = ref(false)

const reportEnabled = ref(false)
const reportTime = ref('07:00')

let toastTimer: ReturnType<typeof setTimeout> | null = null
const toast = ref({ visible: false, message: '', type: 'success', fading: false })
const showToast = (message: string, type: 'success' | 'error' = 'success') => {
  if (toastTimer) clearTimeout(toastTimer)
  toast.value = { visible: true, message, type, fading: false }
  toastTimer = setTimeout(() => {
    toast.value.fading = true            // trigger fade-out
    setTimeout(() => { toast.value.visible = false }, 350)
  }, 3000)
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
const staffList = ref<any[]>([])
const loadingStaffList = ref(false)
const showEditStaffModal = ref(false)
const editStaffForm = ref({ id: '', username: '', phone_number: '', password: '' })
const loadingUpdateStaff = ref(false)
const userRole = computed(() => localStorage.getItem('role') || '')

const fetchStaffList = async () => {
  if (userRole.value !== 'owner') return;
  loadingStaffList.value = true;
  try {
    const res = await authApi.getStaffList();
    if (res.success && res.data) {
      staffList.value = res.data;
    }
  } catch (error) {
    console.error('Failed to fetch staff:', error);
  } finally {
    loadingStaffList.value = false;
  }
}

const openEditStaffModal = (staff: any) => {
  editStaffForm.value = {
    id: staff.id,
    username: staff.username,
    phone_number: staff.phone_number || '',
    password: ''
  };
  showEditStaffModal.value = true;
}

const handleUpdateStaff = async () => {
  if (!editStaffForm.value.username) {
    showToast('Username tidak boleh kosong', 'error');
    return;
  }

  loadingUpdateStaff.value = true;
  try {
    const res = await authApi.updateStaff(editStaffForm.value);
    if (res.success) {
      showToast('Pegawai berhasil diperbarui', 'success');
      showEditStaffModal.value = false;
      fetchStaffList();
    } else {
      showToast(res.message || 'Gagal memperbarui pegawai', 'error');
    }
  } catch (error: any) {
    showToast(error.message || 'Terjadi kesalahan jaringan', 'error');
  } finally {
    loadingUpdateStaff.value = false;
  }
}

const handleDeleteStaff = async (id: string) => {
  if (!confirm('Yakin ingin menghapus pegawai ini?')) return;

  try {
    const res = await authApi.deleteStaff(id);
    if (res.success) {
      showToast('Pegawai berhasil dihapus', 'success');
      fetchStaffList();
    } else {
      showToast(res.message || 'Gagal menghapus pegawai', 'error');
    }
  } catch (error: any) {
    showToast(error.message || 'Terjadi kesalahan jaringan', 'error');
  }
}

const handleAddStaff = async () => {
  if (!staffForm.value.username || !staffForm.value.phoneNumber) {
    showToast("Isi username dan nomor HP!", "error")
    return
  }
  if (!staffForm.value.password) {
    showToast("Password wajib diisi!", "error")
    return
  }
  loadingStaff.value = true
  try {
    const data = await api.post('/auth/add-staff', {
      username: staffForm.value.username,
      email: staffForm.value.email || '',
      password: staffForm.value.password,
      role: staffForm.value.role || 'kasir',
      phoneNumber: staffForm.value.phoneNumber,
    })
    if (data?.success) {
      showToast('Pegawai berhasil ditambahkan')
      staffForm.value = { username: '', email: '', password: '', phoneNumber: '', role: 'kasir' }
      fetchStaffList()
    } else {
      showToast(data?.message || 'Gagal menambahkan pegawai', 'error')
    }
  } catch (err) {
    console.error('Add staff error:', err)
    showToast('Kesalahan jaringan', 'error')
  } finally {
    loadingStaff.value = false
  }
}

const loadSettings = async () => {
  try {
    const data = await api.get('/api/umkm/settings')
    if (data.success && data.data) {
      qrisEnabled.value = data.data.qris_enabled || false
      xenditApiKey.value = data.data.xendit_api_key || ''
      xenditMerchantID.value = data.data.xendit_merchant_id || ''
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
      xendit_merchant_id: xenditMerchantID.value,
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

const editingFaqId = ref<string | null>(null)
const editFaqForm = ref({ question: '', answer: '' })

const startEditFAQ = (faq: any) => {
  editingFaqId.value = faq.id
  editFaqForm.value = { question: faq.question, answer: faq.answer }
}

const cancelEditFAQ = () => {
  editingFaqId.value = null
  editFaqForm.value = { question: '', answer: '' }
}

const saveEditFAQ = async (id: string) => {
  try {
    const res = await api.put('/api/umkm/faqs', {
      id,
      question: editFaqForm.value.question,
      answer: editFaqForm.value.answer,
    })
    if (res.success) {
      showToast('FAQ berhasil diupdate')
      editingFaqId.value = null
      editFaqForm.value = { question: '', answer: '' }
      loadFaqs()
    } else {
      showToast(res.message || 'Gagal update FAQ', 'error')
    }
  } catch (e) {
    showToast('Network error', 'error')
  }
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

// Quota Usage (F025, Task 2.9) — superadmin only
const isSuperadmin = computed(() => (localStorage.getItem('role') || '') === 'superadmin')
const quota = ref<QuotaUsage | null>(null)
const quotaError = ref('')
const quotaTenantInput = ref(localStorage.getItem('tenant_id') || '')
const loadingQuota = ref(false)

function quotaPercent(used: number, limit: number): number {
  if (!limit || limit < 0) return 0
  return Math.min(100, Math.round((used / limit) * 100))
}

function limitDisplay(limit: number): string {
  if (limit === 0) return 'off'
  if (limit < 0) return '∞'
  return String(limit)
}

const quotaRows = computed(() => {
  if (!quota.value) return []
  const limits = quota.value.limits
  const usedBy: Record<string, number> = {}
  for (const u of quota.value.usage || []) {
    usedBy[u.feature] = (usedBy[u.feature] || 0) + (u.used || 0)
  }
  return [
    { key: 'ai_text', label: 'AI Text Requests', used: usedBy.ai_text || 0, limitText: limitDisplay(limits.max_ai_text), percent: quotaPercent(usedBy.ai_text || 0, limits.max_ai_text) },
    { key: 'ai_vision', label: 'AI Vision (Image→Text)', used: usedBy.ai_vision || 0, limitText: limitDisplay(limits.max_ai_vision), percent: quotaPercent(usedBy.ai_vision || 0, limits.max_ai_vision) },
    { key: 'ai_audio_stt', label: 'AI Audio (STT minutes)', used: usedBy.ai_audio_stt || 0, limitText: limitDisplay(limits.max_ai_audio_minutes), percent: quotaPercent(usedBy.ai_audio_stt || 0, limits.max_ai_audio_minutes) },
    { key: 'image_gen', label: 'AI Image Generation', used: usedBy.image_gen || 0, limitText: limitDisplay(limits.max_image_gen), percent: quotaPercent(usedBy.image_gen || 0, limits.max_image_gen) },
    { key: 'chatbot_messages', label: 'Chatbot Messages', used: usedBy.chatbot_messages || 0, limitText: '—', percent: 0 },
    { key: 'transactions', label: 'Transactions (period)', used: 0, limitText: limitDisplay(limits.max_transactions), percent: 0 },
  ]
})

const loadQuota = async () => {
  const tid = quotaTenantInput.value.trim()
  if (!tid) {
    quotaError.value = 'Tenant ID required'
    return
  }
  loadingQuota.value = true
  quotaError.value = ''
  quota.value = null
  try {
    const data = await getQuotaUsage(tid)
    if (data) {
      quota.value = data
    } else {
      quotaError.value = 'Gagal memuat quota (403/404 atau respons tidak valid).'
    }
  } catch (e: any) {
    quotaError.value = e?.message || 'Kesalahan jaringan'
  } finally {
    loadingQuota.value = false
  }
}

onMounted(() => {
  loadSettings()
  loadProfile()
  loadFaqs()
  loadForwarders()
  fetchStaffList()
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

/* Toast styles are in main.css — do not duplicate here */

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

.quota-bar-fill {
  background: #4f46e5;
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.quota-bar-warn {
  background: #f59e0b;
}
</style>