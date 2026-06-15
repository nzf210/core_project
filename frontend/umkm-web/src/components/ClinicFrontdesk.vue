<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api'

const queueItems = ref<any[]>([])
const settings = ref({
  queue_type: 'sequential',
  slot_duration_minutes: 30,
  is_active: true
})

const loading = ref(true)
const callingNumber = ref('')

let pollInterval: any

const fetchQueue = async () => {
  try {
    const res = await api.getClinicQueue()
    if (res && res.success) {
      queueItems.value = res.data || []
    }
  } catch (err) {
    console.error('Failed to fetch queue', err)
  } finally {
    loading.value = false
  }
}

const fetchSettings = async () => {
  try {
    const res = await api.getClinicSettings()
    if (res && res.success) {
      settings.value = res.data
    }
  } catch (err) {
    console.error('Failed to fetch settings', err)
  }
}

const saveSettings = async () => {
  try {
    const res = await api.updateClinicSettings(settings.value)
    if (res && res.success) {
      alert('Pengaturan tersimpan!')
    }
  } catch (err) {
    alert('Gagal menyimpan pengaturan')
  }
}

const callNext = async (id: string) => {
  try {
    const res = await api.callClinicAppointment(id)
    if (res && res.success) {
      callingNumber.value = id
      await fetchQueue()
    }
  } catch (err) {
    alert('Gagal memanggil antrian')
  }
}

const cancelAppointment = async (id: string, performedBy: string = 'clinic') => {
  if (!confirm('Yakin mau membatalkan antrian ini?')) return
  try {
    const res = await api.cancelClinicAppointment(id, performedBy)
    if (res && res.success) {
      await fetchQueue()
    }
  } catch (err) {
    alert('Gagal membatalkan antrian')
  }
}

onMounted(() => {
  fetchQueue()
  fetchSettings()
  pollInterval = setInterval(fetchQueue, 3000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <div class="p-6 bg-gray-50 min-h-screen">
    <!-- Header -->
    <div class="flex justify-between items-center mb-8">
      <h1 class="text-3xl font-bold text-gray-800">Dashboard Klinik</h1>
      <button @click="saveSettings" class="px-6 py-3 bg-blue-600 text-white rounded-xl font-semibold hover:bg-blue-700 transition">
        Simpan Pengaturan
      </button>
    </div>

    <!-- Settings Panel -->
    <div class="bg-white rounded-xl shadow border p-6 mb-8">
      <h2 class="text-xl font-bold text-gray-700 mb-4">Pengaturan Antrean</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div>
          <label class="block text-sm font-semibold text-gray-600 mb-2">Jenis Antrean</label>
          <select v-model="settings.queue_type" class="w-full px-4 py-3 border rounded-lg">
            <option value="sequential">Nomor Urut (A-001, A-002, ...)</option>
            <option value="timeslot">Slot Waktu (09:00, 10:30, ...)</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-semibold text-gray-600 mb-2">Durasi Slot (menit)</label>
          <input v-model="settings.slot_duration_minutes" type="number" min="15" max="120" class="w-full px-4 py-3 border rounded-lg" />
        </div>
        <div class="flex items-end">
          <label class="flex items-center gap-3">
            <input v-model="settings.is_active" type="checkbox" class="w-5 h-5" />
            <span class="font-semibold">Klinik Aktif</span>
          </label>
        </div>
      </div>
    </div>

    <!-- Queue List -->
    <div class="bg-white rounded-xl shadow border overflow-hidden">
      <div class="p-5 border-b">
        <h2 class="text-xl font-bold text-gray-700">Daftar Antrean ({{ queueItems.length }} orang)</h2>
      </div>
      <div v-if="loading" class="p-10 text-center">Memuat...</div>
      <div v-else-if="queueItems.length === 0" class="p-10 text-center text-gray-500">
        Tidak ada antrean hari ini.
      </div>
      <table v-else class="w-full text-left">
        <thead class="bg-gray-50">
          <tr>
            <th class="p-4 font-semibold">No. Antrian</th>
            <th class="p-4 font-semibold">Nama Pasien</th>
            <th class="p-4 font-semibold">No. WA</th>
            <th class="p-4 font-semibold">Status</th>
            <th class="p-4 font-semibold">Aksi</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in queueItems" :key="item.id" class="border-t hover:bg-gray-50">
            <td class="p-4 font-bold text-lg">{{ item.queue_number }}</td>
            <td class="p-4">{{ item.patient_name }}</td>
            <td class="p-4 text-gray-600">{{ item.patient_phone }}</td>
            <td class="p-4">
              <span :class="item.status === 'waiting' ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700'" 
                    class="px-3 py-1 rounded-full text-sm font-semibold">
                {{ item.status === 'waiting' ? 'Menunggu' : 'Dipanggil' }}
              </span>
            </td>
            <td class="p-4">
              <div class="flex gap-2">
                <button v-if="item.status === 'waiting'" @click="callNext(item.id)" 
                        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm">
                  Panggil
                </button>
                <button @click="cancelAppointment(item.id)" 
                        class="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 text-sm">
                  Batal
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>