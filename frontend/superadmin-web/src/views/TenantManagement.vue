<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTenantManagement } from '../composables/useTenantManagement'
import { api } from '../api/client'

const router = useRouter()
const planOptions = ref<any[]>([])

const {
  tenants, loading, deleting, deleteTarget, deleteError,
  showAddTenant, formData, savingAddTenant: saving,
  planCounts, fetchTenants,
  openEditProfile, editTarget, editForm, editFormRaw,
  editLogoPreview, profileError, savingProfile,
  onLogoFileChange, saveProfile, saveNewTenant,
  confirmDelete, executeDelete,
  businessTypes,
} = useTenantManagement()

onMounted(async () => {
  await fetchTenants()
  const res = await api.listPlans()
  if (res.success && res.data) planOptions.value = res.data
})

async function impersonateTenant(tenantId: string, name: string) {
  if (!confirm(`Login sebagai ${name}?`)) return
  try {
    const res = await fetch(`/admin/tenants/${tenantId}/impersonate`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${localStorage.getItem('access_token')}` }
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.message)
    sessionStorage.setItem('superadmin_token', localStorage.getItem('access_token') || '')
    const url = new URL(globalThis.location.origin.replace('3202', '3201'))
    url.searchParams.set('impersonate_token', data.data.access_token)
    window.open(url.toString(), '_blank')
    alert(`Login sebagai ${name} — cek tab baru.`)
  } catch (e: any) {
    alert(e.message)
  }
}

function copyId(id: string) {
  navigator.clipboard.writeText(id)
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Tenant Management</h1>
        <p class="subtitle">Kelola semua tenant, paket, dan akses</p>
      </div>
      <div class="header-actions">
        <button class="btn" @click="fetchTenants" :disabled="loading">🔄 Refresh</button>
        <button class="btn btn-accent" @click="showAddTenant = true">+ Tambah Tenant</button>
      </div>
    </div>

    <!-- Stats row -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-label">Total Tenant</div>
        <div class="stat-value">{{ tenants.length }}</div>
      </div>
      <div class="stat-card" v-for="(cnt, plan) in planCounts" :key="plan">
        <div class="stat-label">{{ plan?.toUpperCase() }}</div>
        <div class="stat-value">{{ cnt }}</div>
      </div>
    </div>

    <!-- Action shortcuts -->
    <div class="action-row">
      <button class="action-btn" @click="router.push('/plan-features')">💰 Kelola Paket</button>
      <button class="action-btn" @click="router.push('/addon-pricing')">📦 Kelola Add-on</button>
      <button class="action-btn" @click="router.push('/feature-matrix')">🔲 Feature Matrix</button>
    </div>

    <!-- Tenant table -->
    <div class="section">
      <h2>Daftar Tenant</h2>
      <div v-if="loading" class="loading">Memuat...</div>
      <div v-else-if="!tenants.length" class="empty">Belum ada tenant.</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Nama Usaha</th>
            <th>Owner</th>
            <th>Phone</th>
            <th>Paket</th>
            <th>Xendit</th>
            <th>Daftar</th>
            <th>Aksi</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in tenants" :key="t.id">
            <td>
              <strong>{{ t.name }}</strong>
              <button class="copy-btn" @click="copyId(t.id)" title="Copy ID">📋</button>
            </td>
            <td>{{ t.owner_username || '-' }}</td>
            <td>{{ t.owner_phone || '-' }}</td>
            <td><span :class="['badge', 'badge-' + (t.plan || 'lite')]">{{ (t.plan || 'lite').toUpperCase() }}</span></td>
            <td>
              <code v-if="t.xendit_merchant_id" style="font-size: 11px;">{{ t.xendit_merchant_id }}</code>
              <span v-else style="color: var(--muted); font-size: 12px;">SaaS</span>
            </td>
            <td>{{ t.created_at ? new Date(t.created_at).toLocaleDateString('id-ID') : '-' }}</td>
            <td class="actions">
              <button class="btn-sm" @click="openEditProfile(t)">✏️ Edit</button>
              <button class="btn-sm btn-danger" @click="confirmDelete(t)">🗑️</button>
              <button class="btn-sm btn-impersonate" @click="impersonateTenant(t.id, t.name)">🔓</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Add Tenant Modal -->
    <Teleport to="body">
      <div v-if="showAddTenant" class="modal-overlay" @click.self="showAddTenant = false">
        <div class="modal-card">
          <h3>Tambah Tenant Baru</h3>
          <div class="form-grid">
            <label> Nama Usaha <input v-model="formData.name" class="form-control" placeholder="Nama bisnis"> </label>
            <label> Username Owner <input v-model="formData.username" class="form-control" placeholder="Username login"> </label>
            <label> Password <input v-model="formData.password" type="password" class="form-control" placeholder="Min 8 karakter"> </label>
            <label> Email <input v-model="formData.email" type="email" class="form-control" placeholder="email@example.com"> </label>
            <label> No. HP <input v-model="formData.phone_number" class="form-control" placeholder="08xxxxxxxxxx"> </label>
            <label> Paket
              <select v-model="formData.plan" class="form-control">
                <option v-for="p in planOptions" :key="p.id" :value="p.id">{{ p.name }}</option>
              </select>
            </label>
          </div>
          <div class="modal-actions">
            <button class="btn" @click="showAddTenant = false">Batal</button>
            <button class="btn btn-accent" @click="saveNewTenant" :disabled="saving">{{ saving ? 'Menyimpan...' : 'Simpan' }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Edit Tenant Modal -->
    <Teleport to="body">
      <div v-if="editTarget" class="modal-overlay" @click.self="editTarget = null">
        <div class="modal-card modal-wide">
          <h3>Edit Tenant: {{ editTarget.name }}</h3>
          <div class="form-grid">
            <label> Nama Usaha <input v-model="editForm.name" class="form-control"> </label>
            <label> Business Name <input v-model="editForm.business_name" class="form-control"> </label>
            <label> Owner Username <input v-model="editForm.owner_username" class="form-control"> </label>
            <label> Owner Phone <input v-model="editForm.owner_phone" class="form-control"> </label>
            <label> Business Type
              <select v-model="editForm.business_type" class="form-control">
                <option v-for="bt in businessTypes" :key="bt.id" :value="bt.id">{{ bt.name }}</option>
              </select>
            </label>
            <label> Paket
              <select v-model="editForm.plan" class="form-control">
                <option v-for="p in planOptions" :key="p.id" :value="p.id">{{ p.name }}</option>
              </select>
            </label>
            <label> Alamat <input v-model="editForm.business_address" class="form-control"> </label>
            <label> Xendit Merchant ID <input v-model="editForm.xendit_merchant_id" class="form-control"> </label>
            <label> WA Number <input v-model="editForm.wa_number" class="form-control"> </label>
            <label> Password Baru (kosongkan jika tidak diubah) <input v-model="editFormRaw.new_password" type="password" class="form-control" placeholder="Min 8 karakter"> </label>
            <label class="span2"> Logo
              <div class="logo-upload">
                <img v-if="editLogoPreview" :src="editLogoPreview" class="logo-preview"></img>
                <input type="file" accept="image/*" @change="onLogoFileChange"></input>
              </div>
            </label>
          </div>
          <p v-if="profileError" class="error-msg">{{ profileError }}</p>
          <div class="modal-actions">
            <button class="btn" @click="editTarget = null">Batal</button>
            <button class="btn btn-accent" @click="saveProfile" :disabled="savingProfile">{{ savingProfile ? 'Menyimpan...' : 'Simpan' }}</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete Confirm -->
    <Teleport to="body">
      <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget = null">
        <div class="modal-card" style="max-width: 400px;">
          <h3>Hapus Tenant?</h3>
          <p>Hapus <strong>{{ deleteTarget.name }}</strong>? Aksi ini tidak bisa dibatalkan.</p>
          <p v-if="deleteError" class="error-msg">{{ deleteError }}</p>
          <div class="modal-actions">
            <button class="btn" @click="deleteTarget = null">Batal</button>
            <button class="btn btn-danger" @click="executeDelete" :disabled="deleting">{{ deleting ? 'Menghapus...' : 'Hapus Permanen' }}</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; gap: 16px; }
.page-header h1 { font-size: 22px; margin-bottom: 2px; }
.subtitle { color: var(--muted); font-size: 13px; }
.header-actions { display: flex; gap: 8px; align-items: center; flex-shrink: 0; }

.stats-row { display: flex; gap: 12px; margin-bottom: 20px; flex-wrap: wrap; }
.stat-card { background: var(--card); border: 1px solid var(--border); border-radius: 8px; padding: 14px 20px; min-width: 90px; }
.stat-label { font-size: 11px; text-transform: uppercase; color: var(--muted); letter-spacing: 0.5px; }
.stat-value { font-size: 22px; font-weight: 700; margin-top: 4px; }

.action-row { display: flex; gap: 8px; margin-bottom: 24px; flex-wrap: wrap; }
.action-btn { background: var(--card); border: 1px solid var(--border); color: var(--text); padding: 7px 14px; border-radius: 6px; font-size: 13px; cursor: pointer; }
.action-btn:hover { background: var(--bg); }

.section { background: var(--card); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; margin-bottom: 24px; }
.section h2 { font-size: 14px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.5px; padding: 16px 20px 12px; border-bottom: 1px solid var(--border); }

.data-table { width: 100%; border-collapse: collapse; }
.data-table th { text-align: left; font-size: 11px; text-transform: uppercase; color: var(--muted); padding: 10px 16px; border-bottom: 1px solid var(--border); letter-spacing: 0.3px; }
.data-table td { padding: 12px 16px; border-bottom: 1px solid rgba(255,255,255,0.03); font-size: 13px; }
.data-table tr:last-child td { border-bottom: none; }
.data-table tr:hover td { background: rgba(255,255,255,0.02); }
.copy-btn { background: none; border: none; cursor: pointer; font-size: 12px; opacity: 0.5; margin-left: 4px; }
.copy-btn:hover { opacity: 1; }

.badge { padding: 2px 8px; border-radius: 4px; font-size: 10px; font-weight: 700; }
.badge-lite { background: rgba(100,116,139,0.2); color: #94a3b8; }
.badge-pro { background: rgba(59,130,246,0.15); color: #60a5fa; }
.badge-ultimate { background: rgba(168,85,247,0.15); color: #a855f7; }

.actions { display: flex; gap: 6px; align-items: center; }
.btn-sm { background: var(--bg); border: 1px solid var(--border); color: var(--text); padding: 4px 8px; border-radius: 5px; font-size: 12px; cursor: pointer; }
.btn-sm.btn-danger { border-color: rgba(239,68,68,0.4); color: var(--danger); }
.btn-sm.btn-impersonate { background: linear-gradient(135deg, #7c3aed, #4f46e5); border: none; color: white; padding: 4px 8px; border-radius: 5px; font-size: 12px; cursor: pointer; }

.loading, .empty { text-align: center; color: var(--muted); padding: 40px; }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.6); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal-card { background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 28px; max-width: 520px; width: 100%; max-height: 90vh; overflow-y: auto; }
.modal-wide { max-width: 720px; }
.modal-card h3 { margin: 0 0 20px; font-size: 18px; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin-bottom: 20px; }
.form-grid label { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--muted); }
.form-grid label.span2 { grid-column: span 2; }
.form-grid input, .form-grid select { font-size: 14px; }
.logo-upload { display: flex; align-items: center; gap: 10px; }
.logo-preview { width: 48px; height: 48px; object-fit: cover; border-radius: 6px; border: 1px solid var(--border); }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; }
.error-msg { color: var(--danger); font-size: 13px; margin-bottom: 12px; }

/* Global btn helpers */
.btn { background: var(--bg); border: 1px solid var(--border); color: var(--text); padding: 8px 16px; border-radius: 6px; font-size: 13px; cursor: pointer; }
.btn-accent { background: var(--accent); border-color: var(--accent); color: white; }
.btn-danger { background: rgba(239,68,68,0.15); border-color: rgba(239,68,68,0.4); color: var(--danger); }
.form-control { background: var(--bg); border: 1px solid var(--border); color: var(--text); padding: 8px 12px; border-radius: 6px; width: 100%; box-sizing: border-box; }
</style>
