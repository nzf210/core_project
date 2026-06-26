<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { request } from '../api/client'

interface License {
  id: string
  license_key: string
  program_name: string
  election_type: string
  is_used: boolean
  used_by_tenant_id: string | null
  created_at: string
  used_at: string | null
  max_voters: number
  wargame_tokens: number
}

const licenses = ref<License[]>([])
const loading = ref(true)
const generating = ref(false)
const error = ref('')
const successMsg = ref('')
const filterUsed = ref<string>('all')
const copyFeedback = ref<Record<string, boolean>>({})

const form = ref({
  election_type: 'pilkada',
  max_voters: 10000,
  wargame_tokens: 10,
  validity_days: 365,
  quantity: 1,
  program_name: '',
})

const ELECTION_TYPES = ['pilkada', 'pileg_dpr', 'pileg_dprd', 'dpd', 'pilpres']

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams()
    if (filterUsed.value !== 'all') params.set('used', filterUsed.value)
    params.set('limit', '100')
    const res = await request(`/api/superadmin/licenses?${params.toString()}`)
    licenses.value = res.data || []
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function generateLicense() {
  generating.value = true
  error.value = ''
  try {
    const res = await request('/api/superadmin/licenses/generate', {
      method: 'POST',
      body: JSON.stringify({
        election_type: form.value.election_type,
        max_voters: Number(form.value.max_voters),
        wargame_tokens: Number(form.value.wargame_tokens),
        validity_days: Number(form.value.validity_days),
        quantity: Number(form.value.quantity),
        program_name: form.value.program_name || undefined,
      }),
    })
    const generated = res.data?.keys || [res.data?.license_key].filter(Boolean)
    successMsg.value = `${generated.length} license key berhasil dibuat`
    setTimeout(() => successMsg.value = '', 4000)
    await load()
  } catch (e: any) {
    error.value = 'Gagal generate: ' + e.message
  } finally {
    generating.value = false
  }
}

async function copyKey(key: string) {
  try {
    await navigator.clipboard.writeText(key)
    copyFeedback.value[key] = true
    setTimeout(() => delete copyFeedback.value[key], 2000)
  } catch {}
}

function formatDate(d: string | null) {
  if (!d) return '-'
  return new Date(d).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' })
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1>Campaign Licenses</h1>
        <p class="subtitle">Generate dan kelola license key B2B untuk modul Campaign (F044).</p>
      </div>
    </div>

    <div v-if="successMsg" class="success-banner">✅ {{ successMsg }}</div>
    <div v-if="error" class="error-banner">⚠️ {{ error }}</div>

    <!-- Generate Form -->
    <div class="card generate-card">
      <h2>🔑 Generate License Key</h2>
      <div class="form-grid">
        <label>
          Tipe Pemilihan
          <select v-model="form.election_type">
            <option v-for="t in ELECTION_TYPES" :key="t" :value="t">{{ t }}</option>
          </select>
        </label>
        <label>
          Nama Program (opsional)
          <input v-model="form.program_name" placeholder="Pilkada Bandung 2027" />
        </label>
        <label>
          Max Pemilih
          <input type="number" v-model="form.max_voters" min="100" step="1000" />
        </label>
        <label>
          Wargame Tokens
          <input type="number" v-model="form.wargame_tokens" min="0" max="999" />
        </label>
        <label>
          Masa Aktif (hari)
          <input type="number" v-model="form.validity_days" min="30" max="3650" />
        </label>
        <label>
          Jumlah Key (1–50)
          <input type="number" v-model="form.quantity" min="1" max="50" />
        </label>
      </div>
      <button class="btn-generate" @click="generateLicense" :disabled="generating">
        {{ generating ? '⏳ Generating...' : '🔑 Generate License Key' }}
      </button>
    </div>

    <!-- License List -->
    <div class="section">
      <div class="list-header">
        <h2>Daftar License Key</h2>
        <div class="filters">
          <label class="filter-label" for="filter-used">
            <span>Filter</span>
            <select id="filter-used" v-model="filterUsed" @change="load">
              <option value="all">Semua</option>
              <option value="false">Belum Digunakan</option>
              <option value="true">Sudah Digunakan</option>
            </select>
          </label>
          <button class="btn-refresh" @click="load">🔄</button>
        </div>
      </div>

      <div v-if="loading" class="loading">Memuat...</div>
      <div v-else-if="licenses.length === 0" class="empty">Tidak ada license key yang cocok dengan filter.</div>
      <div v-else class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>License Key</th>
              <th>Program</th>
              <th>Tipe</th>
              <th>Max Pemilih</th>
              <th>Tokens</th>
              <th>Status</th>
              <th>Dibuat</th>
              <th>Dipakai</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="lic in licenses" :key="lic.id">
              <td>
                <code class="license-key">{{ lic.license_key }}</code>
              </td>
              <td>{{ lic.program_name || '-' }}</td>
              <td><code>{{ lic.election_type }}</code></td>
              <td>{{ (lic.max_voters || 0).toLocaleString('id-ID') }}</td>
              <td>{{ lic.wargame_tokens }}</td>
              <td>
                <span :class="['status-badge', lic.is_used ? 'used' : 'unused']">
                  {{ lic.is_used ? '✅ Terpakai' : '⏳ Tersedia' }}
                </span>
              </td>
              <td>{{ formatDate(lic.created_at) }}</td>
              <td>{{ formatDate(lic.used_at) }}</td>
              <td>
                <button
                  v-if="!lic.is_used"
                  class="btn-copy"
                  @click="copyKey(lic.license_key)"
                >
                  {{ copyFeedback[lic.license_key] ? '✅' : '📋' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { color: var(--muted); font-size: 14px; }

.success-banner {
  background: rgba(16, 185, 129, 0.1); border: 1px solid rgba(16, 185, 129, 0.3);
  color: var(--success); border-radius: 8px; padding: 10px 16px; margin-bottom: 16px; font-size: 14px;
}
.error-banner {
  background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3);
  color: var(--danger); border-radius: 8px; padding: 10px 16px; margin-bottom: 16px; font-size: 14px;
}

.card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 32px;
}
.card h2 { font-size: 16px; margin-bottom: 16px; }

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 14px;
  margin-bottom: 16px;
}

.form-grid label {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 12px;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.4px;
}
.form-grid input, .form-grid select { font-size: 14px; width: 100%; }

.btn-generate {
  background: var(--accent);
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
}
.btn-generate:hover:not(:disabled) { background: #2563eb; }

.list-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.list-header h2 { font-size: 16px; }
.filters { display: flex; gap: 8px; align-items: center; }
.filters select { font-size: 13px; padding: 6px 10px; }
.btn-refresh { background: var(--bg); border: 1px solid var(--border); color: var(--text); padding: 6px 10px; border-radius: 6px; font-size: 14px; }

.table-wrap { background: var(--card); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; }

.license-key { font-size: 11px; font-family: monospace; letter-spacing: 0.5px; }

.status-badge {
  font-size: 11px;
  font-weight: 600;
  padding: 3px 8px;
  border-radius: 20px;
  white-space: nowrap;
}
.status-badge.used { background: rgba(16, 185, 129, 0.15); color: var(--success); }
.status-badge.unused { background: rgba(245, 158, 11, 0.15); color: var(--warning); }

.btn-copy {
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 13px;
}
.btn-copy:hover { background: var(--border); }

.loading, .empty {
  padding: 40px;
  text-align: center;
  color: var(--muted);
  background: var(--card);
  border-radius: 12px;
  border: 1px dashed var(--border);
}
</style>
