<template>
  <div class="users-page">
    <div class="header flex justify-between align-center" style="margin-bottom: 2rem;">
      <div>
        <h2 style="font-size: 1.5rem; color: var(--accent-primary);">Manajemen Pengguna</h2>
        <p style="color: var(--text-secondary);">Kelola hierarki admin, koordinator, dan relawan.</p>
      </div>
      <button type="button" class="btn-primary" @click="showModal = true">+ Tambah Pengguna</button>
    </div>

    <div class="table-container glass-card">
      <table class="data-table">
        <thead>
          <tr>
            <th>Username</th>
            <th>Email</th>
            <th>Role / Jenjang</th>
            <th>No. Telepon</th>
            <th>Terdaftar</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="users.length === 0">
            <td colspan="5" style="text-align: center; color: var(--text-muted);">Belum ada data pengguna.</td>
          </tr>
          <tr v-for="u in users" :key="u.id">
            <td style="font-weight: 600;">{{ u.username }}</td>
            <td>{{ u.email }}</td>
            <td>
              <span class="badge" :class="getRoleBadge(u.role)">{{ u.role.toUpperCase() }}</span>
            </td>
            <td>{{ u.phone_number || '-' }}</td>
            <td>{{ new Date(u.created_at).toLocaleDateString('id-ID') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showModal" class="modal-overlay">
      <div class="modal-content glass-card animate-fade-in" style="max-height: 90vh; overflow-y: auto;">
        <h3 style="margin-bottom: 1.5rem">Tambah Pengguna Manual</h3>
        <p class="text-muted" style="margin-bottom: 1rem; font-size: 0.9rem;">
          Pendaftaran manual akan melewati verifikasi OTP. Gunakan hanya untuk pendaftaran via kandidat/admin.
        </p>
        <form @submit.prevent="addUser" class="flex flex-col gap-4">
          <div>
            <label for="user-nik" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">NIK</label>
            <input id="user-nik" v-model="form.nik" placeholder="NIK (Nomor Induk Kependudukan)" required class="input-field" maxlength="16" />
          </div>
          <div>
            <label for="user-name" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Nama Lengkap</label>
            <input id="user-name" v-model="form.name" placeholder="Nama Lengkap" required class="input-field" />
          </div>
          <div>
            <label for="user-phone" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Nomor Telepon / WA</label>
            <input id="user-phone" v-model="form.phone_number" placeholder="Nomor Telepon / WA" required class="input-field" />
          </div>

          <div>
            <label for="user-role" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Role</label>
            <select id="user-role" v-model="form.role" class="input-field" required>
              <option value="" disabled>-- Pilih Jenjang (Role) --</option>
              <option value="admin">Admin</option>
              <option value="koordinator">Koordinator</option>
              <option value="relawan">Relawan</option>
              <option value="user_biasa">Pemilih Biasa</option>
            </select>
          </div>

          <h4 style="margin-top: 0.5rem;">Informasi Lokasi</h4>
          <div>
            <label for="user-dusun" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Dusun / Lingkungan</label>
            <input id="user-dusun" v-model="form.dusun" placeholder="Dusun / Lingkungan" required class="input-field" />
          </div>
          <div>
            <label for="user-tps" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Nomor TPS</label>
            <input id="user-tps" v-model="form.tps" placeholder="Nomor TPS (contoh: TPS 01)" required class="input-field" />
          </div>

          <div class="flex justify-end gap-2" style="margin-top: 1rem;">
            <button type="button" class="btn-outline" @click="showModal = false">Batal</button>
            <button type="submit" class="btn-primary">Simpan Pengguna</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient, authApi } from '../api'

import { ref, onMounted } from 'vue'

const users = ref<any[]>([])
const showModal = ref(false)
const form = ref({ nik: '', name: '', phone_number: '', role: '', dusun: '', tps: '' })

const fetchUsers = async () => {
  try {
    const res = await apiClient('/users', {
      headers: { 'X-Tenant-ID': 'tenant-1' } // hardcode tenant for demo
    })
    const data = await res.json()
    if (data.success) {
      users.value = data.data
    }
  } catch { /* ignore fetch errors */ }
}

const addUser = async () => {
  try {
    const res = await authApi.manualRegister(
      localStorage.getItem('accessToken') || '',
      {
        nik: form.value.nik,
        phoneNumber: form.value.phone_number,
        name: form.value.name,
        role: form.value.role,
        dusun: form.value.dusun,
        tps: form.value.tps
      }
    )
    const data = await res.json()
    if (data.success) {
      form.value = { nik: '', name: '', phone_number: '', role: '', dusun: '', tps: '' }
      showModal.value = false
      fetchUsers()
    } else {
      alert("Gagal menambahkan pengguna: " + data.message)
    }
  } catch { /* ignore fetch errors */ }
}

const getRoleBadge = (role: string) => {
  if (role === 'admin') return 'badge-danger'
  if (role === 'koordinator') return 'badge-warning'
  return 'badge-primary'
}

onMounted(fetchUsers)
</script>

<style scoped>
.input-field {
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--surface-1);
  color: var(--text-primary);
  outline: none;
  font-family: inherit;
  width: 100%;
}

.input-field:focus {
  border-color: var(--accent-primary);
}

.btn-primary {
  background: var(--accent-gradient);
  color: white;
  border: none;
  padding: 0.5rem 1.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-weight: 600;
}

.btn-outline {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  padding: 0.5rem 1.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.table-container {
  overflow-x: auto;
  border-radius: var(--radius-md);
  background: var(--surface-0);
  border: 1px solid var(--border-color);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
}

.data-table th,
.data-table td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.data-table th {
  font-weight: 600;
  color: var(--text-secondary);
  background-color: rgba(255, 255, 255, 0.02);
}

.badge-danger {
  background: #dc2626;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
}

.badge-warning {
  background: #92400e;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
}

.badge-primary {
  background: #2563eb;
  color: white;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
}

.modal-content {
  width: 100%;
  max-width: 500px;
  padding: 2rem;
  background: var(--surface-0);
  border-radius: var(--radius-md);
}

.flex {
  display: flex;
}

.justify-between {
  justify-content: space-between;
}

.justify-end {
  justify-content: flex-end;
}

.align-center {
  align-items: center;
}

.gap-4 {
  gap: 1rem;
}

.gap-2 {
  gap: 0.5rem;
}

.flex-col {
  flex-direction: column;
}
</style>