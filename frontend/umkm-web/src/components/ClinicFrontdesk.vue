<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'

const queueItems = ref<any[]>([])
const settings = ref({
  queue_type: 'sequential',
  slot_duration_minutes: 30,
  is_active: true
})

const loading = ref(true)

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
      await fetchQueue()
    }
  } catch (err) {
    console.error('Failed to call patient', err)
  }
}

const cancelAppointment = async (id: string) => {
  try {
    const res = await api.cancelClinicAppointment(id, 'frontdesk')
    if (res && res.success) {
      await fetchQueue()
    }
  } catch (err) {
    console.error('Failed to cancel appointment', err)
  }
}

onMounted(() => {
  fetchQueue()
  fetchSettings()
})
</script>

<template>
  <div class="p-6" style="min-height: 100vh;">
    <!-- Header -->
    <div class="flex justify-between items-center mb-8">
      <h1 class="text-3xl font-bold">Dashboard Klinik</h1>
      <button @click="saveSettings" class="btn btn-primary">
        💾 Simpan Pengaturan
      </button>
    </div>

    <!-- Settings Panel -->
    <div class="glass-card animate-fade-in mb-8" style="padding: 1.5rem;">
      <h2 class="text-xl font-bold mb-4">⚙️ Pengaturan Antrean</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div>
          <label class="block text-sm font-semibold mb-2">Jenis Antrean</label>
          <select v-model="settings.queue_type" class="form-control">
            <option value="sequential">🔢 Nomor Urut (A-001, A-002, ...)</option>
            <option value="timeslot">🕐 Slot Waktu (09:00, 10:30, ...)</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-semibold mb-2">Durasi Slot (menit)</label>
          <input v-model="settings.slot_duration_minutes" type="number" min="15" max="120" class="form-control" />
        </div>
        <div class="flex items-end">
          <label class="flex items-center gap-3 cursor-pointer">
            <input v-model="settings.is_active" type="checkbox" class="w-5 h-5" />
            <span class="font-semibold">Klinik Aktif</span>
          </label>
        </div>
      </div>
    </div>

    <!-- Queue List -->
    <div class="glass-card animate-fade-in" style="padding: 0; overflow: hidden;">
      <div style="padding: 1.25rem; border-bottom: 1px solid var(--glass-border);">
        <h2 class="text-xl font-bold">📋 Daftar Antrean ({{ queueItems.length }} orang)</h2>
      </div>
      <div v-if="loading" class="text-center" style="padding: 2.5rem;">⏳ Memuat...</div>
      <div v-else-if="queueItems.length === 0" class="text-center" style="padding: 2.5rem; opacity: 0.6;">
        ✨ Tidak ada antrean hari ini.
      </div>
      <table v-else style="width: 100%; text-align: left;">
        <thead style="background: rgba(255,255,255,0.05);">
          <tr>
            <th class="font-semibold" style="padding: 1rem;">No. Antrian</th>
            <th class="font-semibold" style="padding: 1rem;">Nama Pasien</th>
            <th class="font-semibold" style="padding: 1rem;">No. WA</th>
            <th class="font-semibold" style="padding: 1rem;">Status</th>
            <th class="font-semibold" style="padding: 1rem;">Aksi</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in queueItems" :key="item.id" style="border-top: 1px solid var(--glass-border);">
            <td style="padding: 1rem; font-weight: bold; font-size: 1.1rem;">{{ item.queue_number }}</td>
            <td style="padding: 1rem;">{{ item.patient_name }}</td>
            <td style="padding: 1rem; opacity: 0.7;">{{ item.patient_phone }}</td>
            <td style="padding: 1rem;">
              <span :class="item.status === 'waiting' ? 'badge-waiting' : 'badge-called'" class="status-badge">
                {{ item.status === 'waiting' ? '🟡 Menunggu' : '🟢 Dipanggil' }}
              </span>
            </td>
            <td style="padding: 1rem;">
              <div class="flex gap-2">
                <button v-if="item.status === 'waiting'" @click="callNext(item.id)"
                        class="btn btn-primary" style="font-size: 0.85rem; padding: 0.4rem 0.8rem;">
                  📞 Panggil
                </button>
                <button @click="cancelAppointment(item.id)"
                        class="btn btn-secondary" style="font-size: 0.85rem; padding: 0.4rem 0.8rem; color: #f87171; border-color: rgba(248,113,113,0.3);">
                  ❌ Batal
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.status-badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.85rem;
  font-weight: 600;
}
.badge-waiting {
  background: rgba(251, 191, 36, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(251, 191, 36, 0.3);
}
.badge-called {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
  border: 1px solid rgba(34, 197, 94, 0.3);
}
</style>
