<template>
  <div class="candidates-page flex flex-col gap-4">
    <div class="header">
      <h2 style="font-size: 1.5rem; color: var(--accent-primary);">Verifikasi Calon</h2>
      <p style="color: var(--text-secondary);">Daftar kandidat yang perlu diverifikasi secara manual oleh sistem.</p>
    </div>

    <div class="table-container glass-card">
      <table class="data-table">
        <thead>
          <tr>
            <th>ID / Nama Calon</th>
            <th>Dokumen Verifikasi</th>
            <th>Status</th>
            <th>Aksi</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="candidates.length === 0">
            <td colspan="4" style="text-align: center; color: var(--text-muted);">Belum ada kandidat mendaftar.</td>
          </tr>
          <tr v-for="c in candidates" :key="c.id">
            <td>
              <div style="font-weight: 600;">{{ c.name }}</div>
              <div style="font-size: 0.8rem; color: var(--text-muted);">{{ c.id }}</div>
            </td>
            <td>
              <a v-if="c.verification_document" href="#" style="color: var(--accent-primary);">Lihat Dokumen</a>
              <span v-else style="color: var(--text-muted);">Tidak ada dokumen</span>
            </td>
            <td>
              <span class="badge" :class="c.is_verified ? 'badge-success' : 'badge-warning'">
                {{ c.is_verified ? 'Terverifikasi' : 'Menunggu' }}
              </span>
            </td>
            <td>
              <button 
                v-if="!c.is_verified" 
                class="btn-primary btn-sm" 
                @click="verifyCandidate(c.id)"
              >
                Verifikasi Manual
              </button>
              <button v-else class="btn-outline btn-sm" disabled>Sudah Diverifikasi</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiClient } from '../api'

import { ref, onMounted } from 'vue'

const candidates = ref<any[]>([])

const fetchCandidates = async () => {
  try {
    const res = await apiClient('/candidates', {
      headers: { 'X-Tenant-ID': 'tenant-1' } // Demo tenant
    })
    const data = await res.json()
    if (data.success) {
      candidates.value = data.data
    }
  } catch (err) {
    console.error(err)
  }
}

const verifyCandidate = async (id: string) => {
  if (!confirm('Apakah Anda yakin ingin memverifikasi calon ini? Akses kampanye penuh akan diberikan.')) return

  try {
    const res = await apiClient(`/candidates/${id}/verify`, {
      method: 'PUT',
      headers: { 'X-Tenant-ID': 'tenant-1' }
    })
    const data = await res.json()
    if (data.success) {
      alert('Kandidat berhasil diverifikasi!')
      fetchCandidates()
    } else {
      alert('Gagal: ' + data.message)
    }
  } catch (err) {
    console.error(err)
  }
}

onMounted(fetchCandidates)
</script>

<style scoped>
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
.data-table th, .data-table td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}
.data-table th {
  font-weight: 600;
  color: var(--text-secondary);
  background-color: rgba(255, 255, 255, 0.02);
}
.badge { padding: 0.25rem 0.5rem; border-radius: 4px; font-size: 0.75rem; color: white; }
.badge-warning { background: #f59e0b; }
.badge-success { background: #10b981; }
.btn-primary {
  background: var(--accent-gradient);
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-weight: 600;
}
.btn-outline {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm);
  cursor: not-allowed;
}
.btn-sm { padding: 0.3rem 0.8rem; font-size: 0.8rem; }
.flex { display: flex; }
.flex-col { flex-direction: column; }
.gap-4 { gap: 1rem; }
</style>
