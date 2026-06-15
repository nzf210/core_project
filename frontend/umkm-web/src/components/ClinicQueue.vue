<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '../api'

const queueItems = ref<any[]>([])
const loading = ref(true)
const error = ref('')

// Tiga nomor terakhir yang dipanggil
const lastCalled = ref<any[]>([])
const currentCalling = ref<any | null>(null)

let pollInterval: any

const fetchQueue = async () => {
  try {
    const res = await api.getClinicQueue()
    if (res && res.success) {
      // Split into 'waiting' and 'in_progress'
      const all = res.data || []
      
      const inProgress = all.filter((x: any) => x.status === 'in_progress')
      const waiting = all.filter((x: any) => x.status === 'waiting')
      
      // Update display lists
      if (inProgress.length > 0) {
        // Ambil yang paling baru dipanggil
        currentCalling.value = inProgress[inProgress.length - 1]
        
        // Simpan history yang sebelumnya
        if (inProgress.length > 1) {
          lastCalled.value = inProgress.slice(0, inProgress.length - 1).reverse().slice(0, 3)
        }
      } else {
        currentCalling.value = null
      }
      
      queueItems.value = waiting
    } else {
      error.value = res.message || 'Gagal memuat antrean'
    }
  } catch (err: any) {
    error.value = err.message || 'Kesalahan jaringan'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchQueue()
  // Auto-refresh setiap 5 detik
  pollInterval = setInterval(fetchQueue, 5000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <div class="h-screen bg-gray-50 flex flex-col font-sans overflow-hidden">
    <!-- Header -->
    <header class="bg-blue-600 text-white p-6 shadow-md flex justify-between items-center">
      <div>
        <h1 class="text-4xl font-extrabold tracking-tight">Klinik Sehat UMKM</h1>
        <p class="text-lg opacity-80 mt-1">Daftar Antrean Pasien</p>
      </div>
      <div class="text-right">
        <div class="text-3xl font-mono">{{ new Date().toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }) }}</div>
        <div class="text-sm opacity-80">{{ new Date().toLocaleDateString('id-ID', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' }) }}</div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 flex p-6 gap-6">
      <!-- Left Column (Current Calling) -->
      <div class="w-1/2 flex flex-col gap-6">
        <div class="bg-white rounded-2xl shadow-lg border border-gray-100 p-8 flex-1 flex flex-col justify-center items-center text-center">
          <h2 class="text-3xl font-semibold text-gray-500 mb-4 uppercase tracking-widest">Nomor Antrean</h2>
          
          <div v-if="currentCalling" class="w-full">
            <div class="text-9xl font-black text-blue-600 my-8 py-8 bg-blue-50 rounded-xl border-4 border-blue-100 w-full shadow-inner animate-pulse">
              {{ currentCalling.queue_number }}
            </div>
            <h3 class="text-5xl font-bold text-gray-800">{{ currentCalling.patient_name }}</h3>
            <p class="text-2xl text-green-600 font-semibold mt-6 flex items-center justify-center gap-3">
              <span class="w-4 h-4 bg-green-500 rounded-full animate-ping"></span>
              SILAKAN MENUJU RUANG DOKTER
            </p>
          </div>
          
          <div v-else class="text-gray-400 flex flex-col items-center">
            <div class="text-8xl mb-6">☕</div>
            <p class="text-2xl">Belum ada pasien yang dipanggil.</p>
          </div>
        </div>

        <!-- History -->
        <div class="bg-white rounded-xl shadow border border-gray-100 p-6">
          <h3 class="text-xl font-bold text-gray-700 mb-4">Panggilan Sebelumnya</h3>
          <div class="flex gap-4">
            <div v-for="item in lastCalled" :key="item.id" class="flex-1 bg-gray-50 rounded-lg p-4 text-center border">
              <div class="text-3xl font-bold text-gray-800">{{ item.queue_number }}</div>
              <div class="text-sm text-gray-500 truncate mt-1">{{ item.patient_name }}</div>
            </div>
            <div v-if="lastCalled.length === 0" class="text-gray-400 italic text-center w-full">Kosong</div>
          </div>
        </div>
      </div>

      <!-- Right Column (Waiting List) -->
      <div class="w-1/2 bg-white rounded-2xl shadow-lg border border-gray-100 flex flex-col overflow-hidden">
        <div class="bg-gray-800 text-white p-4">
          <h2 class="text-2xl font-bold flex justify-between">
            <span>Daftar Menunggu</span>
            <span class="bg-white text-gray-800 px-3 py-1 rounded-full text-lg">{{ queueItems.length }} Pasien</span>
          </h2>
        </div>
        
        <div class="flex-1 overflow-y-auto p-4">
          <div v-if="loading" class="text-center py-10 text-gray-500">Memuat data...</div>
          <div v-else-if="queueItems.length === 0" class="text-center py-20 text-gray-400">
            <div class="text-6xl mb-4">🩺</div>
            <p class="text-xl">Antrean kosong. Dokter siap melayani.</p>
          </div>
          
          <div v-else class="flex flex-col gap-3">
            <div v-for="(item, index) in queueItems" :key="item.id" 
                 class="flex items-center justify-between p-5 rounded-xl border-l-4"
                 :class="index === 0 ? 'bg-blue-50 border-blue-500 shadow-sm' : 'bg-gray-50 border-gray-300'">
              
              <div class="flex items-center gap-6">
                <div class="text-4xl font-bold" :class="index === 0 ? 'text-blue-600' : 'text-gray-700'">
                  {{ item.queue_number }}
                </div>
                <div>
                  <div class="text-2xl font-semibold text-gray-800">{{ item.patient_name }}</div>
                  <div class="text-gray-500 mt-1" v-if="item.scheduled_time">
                    Estimasi: {{ new Date(item.scheduled_time).toLocaleTimeString('id-ID', {hour: '2-digit', minute:'2-digit'}) }}
                  </div>
                </div>
              </div>
              
              <div v-if="index === 0" class="text-blue-600 font-bold bg-blue-100 px-4 py-2 rounded-lg">Selanjutnya</div>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
