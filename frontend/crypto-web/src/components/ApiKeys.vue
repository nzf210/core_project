<template>
  <div class="api-keys-container">
    <div class="header-actions flex items-center justify-between" style="margin-bottom: 2rem;">
      <h2>Manajemen API Key Exchange</h2>
      <button class="btn btn-primary" @click="showModal = true">Tambah API Key</button>
    </div>

    <div class="table-container glass-card">
      <table class="data-table">
        <thead>
          <tr>
            <th>Exchange</th>
            <th>Label Nama</th>
            <th>API Key</th>
            <th>Status</th>
            <th>Aksi</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="k in apiKeys" :key="k.id">
            <td>
              <div style="display: flex; align-items: center; gap: 0.5rem;">
                <span :class="['exchange-icon', k.exchange.toLowerCase()]"></span>
                {{ k.exchange }}
              </div>
            </td>
            <td>{{ k.label }}</td>
            <td style="font-family: monospace; color: var(--text-secondary);">
              <code>{{ k.masked_key }}</code>
            </td>
            <td><span class="badge badge-success" :class="{'bg-success': k.is_active}">{{ k.is_active ? 'Active' : 'Inactive' }}</span></td>
            <td>
              <button class="action-btn text-danger" @click="deleteKey(k.id)">Hapus</button>
            </td>
          </tr>
          <tr v-if="apiKeys.length === 0">
            <td colspan="5" style="text-align: center; padding: 2rem; color: var(--text-muted);">
              Belum ada API Key. Tambahkan kunci Binance/Tokocrypto Anda untuk mulai trading.
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Modal Tambah API Key -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-content glass-card">
        <h3>Tambah API Key Baru</h3>
        
        <div class="form-group">
          <label>Exchange</label>
          <select v-model="form.exchange" class="form-control">
            <option value="Binance">Binance</option>
            <option value="Tokocrypto">Tokocrypto</option>
            <option value="Indodax">Indodax</option>
          </select>
        </div>

        <div class="form-group">
          <label>Label / Nama</label>
          <input v-model="form.label" type="text" class="form-control" placeholder="cth: Main Binance Account" />
        </div>
        
        <div class="form-group">
          <label>API Key</label>
          <input v-model="form.key" type="text" class="form-control" placeholder="Paste API Key disini" />
        </div>
        
        <div class="form-group">
          <label>API Secret</label>
          <input v-model="form.secret" type="password" class="form-control" placeholder="Paste Secret Key disini" />
        </div>

        <div class="modal-actions">
          <button class="btn" style="background: transparent; border: 1px solid var(--border-color); color: var(--text-secondary)" @click="showModal = false">Batal</button>
          <button class="btn btn-primary" @click="saveKey">Simpan & Verifikasi</button>
        </div>
      </div>
    </div>

    <!-- Modal Konfirmasi Hapus -->
    <div v-if="showDeleteModal" class="modal-overlay" @click.self="showDeleteModal = false">
      <div class="modal-content glass-card" style="max-width: 400px; text-align: center;">
        <h3 style="color: var(--danger); margin-bottom: 1rem;">Hapus API Key?</h3>
        <p style="color: var(--text-secondary); margin-bottom: 2rem;">Yakin ingin menghapus API Key ini? Bot yang menggunakan key ini akan gagal transaksi.</p>
        <div class="modal-actions" style="justify-content: center;">
          <button class="btn" style="background: transparent; border: 1px solid var(--border-color); color: var(--text-secondary)" @click="showDeleteModal = false">Batal</button>
          <button class="btn btn-danger" style="background: var(--danger); color: white; border: none; padding: 0.75rem 1.5rem; border-radius: var(--radius-sm); cursor: pointer; font-weight: 600;" @click="confirmDelete">Hapus</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { toast } from 'vue3-toastify'
import api from '../api'

const showModal = ref(false)
const showDeleteModal = ref(false)
const keyToDelete = ref('')
const form = ref({ exchange: 'binance', label: '', key: '', secret: '' })
const isLoading = ref(false)

const apiKeys = ref<any[]>([])

const fetchApiKeys = async () => {
  try {
    const res = await api.get('/api/crypto/apikeys')
    if (res.data.success) {
      apiKeys.value = res.data.data || []
    }
  } catch (err) {
    console.error('Failed to fetch API keys', err)
  }
}

onMounted(() => {
  fetchApiKeys()
})



const deleteKey = (id: string) => {
  keyToDelete.value = id
  showDeleteModal.value = true
}

const confirmDelete = async () => {
  if (!keyToDelete.value) return
  try {
    await api.delete(`/api/crypto/apikeys/${keyToDelete.value}`)
    await fetchApiKeys()
    toast.success('API Key berhasil dihapus')
  } catch (err) {
    toast.error('Gagal menghapus API Key')
  } finally {
    showDeleteModal.value = false
    keyToDelete.value = ''
  }
}

const saveKey = async () => {
  if (!form.value.key || !form.value.secret) {
    toast.error('API Key dan Secret harus diisi')
    return
  }
  
  isLoading.value = true
  try {
    const res = await api.post('/api/crypto/apikeys', {
      exchange: form.value.exchange,
      label: form.value.label,
      api_key: form.value.key,
      api_secret: form.value.secret
    })
    
    if (res.data.success) {
      showModal.value = false
      form.value = { exchange: 'binance', label: '', key: '', secret: '' }
      toast.success('API Key berhasil ditambahkan dan diverifikasi!')
      await fetchApiKeys()
    } else {
      toast.error(res.data.message || 'Gagal menambahkan API Key')
    }
  } catch (err: any) {
    toast.error(err.response?.data?.message || 'Terjadi kesalahan')
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th, .data-table td {
  padding: 1.25rem 1rem;
  border-bottom: 1px solid var(--border-color);
  text-align: left;
}
.data-table th {
  color: var(--text-secondary);
  font-weight: 500;
  text-transform: uppercase;
  font-size: 0.75rem;
  letter-spacing: 0.05em;
}
.exchange-icon {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: inline-block;
  background: var(--bg-tertiary);
}
.exchange-icon.binance { background: #f3ba2f; }
.exchange-icon.tokocrypto { background: #1a1a1a; border: 1px solid #333; }

.action-btn {
  background: transparent;
  border: none;
  font-weight: 600;
  cursor: pointer;
}
.text-danger { color: var(--danger); }
.text-danger:hover { text-decoration: underline; }

.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  backdrop-filter: blur(5px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  width: 100%;
  max-width: 500px;
  padding: 2rem;
}
.form-group {
  margin-bottom: 1.25rem;
}
.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}
.form-control {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  margin-top: 2rem;
}
</style>
