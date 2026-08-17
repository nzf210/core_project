<template>
  <div class="voters">
    <h3 style="margin-bottom: 1rem; color: var(--text-secondary)">Pendaftaran Pemilih Baru</h3>

    <ErrorBoundary
      :error="votersApi.state.value.error"
      title="Failed to load voters"
      :on-retry="fetchVoters"
      @dismiss="votersApi.reset()"
    />

    <form @submit.prevent="addVoter" class="flex flex-col gap-4" style="max-width: 500px; margin-bottom: 2rem;">
      <div>
        <label for="voter-nik" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">NIK</label>
        <input id="voter-nik" v-model="form.nik" placeholder="NIK (Akan Dienkripsi)" required class="input-field" />
      </div>
      <div>
        <label for="voter-name" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Nama Lengkap</label>
        <input id="voter-name" v-model="form.name" placeholder="Nama Lengkap" required class="input-field" />
      </div>
      <div>
        <label for="voter-address" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Alamat</label>
        <input id="voter-address" v-model="form.address" placeholder="Alamat Lengkap" required class="input-field" />
      </div>
      <div>
        <label for="voter-phone" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Nomor Telepon</label>
        <input id="voter-phone" v-model="form.phone" placeholder="Nomor Telepon" required class="input-field" />
      </div>
      <div>
        <label for="voter-status" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Status</label>
        <select id="voter-status" v-model="form.status" required class="input-field">
          <option value="" disabled>-- Pilih Status --</option>
          <option value="uncontacted">Belum Dihubungi</option>
          <option value="undecided">Belum Ada Pilihan (Potensi)</option>
          <option value="supported">Pendukung</option>
          <option value="rejected">Menolak</option>
        </select>
      </div>
      <div>
        <label for="voter-potential" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Potensi Suara</label>
        <select id="voter-potential" v-model="form.potential_level" class="input-field">
          <option value="" disabled>-- Potensi Suara --</option>
          <option value="high">Pasti (High)</option>
          <option value="medium">Ragu (Medium)</option>
          <option value="low">Lemah (Low)</option>
        </select>
      </div>
      <div>
        <label for="voter-competitor" style="display:block;font-size:.85rem;margin-bottom:.25rem;color:var(--text-secondary)">Calon Lain</label>
        <input id="voter-competitor" v-model="form.competitor_support" placeholder="Mendukung Calon Lain (Sebutkan jika ada)" class="input-field" />
      </div>
      <button type="submit" class="btn-primary" style="align-self: flex-start;">Registrasi Pemilih</button>
    </form>

    <h3 style="margin-bottom: 1rem; color: var(--text-secondary)">Daftar Pemilih</h3>
    <div class="table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID Pemilih</th>
            <th>Status Relasi</th>
            <th>Potensi</th>
            <th>Pendukung Calon Lain</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="voters.length === 0">
            <td colspan="4" style="text-align: center; color: var(--text-muted);">Belum ada data pemilih terdaftar.</td>
          </tr>
          <tr v-for="v in voters" :key="v.id">
            <td><code>{{ v.id.substring(0, 8) }}...</code></td>
            <td><span class="badge badge-primary">{{ v.status }}</span></td>
            <td><span class="badge badge-secondary">{{ v.potential_level || '-' }}</span></td>
            <td><span class="badge badge-error">{{ v.competitor_support || '-' }}</span></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'
import { ref, onMounted } from 'vue'
import { useApiWithRetry } from '../composables/useApiWithRetry'
import ErrorBoundary from './ErrorBoundary.vue'

interface Voter {
  id: string
  nik: string
  name: string
  address: string
  phone: string
  status: string
  potential_level: string
  competitor_support: string
}

const voters = ref<Voter[]>([])
const form = ref({ nik: '', name: '', address: '', phone: '', status: '', potential_level: '', competitor_support: '' })

const votersApi = useApiWithRetry<Voter[]>()

const fetchVoters = async () => {
  await votersApi.execute(() => apiClient('/voters', {
    headers: { 'X-Tenant-ID': 'default' }
  }), {
    onSuccess: (data) => {
      voters.value = Array.isArray(data) ? data : []
    },
    silent: false
  })
}

const addVoter = async () => {
  try {
    const res = await apiClient('/voters', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'default' },
      body: JSON.stringify(form.value)
    })
    const data = await res.json()
    if (data.success) {
      form.value = { nik: '', name: '', address: '', phone: '', status: '', potential_level: '', competitor_support: '' }
      fetchVoters()
    }
  } catch {
    // fetch error — form stays open for retry
  }
}

onMounted(fetchVoters)
</script>

<style scoped>
.input-field {
  padding: 0.5rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
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
.table-container {
  overflow-x: auto;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th, .data-table td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}
.data-table th {
  font-weight: 600;
  color: var(--text-secondary);
  background-color: var(--bg-tertiary);
}
</style>
