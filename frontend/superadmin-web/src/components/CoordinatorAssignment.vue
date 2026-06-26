<template>
  <div class="page">
    <h1>Hierarchical Coordinator Assignment</h1>
    <p class="subtitle">Assign koordinator kampanye berdasarkan level hierarki (F046).</p>

    <div class="grid">
      <!-- Form panel -->
      <div class="card form-panel">
        <h2>Assign Koordinator</h2>

        <div class="field">
          <label for="campaign">Campaign</label>
          <select id="campaign" v-model="selectedCampaign" @change="fetchCoordinators">
            <option value="">-- Pilih Campaign --</option>
            <option v-for="c in campaigns" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </div>

        <div class="field">
          <label for="nik">NIK (Citizen ID)</label>
          <input id="nik" v-model="nikInput" type="text" placeholder="327xxxxxxxxxxxxxx" maxlength="16" />
        </div>

        <div class="field">
          <label for="level">Level Koordinator</label>
          <select id="level" v-model="selectedLevel" @change="fetchRegions">
            <option value="">-- Pilih Level --</option>
            <option value="korprov">Koordinator Provinsi (Korprov)</option>
            <option value="korKab">Koordinator Kabupaten (KorKab)</option>
            <option value="korKec">Koordinator Kecamatan (KorKec)</option>
            <option value="korKades">Koordinator Desa (KorKades)</option>
            <option value="saksi_tps">Saksi TPS</option>
          </select>
        </div>

        <div class="field" v-if="selectedLevel">
          <label for="region">Wilayah</label>
          <select id="region" v-model="selectedRegion">
            <option value="">-- Pilih Wilayah --</option>
            <option v-for="r in regions" :key="r.id" :value="r.id">{{ r.name }}</option>
          </select>
        </div>

        <button class="btn-assign" @click="assignCoordinator" :disabled="!canAssign || assigning">
          {{ assigning ? '⏳ Menyimpan...' : '✅ Assign Koordinator' }}
        </button>

        <div v-if="statusMsg" :class="['status', statusSuccess ? 'status-ok' : 'status-err']">
          {{ statusMsg }}
        </div>
      </div>

      <!-- Coordinator list -->
      <div class="card list-panel">
        <div class="list-header">
          <h2>Koordinator Terdaftar</h2>
          <button class="btn-refresh" @click="fetchCoordinators" :disabled="!selectedCampaign">🔄</button>
        </div>

        <div v-if="!selectedCampaign" class="empty">Pilih campaign terlebih dahulu</div>
        <div v-else-if="coordinators.length === 0" class="empty">Belum ada koordinator</div>
        <table v-else>
          <thead>
            <tr>
              <th>Level</th>
              <th>NIK</th>
              <th>Wilayah</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in coordinators" :key="c.id">
              <td><code>{{ formatLevel(c.level) }}</code></td>
              <td>{{ maskNIK(c.nik) }}</td>
              <td>{{ c.region_name || c.region_id || '-' }}</td>
              <td>
                <span :class="['badge', c.is_active ? 'badge-active' : 'badge-inactive']">
                  {{ c.is_active ? 'Aktif' : 'Nonaktif' }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { request } from '../api/client'

const campaigns = ref<any[]>([])
const coordinators = ref<any[]>([])
const regions = ref<any[]>([])
const nikInput = ref('')
const selectedCampaign = ref('')
const selectedLevel = ref('')
const selectedRegion = ref('')
const statusMsg = ref('')
const statusSuccess = ref(true)
const assigning = ref(false)

const canAssign = computed(() => {
  return selectedCampaign.value && nikInput.value.length >= 10 && selectedLevel.value && selectedRegion.value
})

function formatLevel(level: string): string {
  const map: Record<string, string> = {
    korprov: 'Korprov',
    korKab: 'KorKab',
    korKec: 'KorKec',
    korKades: 'KorKades',
    saksi_tps: 'Saksi TPS'
  }
  return map[level] || level
}

function maskNIK(nik: string): string {
  if (!nik) return '-'
  return nik.substring(0, 4) + 'xxxxxxxx' + nik.substring(nik.length - 4)
}

async function fetchCampaigns() {
  try {
    const res = await request('/api/campaigns')
    campaigns.value = res.data || []
  } catch (e) {
    console.error('Failed to fetch campaigns', e)
  }
}

async function fetchRegions() {
  if (!selectedLevel.value) return
  selectedRegion.value = ''
  try {
    const endpointMap: Record<string, string> = {
      korprov: '/api/regions/provinces',
      korKab: '/api/regions/regencies',
      korKec: '/api/regions/districts',
      korKades: '/api/regions/villages',
      saksi_tps: '/api/regions/tps',
    }
    const endpoint = endpointMap[selectedLevel.value] || '/api/regions/provinces'
    const res = await request(endpoint)
    regions.value = res.data || res || []
  } catch (e) {
    console.error('Failed to fetch regions', e)
  }
}

async function fetchCoordinators() {
  if (!selectedCampaign.value) return
  try {
    const res = await request(`/api/coordinator/list?campaign_id=${selectedCampaign.value}`)
    coordinators.value = res.data || []
  } catch (e) {
    console.error('Failed to fetch coordinators', e)
  }
}

async function assignCoordinator() {
  statusMsg.value = ''
  assigning.value = true
  try {
    const res = await request('/api/coordinator/assign', {
      method: 'POST',
      body: JSON.stringify({
        citizen_nik: nikInput.value,
        coordinator_level: selectedLevel.value,
        region_id: selectedRegion.value,
        campaign_id: selectedCampaign.value,
      }),
    })
    statusSuccess.value = true
    statusMsg.value = res.message || 'Koordinator berhasil di-assign'
    nikInput.value = ''
    await fetchCoordinators()
  } catch (e: any) {
    statusSuccess.value = false
    statusMsg.value = e.message || 'Gagal assign koordinator'
  } finally {
    assigning.value = false
    setTimeout(() => statusMsg.value = '', 4000)
  }
}

onMounted(() => {
  fetchCampaigns()
})
</script>

<style scoped>
h1 { font-size: 24px; margin-bottom: 4px; }
.subtitle { color: var(--muted); font-size: 14px; margin-bottom: 24px; }

.grid {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 20px;
  align-items: start;
}

.card {
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 20px;
}
.card h2 { font-size: 15px; font-weight: 700; margin-bottom: 16px; }

.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-bottom: 14px;
}
.field label { font-size: 12px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.4px; }
.field input, .field select { width: 100%; font-size: 14px; }

.btn-assign {
  width: 100%;
  padding: 10px;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  margin-top: 4px;
}
.btn-assign:hover:not(:disabled) { background: #2563eb; }

.status {
  margin-top: 12px;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 13px;
}
.status-ok { background: rgba(16, 185, 129, 0.1); color: var(--success); border: 1px solid rgba(16, 185, 129, 0.3); }
.status-err { background: rgba(239, 68, 68, 0.1); color: var(--danger); border: 1px solid rgba(239, 68, 68, 0.3); }

/* List panel */
.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}
.btn-refresh {
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--text);
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 14px;
}
.btn-refresh:hover:not(:disabled) { background: var(--border); }

.empty { padding: 30px; text-align: center; color: var(--muted); font-size: 13px; }

.badge {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 20px;
}
.badge-active { background: rgba(16, 185, 129, 0.15); color: var(--success); }
.badge-inactive { background: rgba(148, 163, 184, 0.1); color: var(--muted); }

@media (max-width: 768px) {
  .grid { grid-template-columns: 1fr; }
}
</style>