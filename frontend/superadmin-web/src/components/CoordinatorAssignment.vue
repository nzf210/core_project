<template>
  <div class="p-4">
    <h2 class="text-xl font-bold mb-4">Hierarchical Coordinator Assignment</h2>
    
    <!-- Campaign Selection -->
    <div class="mb-4">
      <label class="block text-sm font-medium mb-1">Campaign</label>
      <select v-model="selectedCampaign" @change="fetchCoordinators" class="border p-2 w-full">
        <option value="">Select Campaign</option>
        <option v-for="c in campaigns" :key="c.id" :value="c.id">{{ c.name }}</option>
      </select>
    </div>

    <!-- NIK Input -->
    <div class="mb-4">
      <label class="block text-sm font-medium mb-1">NIK (Citizen ID)</label>
      <input v-model="nikInput" type="text" placeholder="327xxxxxx" class="border p-2 w-full" maxlength="16" />
    </div>

    <!-- Level Selection -->
    <div class="mb-4">
      <label class="block text-sm font-medium mb-1">Coordinator Level</label>
      <select v-model="selectedLevel" class="border p-2 w-full">
        <option value="korprov">Provincial Coordinator (Korprov)</option>
        <option value="korKab">Regency Coordinator (KorKab)</option>
        <option value="korKec">District Coordinator (KorKec)</option>
        <option value="korKades">Village Coordinator (KorKades)</option>
        <option value="saksi_tps">TPS Witness (Saksi)</option>
      </select>
    </div>

    <!-- Region Selection (dynamic based on level) -->
    <div class="mb-4" v-if="selectedLevel">
      <label class="block text-sm font-medium mb-1">Region</label>
      <select v-model="selectedRegion" class="border p-2 w-full">
        <option value="">Select Region</option>
        <option v-for="r in regions" :key="r.id" :value="r.id">{{ r.name }}</option>
      </select>
    </div>

    <!-- Assign Button -->
    <button @click="assignCoordinator" :disabled="!canAssign" class="bg-blue-600 text-white px-4 py-2 rounded disabled:opacity-50">
      Assign Coordinator
    </button>

    <!-- Status Message -->
    <p v-if="statusMsg" class="mt-2" :class="statusSuccess ? 'text-green-600' : 'text-red-600'">{{ statusMsg }}</p>

    <!-- Coordinator List -->
    <div class="mt-6">
      <h3 class="font-medium mb-2">Current Coordinators</h3>
      <table class="w-full border" v-if="coordinators.length > 0">
        <thead>
          <tr class="bg-gray-100">
            <th class="p-2 text-left">Level</th>
            <th class="p-2 text-left">NIK</th>
            <th class="p-2 text-left">Region</th>
            <th class="p-2 text-left">Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in coordinators" :key="c.id">
            <td class="p-2">{{ formatLevel(c.level) }}</td>
            <td class="p-2">{{ maskNIK(c.nik) }}</td>
            <td class="p-2">{{ c.region_name || c.region_id }}</td>
            <td class="p-2">
              <span :class="c.is_active ? 'text-green-600' : 'text-gray-400'">
                {{ c.is_active ? 'Active' : 'Inactive' }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else-if="selectedCampaign" class="text-gray-500">No coordinators assigned yet.</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const campaigns = ref([])
const coordinators = ref([])
const regions = ref([])
const nikInput = ref('')
const selectedCampaign = ref('')
const selectedLevel = ref('')
const selectedRegion = ref('')
const statusMsg = ref('')
const statusSuccess = ref(true)

const canAssign = computed(() => {
  return selectedCampaign.value && nikInput.value && selectedLevel.value && selectedRegion.value
})

function formatLevel(level) {
  const map = {
    korprov: 'Korprov',
    korKab: 'KorKab',
    korKec: 'KorKec',
    korKades: 'KorKades',
    saksi_tps: 'Saksi TPS'
  }
  return map[level] || level
}

function maskNIK(nik) {
  if (!nik) return '-'
  return nik.substring(0, 4) + 'xxxx' + nik.substring(nik.length - 3)
}

async function fetchCampaigns() {
  try {
    const res = await fetch('/api/campaigns')
    const data = await res.json()
    campaigns.value = data.data || []
  } catch (e) {
    console.error('Failed to fetch campaigns', e)
  }
}

async function fetchRegions() {
  if (!selectedLevel.value) return
  try {
    const endpoint = {
      korprov: '/api/regions/provinces',
      korKab: '/api/regions/regencies',
      korKec: '/api/regions/districts',
      korKades: '/api/regions/villages',
      saksi_tps: '/api/regions/tps'
    }[selectedLevel.value]
    const res = await fetch(endpoint || '/api/regions/provinces')
    const data = await res.json()
    regions.value = data.data || data || []
  } catch (e) {
    console.error('Failed to fetch regions', e)
  }
}

async function fetchCoordinators() {
  if (!selectedCampaign.value) return
  try {
    const res = await fetch(`/api/coordinator/list?campaign_id=${selectedCampaign.value}`)
    const data = await res.json()
    coordinators.value = data.data || []
  } catch (e) {
    console.error('Failed to fetch coordinators', e)
  }
}

async function assignCoordinator() {
  statusMsg.value = ''
  try {
    const res = await fetch('/api/coordinator/assign', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        citizen_nik: nikInput.value,
        coordinator_level: selectedLevel.value,
        region_id: selectedRegion.value,
        campaign_id: selectedCampaign.value
      })
    })
    const data = await res.json()
    if (res.ok) {
      statusSuccess.value = true
      statusMsg.value = data.message || 'Coordinator assigned successfully'
      fetchCoordinators()
    } else {
      statusSuccess.value = false
      statusMsg.value = data.message || 'Failed to assign coordinator'
    }
  } catch (e) {
    statusSuccess.value = false
    statusMsg.value = 'Network error'
  }
}

onMounted(() => {
  fetchCampaigns()
})
</script>

<style scoped>
/* Add custom styles if needed */
</style>