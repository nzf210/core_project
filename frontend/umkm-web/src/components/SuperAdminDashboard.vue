<template>
  <div class="superadmin-dashboard">
    <div style="margin-bottom: 2rem; display: flex; justify-content: space-between; align-items: center;">
      <div>
        <h2>Super Admin Dashboard</h2>
        <p class="text-muted">Kelola WhatsApp Verifier & pantau semua tenant</p>
      </div>
      <div>
        <button class="btn btn-primary" style="padding: 0.5rem 1rem;" @click="$router.push('/superadmin/n8n')">
          Buka N8n Workflow
        </button>
      </div>
    </div>

    <div class="dashboard-grid">
      <!-- My Profile Card -->
      <div class="card glass-card">
        <div class="card-header">
          <h3>Profil Saya</h3>
        </div>
        <div class="card-body">
          <p class="text-muted" style="margin-bottom: 1rem;">
            <strong>{{ myProfile.username }}</strong> &mdash; Super Admin
          </p>
          <p class="text-muted" style="font-size: 0.85rem;">
            {{ myProfile.phone_number || 'No phone' }}
          </p>
          <button class="btn btn-secondary" style="margin-top: 0.75rem; padding: 0.35rem 0.75rem; font-size: 0.8rem;"
            @click="showMyProfile = true">Edit Profil</button>
        </div>
      </div>

      <!-- WA Verifier Card -->
      <div class="card glass-card" style="grid-column: span 2;">
        <div class="card-header">
          <h3>WhatsApp Verifier</h3>
          <span :class="['status-badge', verifierStatus === 'connected' ? 'status-connected' : 'status-disconnected']">
            {{ verifierStatus === 'connected' ? 'Terhubung' : 'Tidak Terhubung' }}
          </span>
        </div>

        <div class="card-body">
          <div v-if="verifierStatus === 'disconnected' && !qrCode" class="verifier-actions">
            <p class="text-muted">WhatsApp Verifier belum terhubung. Hubungkan untuk mengaktifkan verifikasi OTP via
              WhatsApp.</p>
            <button class="btn btn-primary" @click="connectVerifier" :disabled="loadingQR">
              {{ loadingQR ? 'Menghubungkan...' : 'Hubungkan WhatsApp' }}
            </button>
          </div>

          <div v-if="qrCode" class="qr-section">
            <img :src="qrCode" alt="QR Code" class="qr-image" />
            <p class="text-muted" style="margin-top: 1rem;">Scan QR code ini menggunakan WhatsApp di HP Anda</p>
            <div class="qr-actions">
              <button class="btn btn-primary" @click="checkVerifierStatus" :disabled="checkingStatus">
                {{ checkingStatus ? 'Memeriksa...' : 'Cek Status' }}
              </button>
              <button class="btn"
                style="background: transparent; color: var(--text-secondary); border: 1px solid var(--border-color);"
                @click="qrCode = ''">
                Batal
              </button>
            </div>
          </div>

          <div v-if="verifierStatus === 'connected' && verifierJID" class="verifier-info">
            <div class="info-row">
              <span class="info-label">Nomor</span>
              <span class="info-value">{{ verifierJID }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">Status</span>
              <span class="info-value" style="color: #10b981;">Online</span>
            </div>
            <div style="margin-top: 1.5rem;">
              <button class="btn btn-danger" @click="disconnectVerifier" :disabled="disconnecting"
                style="background: #ef4444; color: white; border: none;">
                {{ disconnecting ? 'Memutuskan...' : 'Putuskan WhatsApp' }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Tenant Overview Card -->
      <div class="card glass-card">
        <div class="card-header">
          <h3>Total Tenant</h3>
        </div>
        <div class="card-body">
          <div class="stat-number">{{ tenants.length }}</div>
          <p class="text-muted">tenant terdaftar</p>
        </div>
      </div>

      <!-- Plan Distribution -->
      <div class="card glass-card">
        <div class="card-header">
          <h3>Paket</h3>
        </div>
        <div class="card-body">
          <div class="plan-stats">
            <div class="plan-stat"><span class="badge badge-free">FREE</span> {{ planCounts.free }}</div>
            <div class="plan-stat"><span class="badge badge-lite">LITE</span> {{ planCounts.lite }}</div>
            <div class="plan-stat"><span class="badge badge-pro">PRO</span> {{ planCounts.pro }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tenant Table -->
    <div class="card glass-card" style="margin-top: 1.5rem;">
      <div class="card-header">
        <h3>Daftar Tenant</h3>
        <div>
          <button class="btn btn-primary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem; margin-right: 0.5rem;" @click="showAddTenant = true">
            Tambah Tenant Baru
          </button>
          <button class="btn" style="background: rgba(168, 85, 247, 0.15); color: #a855f7; border: 1px solid rgba(168, 85, 247, 0.3); padding: 0.35rem 0.75rem; font-size: 0.8rem; margin-right: 0.5rem;" @click="openPlanEditor">
            Kelola Paket
          </button>
          <button class="btn btn-secondary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem;" @click="fetchTenants"
            :disabled="loadingTenants">
            {{ loadingTenants ? '...' : 'Refresh' }}
          </button>
        </div>
      </div>
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>Nama Usaha</th>
              <th>Owner</th>
              <th>Phone</th>
              <th>Users</th>
              <th>Paket</th>
              <th>Terdaftar</th>
              <th>Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in tenants" :key="t.id">
              <td><strong>{{ t.name }}</strong></td>
              <td>{{ t.owner_username || '-' }}</td>
              <td>{{ t.owner_phone || '-' }}</td>
              <td>{{ t.user_count ?? 0 }}</td>
              <td><span :class="['badge', 'badge-' + t.plan]">{{ t.plan?.toUpperCase() }}</span></td>
              <td>{{ t.created_at ? new Date(t.created_at).toLocaleDateString('id-ID') : '-' }}</td>
              <td>
                <button class="btn-edit" @click="openEditProfile(t)" title="Edit profil tenant">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M17 3a2.828 2.828 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z" />
                  </svg>
                </button>
                <button class="btn-delete" @click="confirmDelete(t)" title="Hapus tenant ini">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none"
                    stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6" />
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                  </svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Tambah Tenant Baru -->
    <div v-if="showAddTenant" class="modal-overlay" @click.self="showAddTenant = false">
      <div class="modal-card" style="max-width: 450px;">
        <h3 style="margin: 0 0 1.25rem 0;">Tambah Tenant Baru</h3>
        
        <div class="form-group">
          <label>Nama Usaha (Tenant)</label>
          <input v-model="formData.name" type="text" class="form-control" placeholder="cth: Toko Sembako Budi" />
        </div>

        <div class="form-group">
          <label>Username Pemilik</label>
          <input v-model="formData.username" @input="formData.username = formData.username.replace(/ /g, '_').toLowerCase()" type="text" class="form-control" placeholder="cth: budi_sembako" />
        </div>

        <div class="form-group">
          <label>Email Pemilik</label>
          <input v-model="formData.email" type="email" class="form-control" placeholder="cth: budi@contoh.com" />
        </div>

        <div class="form-group">
          <label>Nomor WhatsApp (Aktif)</label>
          <input v-model="formData.phone_number" type="text" class="form-control" placeholder="cth: 081234567890" />
        </div>

        <div class="form-group">
          <label>Subdomain (Opsional)</label>
          <input v-model="formData.subdomain" type="text" class="form-control" placeholder="cth: sembako.saas.com" />
        </div>

        <div class="form-group">
          <label>Custom Domain (Opsional)</label>
          <input v-model="formData.custom_domain" type="text" class="form-control" placeholder="cth: www.tokosembako.com" />
        </div>

        <div class="form-group">
          <label>Pilih Paket Langganan</label>
          <select v-model="formData.plan" class="form-control">
            <option value="free">Free Tier (Rp 0)</option>
            <option value="lite">Lite (Rp 150.000)</option>
            <option value="pro">Pro (Rp 450.000)</option>
          </select>
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="closeAddModal">Batal</button>
          <button class="btn btn-primary" @click="saveNewTenant">Simpan</button>
        </div>
      </div>
    </div>

    <!-- My Profile Modal -->
    <div v-if="showMyProfile" class="modal-overlay" @click.self="showMyProfile = false">
      <div class="modal-card" style="max-width: 440px;">
        <h3 style="margin: 0 0 1.25rem 0;">Edit Profil Saya</h3>

        <div class="form-group"><label>Username</label><input v-model="myProfile.username" class="form-control" /></div>
        <div class="form-group"><label>Nomor HP</label><input v-model="myProfile.phone_number" class="form-control"
            placeholder="0812..." /></div>
        <div class="form-group"><label>Password Lama</label><input v-model="myProfile.old_password" type="password"
            class="form-control" placeholder="(untuk ganti password)" /></div>
        <div class="form-group"><label>Password Baru</label><input v-model="myProfile.new_password" type="password"
            class="form-control" placeholder="(kosongkan jika tidak diganti)" /></div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="showMyProfile = false" :disabled="savingMyProfile">Batal</button>
          <button class="btn btn-primary" @click="saveMyProfile" :disabled="savingMyProfile">
            {{ savingMyProfile ? 'Menyimpan...' : 'Simpan' }}
          </button>
        </div>
        <div v-if="myProfileError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">
          {{ myProfileError }}</div>
      </div>
    </div>

    <!-- Edit Profile Modal -->
    <div v-if="editTarget" class="modal-overlay" @click.self="editTarget = null">
      <div class="modal-card" style="max-width: 540px; max-height: 85vh; overflow-y: auto;">
        <h3 style="margin: 0 0 0.25rem 0;">Edit Profil Tenant</h3>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.25rem;">
          {{ editTarget.name }} ({{ editTarget.owner_username || 'no owner' }})
        </p>

        <div class="logo-upload-section">
          <div class="logo-preview">
            <img v-if="editLogoPreview" :src="editLogoPreview" alt="Logo preview" />
            <img v-else-if="editForm.logo_url" :src="API_BASE + editForm.logo_url + '?t=' + Date.now()"
              alt="Current logo" />
            <div v-else class="logo-placeholder">No Logo</div>
          </div>
          <label class="file-input-label">
            <input type="file" accept="image/png,image/jpeg,image/webp" @change="onLogoFileChange"
              style="display:none" />
            <span class="btn btn-secondary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem; cursor: pointer;">Pilih
              Logo</span>
          </label>
          <button v-if="editLogoFile" class="btn"
            style="background: transparent; color: var(--accent-primary); border: 1px solid var(--border-color); padding: 0.35rem 0.75rem; font-size: 0.8rem;"
            @click="uploadLogo" :disabled="uploadingLogo">
            {{ uploadingLogo ? 'Uploading...' : 'Upload Logo' }}
          </button>
        </div>

        <div class="form-group"><label>Nama Toko</label><input v-model="editForm.business_name" class="form-control"
            placeholder="Nama usaha/toko" /></div>
        <div class="form-group"><label>Nama Usaha (Tenant)</label><input v-model="editForm.name" class="form-control" />
        </div>
        <div class="form-group"><label>Subdomain</label><input v-model="editForm.subdomain" class="form-control" placeholder="opsional" /></div>
        <div class="form-group"><label>Custom Domain</label><input v-model="editForm.custom_domain" class="form-control" placeholder="opsional" /></div>
        <div class="form-group"><label>Nomor WA Toko (CS)</label><input v-model="editForm.wa_number" class="form-control"
            placeholder="0812..." /></div>
        <div class="form-group"><label>Nomor WA Owner (Login)</label><input v-model="editForm.owner_phone" class="form-control"
            placeholder="0812..." /></div>
        <div class="form-group"><label>Alamat Usaha</label><textarea v-model="editForm.business_address"
            class="form-control" rows="2" placeholder="Alamat lengkap" /></div>
        <div class="form-group"><label>Jenis Usaha</label>
          <select v-model="editForm.business_type" class="form-control">
            <option v-for="bt in businessTypes" :key="bt.id" :value="bt.id">{{ bt.name }}</option>
          </select>
        </div>
        <div class="form-group"><label>Paket</label>
          <select v-model="editForm.plan" class="form-control">
            <option value="free">FREE</option>
            <option value="lite">LITE</option>
            <option value="pro">PRO</option>
          </select>
        </div>
        <div class="form-group"><label>Reset Password Owner <span
              style="color: var(--text-secondary); font-weight: 400; font-size: 0.8rem;">(kosongkan jika tidak ingin
              diubah)</span></label>
          <input v-model="editForm.new_password" type="password" class="form-control" placeholder="Password baru" />
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="editTarget = null" :disabled="savingProfile">Batal</button>
          <button class="btn btn-primary" @click="saveProfile" :disabled="savingProfile">
            {{ savingProfile ? 'Menyimpan...' : 'Simpan' }}
          </button>
        </div>
        <div v-if="profileError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">{{ profileError }}
        </div>
      </div>
    </div>

    <!-- Modal Delete Confirmation -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget = null">
      <div class="modal-card">
        <h3 style="margin: 0 0 0.75rem 0;">Hapus Tenant?</h3>
        <p style="margin-bottom: 0.5rem;">Anda akan menghapus:</p>
        <p style="margin-bottom: 1.5rem; font-weight: 600;">{{ deleteTarget.name }} ({{ deleteTarget.owner_username ||
          'no owner' }})</p>
        <p style="margin-bottom: 1.5rem; color: #ef4444; font-size: 0.85rem;">
          ⚠️ Semua data: users, produk, jurnal, transaksi, dan data tenant akan dihapus permanen.
        </p>
        <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
          <button class="btn btn-secondary" @click="deleteTarget = null" :disabled="deleting">Batal</button>
          <button class="btn" style="background: #ef4444; color: white; border: none;" @click="executeDelete"
            :disabled="deleting">
            {{ deleting ? 'Menghapus...' : 'Hapus Permanen' }}
          </button>
        </div>
        <div v-if="deleteError" style="margin-top: 1rem; color: #ef4444; font-size: 0.85rem;">{{ deleteError }}</div>
      </div>
    </div>

    <!-- Modal Paket Langganan (Edit Harga) -->
    <div v-if="showPlanEditor" class="modal-overlay" @click.self="showPlanEditor = false">
      <div class="modal-card" style="max-width: 520px;">
        <h3 style="margin: 0 0 0.25rem 0;">Kelola Paket Langganan</h3>
        <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.5rem;">
          Ubah harga paket SaaS untuk semua tenant.
        </p>

        <div v-if="loadingPlans" style="text-align: center; padding: 2rem; color: var(--text-secondary);">
          Memuat...
        </div>

        <div v-else>
          <div v-for="plan in editablePlans" :key="plan.id" class="plan-editor-row">
            <div class="plan-editor-header">
              <div>
                <span class="badge" :class="['badge-' + plan.id]">{{ plan.name.toUpperCase() }}</span>
              </div>
              <div style="display: flex; gap: 0.5rem; align-items: center;">
                <label style="font-size: 0.75rem; color: var(--text-secondary);">Aktif</label>
                <label class="toggle-switch">
                  <input type="checkbox" v-model="plan.is_active" />
                  <span class="toggle-slider"></span>
                </label>
              </div>
            </div>
            <div class="plan-editor-fields">
              <div class="form-group">
                <label>Harga Bulanan (Rp)</label>
                <div style="display: flex; align-items: center; gap: 0.25rem;">
                  <span style="font-size: 0.85rem; color: var(--text-secondary);">Rp</span>
                  <input v-model.number="plan.price_monthly_display" type="number" class="form-control" min="0"
                    step="1000" style="width: 100%;" @input="syncPlanPrice(plan, 'monthly')" />
                </div>
                <small style="color: var(--text-secondary); font-size: 0.7rem;">
                  Dalam sen: {{ plan.price_monthly?.toLocaleString() }} sen
                </small>
              </div>
              <div class="form-group">
                <label>Harga Tahunan (Rp)</label>
                <div style="display: flex; align-items: center; gap: 0.25rem;">
                  <span style="font-size: 0.85rem; color: var(--text-secondary);">Rp</span>
                  <input v-model.number="plan.price_yearly_display" type="number" class="form-control" min="0"
                    step="1000" style="width: 100%;" @input="syncPlanPrice(plan, 'yearly')" />
                </div>
                <small style="color: var(--text-secondary); font-size: 0.7rem;">
                  Dalam sen: {{ plan.price_yearly?.toLocaleString() }} sen
                </small>
              </div>
            </div>
          </div>
        </div>

        <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
          <button class="btn btn-secondary" @click="showPlanEditor = false" :disabled="savingPlans">Tutup</button>
          <button class="btn btn-primary" @click="savePlanPrices" :disabled="savingPlans">
            {{ savingPlans ? 'Menyimpan...' : 'Simpan Semua' }}
          </button>
        </div>
        <div v-if="planError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">{{ planError }}</div>
      </div>
    </div>

    <!-- Toast -->
    <div v-if="toast.visible" :class="['toast-notification', `toast-${toast.type}`]">
      {{ toast.message }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { superadminApi } from '../superadminApi'
import { api, API_BASE } from '../api'

const verifierStatus = ref<'connected' | 'disconnected'>('disconnected')
const verifierJID = ref('')
const qrCode = ref('')
const loadingQR = ref(false)
const checkingStatus = ref(false)
const disconnecting = ref(false)
const tenants = ref<any[]>([])
const deleting = ref(false)
const deleteTarget = ref<any>(null)
const deleteError = ref('')
const loadingTenants = ref(false)

const showAddTenant = ref(false)
const formData = ref({
  name: '',
  username: '',
  password: 'Password123',
  email: '',
  phone_number: '',
  role: 'owner',
  plan: 'free',
  subdomain: '',
  custom_domain: ''
})

const editTarget = ref<any>(null)
const editForm = ref({
  tenant_id: '',
  name: '',
  business_name: '',
  wa_number: '',
  owner_phone: '',
  business_address: '',
  business_type: '',
  plan: 'free',
  new_password: '',
  logo_url: '',
  subdomain: '',
  custom_domain: ''
})
const editLogoFile = ref<File | null>(null)
const editLogoPreview = ref('')
const uploadingLogo = ref(false)
const savingProfile = ref(false)
const profileError = ref('')

const businessTypes = [
  { id: 'umum', name: 'Umum / General' },
  { id: 'warung', name: 'Warung / Toko Kelontong' },
  { id: 'laundry', name: 'Laundry' },
  { id: 'industri_kreatif', name: 'Industri Kreatif' },
  { id: 'toko_online', name: 'Toko Online / E-Commerce' },
  { id: 'restoran', name: 'Restoran / F&B' },
  { id: 'jasa', name: 'Jasa / Service' },
]

const toast = ref({ visible: false, message: '', type: 'success' })
const showToast = (message: string, type: 'success' | 'error' = 'success') => {
  toast.value = { visible: true, message, type }
  setTimeout(() => { toast.value.visible = false }, 3000)
}

// Self-profile (super admin's own profile)
const showMyProfile = ref(false)
const myProfile = ref({ username: '', phone_number: '', old_password: '', new_password: '' })
const savingMyProfile = ref(false)
const myProfileError = ref('')

const loadMyProfile = async () => {
  try {
    const data = await api.get('/api/profile')
    if (data.success && data.data) {
      myProfile.value = {
        username: data.data.username || '',
        phone_number: data.data.phone_number || '',
        old_password: '',
        new_password: '',
      }
    }
  } catch (e) {
    console.error('Failed to load my profile', e)
  }
}

const saveMyProfile = async () => {
  savingMyProfile.value = true
  myProfileError.value = ''
  try {
    const payload: any = {}
    if (myProfile.value.username) payload.username = myProfile.value.username
    if (myProfile.value.phone_number) payload.phone_number = myProfile.value.phone_number
    if (myProfile.value.new_password) {
      if (!myProfile.value.old_password) {
        myProfileError.value = 'Password lama harus diisi untuk mengganti password'
        savingMyProfile.value = false
        return
      }
      payload.old_password = myProfile.value.old_password
      payload.new_password = myProfile.value.new_password
    }
    const data = await api.put('/api/profile', payload)
    if (data.success) {
      showToast('Profil berhasil disimpan')
      myProfile.value.old_password = ''
      myProfile.value.new_password = ''
      showMyProfile.value = false
    } else {
      myProfileError.value = data.message || 'Gagal menyimpan'
    }
  } catch (e) {
    myProfileError.value = 'Kesalahan jaringan'
  } finally {
    savingMyProfile.value = false
  }
}

const planCounts = computed(() => {
  const counts = { free: 0, lite: 0, pro: 0 }
  tenants.value.forEach((t: any) => {
    const plan = t.plan as keyof typeof counts
    if (counts[plan] !== undefined) {
      counts[plan]++
    }
  })
  return counts
})

const fetchTenants = async () => {
  loadingTenants.value = true
  try {
    const data = await superadminApi.getTenants()
    if (data.success && data.data) {
      tenants.value = data.data
    }
  } catch (e) {
    console.error('Failed to fetch tenants', e)
  } finally {
    loadingTenants.value = false
  }
}

const confirmDelete = (tenant: any) => {
  deleteTarget.value = tenant
  deleteError.value = ''
}

const executeDelete = async () => {
  if (!deleteTarget.value) return
  deleting.value = true
  deleteError.value = ''

  try {
    const data = await superadminApi.deleteTenant(deleteTarget.value.id)
    if (data.success) {
      tenants.value = tenants.value.filter((t: any) => t.id !== deleteTarget.value.id)
      showToast('Tenant berhasil dihapus', 'success')
      deleteTarget.value = null
    } else {
      deleteError.value = data.message || 'Gagal menghapus tenant'
    }
  } catch (e) {
    deleteError.value = 'Kesalahan jaringan saat menghapus tenant'
  } finally {
    deleting.value = false
  }
}

const checkVerifierStatus = async () => {
  checkingStatus.value = true
  try {
    const data = await superadminApi.getVerifierStatus()
    if (data.success && data.data) {
      verifierStatus.value = data.data.status === 'connected' ? 'connected' : 'disconnected'
      verifierJID.value = data.data.jid || ''
      if (verifierStatus.value === 'connected') {
        qrCode.value = ''
        showToast('WhatsApp Verifier berhasil terhubung!', 'success')
      }
    }
  } catch (e) {
    showToast('Gagal memeriksa status verifier', 'error')
  } finally {
    checkingStatus.value = false
  }
}

const connectVerifier = async () => {
  loadingQR.value = true
  try {
    const data = await superadminApi.getVerifierQR()
    if (data.success && data.data) {
      if (data.data.status === 'qr') {
        qrCode.value = data.data.qr_code
      } else if (data.data.status === 'connected') {
        verifierStatus.value = 'connected'
        verifierJID.value = data.data.jid || ''
        showToast('WhatsApp Verifier sudah terhubung!', 'success')
      }
    } else {
      showToast(data.message || 'Gagal mendapatkan QR code', 'error')
    }
  } catch (e) {
    showToast('Gagal menghubungkan verifier', 'error')
  } finally {
    loadingQR.value = false
  }
}

const disconnectVerifier = async () => {
  disconnecting.value = true
  try {
    const data = await superadminApi.disconnectVerifier()
    if (data.success) {
      verifierStatus.value = 'disconnected'
      verifierJID.value = ''
      qrCode.value = ''
      showToast('WhatsApp Verifier telah diputuskan', 'success')
    } else {
      showToast(data.message || 'Gagal memutuskan verifier', 'error')
    }
  } catch (e) {
    showToast('Gagal memutuskan verifier', 'error')
  } finally {
    disconnecting.value = false
  }
}

const openEditProfile = async (tenant: any) => {
  editTarget.value = tenant
  profileError.value = ''
  editLogoFile.value = null
  editLogoPreview.value = ''
  editForm.value.new_password = ''
  editForm.value.logo_url = ''

  try {
    const data = await superadminApi.getTenantProfile(tenant.id)
    if (data.success && data.data) {
      const p = data.data
      editForm.value.name = p.name || ''
      editForm.value.business_name = p.business_name || ''
      editForm.value.wa_number = p.wa_number || ''
      editForm.value.business_address = p.business_address || ''
      editForm.value.business_type = p.business_type || 'umum'
      editForm.value.plan = p.plan || 'free'
      editForm.value.logo_url = p.logo_url || ''
      editForm.value.owner_phone = p.owner_phone || ''
      editForm.value.subdomain = p.subdomain || ''
      editForm.value.custom_domain = p.custom_domain || ''
    } else {
      profileError.value = 'Gagal memuat profil'
    }
  } catch (e) {
    profileError.value = 'Kesalahan jaringan'
  }
}

const onLogoFileChange = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  editLogoFile.value = file
  editLogoPreview.value = URL.createObjectURL(file)
}

const uploadLogo = async () => {
  if (!editLogoFile.value || !editTarget.value) return
  uploadingLogo.value = true
  try {
    const result = await superadminApi.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
    if (result.success) {
      editForm.value.logo_url = result.data?.logo_url || ''
      editLogoFile.value = null
      showToast('Logo berhasil diupload', 'success')
    } else {
      profileError.value = result.message || 'Gagal upload logo'
    }
  } catch (e) {
    profileError.value = 'Kesalahan jaringan saat upload'
  } finally {
    uploadingLogo.value = false
  }
}

const saveProfile = async () => {
  if (!editTarget.value) return
  savingProfile.value = true
  profileError.value = ''

  try {
    if (editLogoFile.value) {
      const logoResult = await superadminApi.uploadTenantLogo(editTarget.value.id, editLogoFile.value)
      if (!logoResult.success) {
        profileError.value = logoResult.message || 'Gagal upload logo'
        savingProfile.value = false
        return
      }
      editForm.value.logo_url = logoResult.data?.logo_url || ''
      editLogoFile.value = null
    }

    const payload: any = {
      tenant_id: editTarget.value.id,
      name: editForm.value.name,
      business_name: editForm.value.business_name,
      wa_number: editForm.value.wa_number,
      owner_phone: editForm.value.owner_phone,
      business_address: editForm.value.business_address,
      business_type: editForm.value.business_type,
      plan: editForm.value.plan,
      subdomain: editForm.value.subdomain,
      custom_domain: editForm.value.custom_domain
    }
    if (editForm.value.new_password) {
      payload.new_password = editForm.value.new_password
    }

    const result = await superadminApi.updateTenantProfile(payload)
    if (result.success) {
      showToast('Profil tenant berhasil disimpan', 'success')
      editTarget.value = null
      fetchTenants()
    } else {
      profileError.value = result.message || 'Gagal menyimpan'
    }
  } catch (e) {
    profileError.value = 'Kesalahan jaringan'
  } finally {
    savingProfile.value = false
  }
}

const closeAddModal = () => {
  showAddTenant.value = false
  formData.value = { name: '', username: '', password: 'Password123', email: '', phone_number: '', role: 'owner', plan: 'free', subdomain: '', custom_domain: '' }
}

// ── Plan Editor ──────────────────────────────────────────────────────────────────

const showPlanEditor = ref(false)
const editablePlans = ref<any[]>([])
const loadingPlans = ref(false)
const savingPlans = ref(false)
const planError = ref('')

const openPlanEditor = async () => {
  showPlanEditor.value = true
  planError.value = ''
  loadingPlans.value = true
  try {
    const data = await superadminApi.getPlans()
    const plans = data.data || (data.success ? data.data : null)
    if (plans && Array.isArray(plans)) {
      editablePlans.value = plans.map((p: any) => ({
        ...p,
        price_monthly_display: Math.round((p.price_monthly || 0) / 100),
        price_yearly_display: Math.round((p.price_yearly || 0) / 100),
      }))
    } else {
      planError.value = 'Gagal memuat daftar paket'
    }
  } catch (e) {
    planError.value = 'Kesalahan jaringan'
  } finally {
    loadingPlans.value = false
  }
}

const syncPlanPrice = (plan: any, kind: 'monthly' | 'yearly') => {
  if (kind === 'monthly') {
    plan.price_monthly = (plan.price_monthly_display || 0) * 100
  } else {
    plan.price_yearly = (plan.price_yearly_display || 0) * 100
  }
}

const savePlanPrices = async () => {
  savingPlans.value = true
  planError.value = ''
  try {
    let allOk = true
    for (const plan of editablePlans.value) {
      const result = await superadminApi.updatePlan(plan.id, {
        price_monthly: plan.price_monthly || 0,
        price_yearly: plan.price_yearly || 0,
        is_active: plan.is_active,
        sort_order: plan.sort_order || 0,
      })
      if (!result.success && result.status !== 200) {
        allOk = false
        planError.value = `Gagal menyimpan paket ${plan.name}: ${result.message}`
      }
    }
    if (allOk) {
      showToast('Harga paket berhasil diperbarui')
      showPlanEditor.value = false
    }
  } catch (e) {
    planError.value = 'Kesalahan jaringan'
  } finally {
    savingPlans.value = false
  }
}

const saveNewTenant = async () => {
  try {
    const data = await api.post('/api/umkm/admin/tenants', {
      name: formData.value.name,
      username: formData.value.username,
      email: formData.value.email,
      phone_number: formData.value.phone_number,
      plan: formData.value.plan,
      subdomain: formData.value.subdomain,
      custom_domain: formData.value.custom_domain
    })
    if (data.success) {
      showToast("Berhasil mendaftarkan UMKM baru!", "success")
      closeAddModal()
      fetchTenants()
    } else {
      showToast("Gagal: " + data.message, "error")
    }
  } catch (e) {
    console.error(e)
    showToast("Terjadi kesalahan jaringan.", "error")
  }
}

onMounted(() => {
  checkVerifierStatus()
  fetchTenants()
  loadMyProfile()
})
</script>

<style scoped>
.superadmin-dashboard {
  max-width: 1200px;
  margin: 0 auto;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
}

@media (max-width: 768px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

.card {
  padding: 1.5rem;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.card-header h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.status-badge {
  padding: 0.3rem 0.75rem;
  border-radius: 20px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-connected {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.status-disconnected {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.verifier-actions {
  text-align: center;
  padding: 1rem 0;
}

.verifier-actions p {
  margin-bottom: 1.25rem;
}

.qr-section {
  text-align: center;
}

.qr-image {
  width: 200px;
  height: 200px;
  border-radius: 8px;
  border: 2px solid var(--border-color);
}

.qr-actions {
  display: flex;
  justify-content: center;
  gap: 1rem;
  margin-top: 1rem;
}

.verifier-info {
  padding: 0.5rem 0;
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border-color);
}

.info-label {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.info-value {
  font-weight: 600;
  font-size: 0.9rem;
}

.stat-number {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--accent-primary);
}

.plan-stats {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.plan-stat {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 600;
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.data-table th,
.data-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
}

.data-table th {
  color: var(--text-secondary);
  font-weight: 500;
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.05em;
}

.badge {
  padding: 0.2rem 0.6rem;
  border-radius: 20px;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
}

.badge-free {
  background: rgba(100, 116, 139, 0.2);
  color: #94a3b8;
}

.badge-lite {
  background: rgba(245, 158, 11, 0.2);
  color: #fbbf24;
}

.badge-pro {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.badge-business {
  background: rgba(168, 85, 247, 0.2);
  color: #c084fc;
}

.toast-notification {
  position: fixed;
  bottom: 2rem;
  right: 2rem;
  padding: 1rem 1.5rem;
  border-radius: 8px;
  color: #fff;
  font-weight: 500;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  z-index: 9999;
  animation: slideIn 0.3s ease-out;
}

.toast-success {
  background-color: #10b981;
}

.toast-error {
  background-color: #ef4444;
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

.btn-edit {
  background: transparent;
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: #60a5fa;
  padding: 0.35rem 0.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 0.35rem;
}

.btn-edit:hover {
  background: rgba(59, 130, 246, 0.15);
  border-color: #60a5fa;
}

.logo-upload-section {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.logo-preview {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  flex-shrink: 0;
}

.logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-placeholder {
  font-size: 0.7rem;
  color: var(--text-secondary);
  text-align: center;
}

.form-group {
  margin-bottom: 0.75rem;
}

.form-group label {
  display: block;
  font-size: 0.8rem;
  font-weight: 500;
  margin-bottom: 0.3rem;
  color: var(--text-primary);
}

.form-control {
  width: 100%;
  padding: 0.6rem 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.875rem;
  box-sizing: border-box;
}

.form-control:focus {
  outline: none;
  border-color: var(--accent-primary);
}

.btn-delete {
  background: transparent;
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #ef4444;
  padding: 0.35rem 0.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.btn-delete:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: #ef4444;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  padding: 1rem;
}

.modal-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 2rem;
  max-width: 420px;
  width: 100%;
}

@media (max-width: 480px) {

  .data-table th,
  .data-table td {
    padding: 0.5rem;
    font-size: 0.8rem;
  }
}

/* Plan Editor */
.plan-editor-row {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 1rem;
  margin-bottom: 1rem;
}

.plan-editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.plan-editor-fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

/* Toggle Switch */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  cursor: pointer;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  inset: 0;
  background-color: rgba(100, 116, 139, 0.4);
  border-radius: 22px;
  transition: 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: white;
  border-radius: 50%;
  transition: 0.2s;
}

.toggle-switch input:checked + .toggle-slider {
  background-color: #10b981;
}

.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(18px);
}

@media (max-width: 480px) {
  .plan-editor-fields {
    grid-template-columns: 1fr;
  }
}

</style>
