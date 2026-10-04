<template>
  <div class="p-6 max-w-7xl mx-auto space-y-6">
    <!-- Header & Quick Stats -->
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <span>🔧</span> Surat Perintah Kerja (SPK) & Servis
        </h1>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Pantau progres perbaikan unit pelanggan, penugasan teknisi, kalkulasi biaya akhir, dan notifikasi siap ambil
        </p>
      </div>
      <div class="flex items-center gap-3">
        <button
          @click="showNewOrderModal = true"
          class="px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white font-medium rounded-lg shadow flex items-center gap-2 transition"
        >
          <span>➕</span> Terbitkan SPK Baru
        </button>
      </div>
    </div>

    <!-- Status Tabs / Counters -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-7 gap-3">
      <button
        v-for="s in statusOptions"
        :key="s.key"
        @click="selectedStatus = s.key; fetchOrders()"
        :class="[
          'p-3 rounded-xl border text-left transition flex flex-col justify-between',
          selectedStatus === s.key
            ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20 text-indigo-700 dark:text-indigo-300 ring-2 ring-indigo-500'
            : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700/50'
        ]"
      >
        <span class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ s.label }}</span>
        <div class="flex items-center justify-between mt-2">
          <span class="text-xl">{{ s.icon }}</span>
          <span class="text-lg font-bold">{{ countByStatus(s.key) }}</span>
        </div>
      </button>
    </div>

    <!-- Search & Filter Bar -->
    <div class="flex flex-col sm:flex-row gap-3 bg-white dark:bg-gray-800 p-4 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm">
      <div class="relative flex-1">
        <label for="spkSearchInput" class="sr-only">Cari Tiket Servis</label>
        <span class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-gray-400">🔍</span>
        <input
          id="spkSearchInput"
          v-model="searchQuery"
          @input="debounceSearch"
          type="text"
          placeholder="Cari nomor SPK (SPK-...), nama unit, plat/identitas, nama pelanggan..."
          class="w-full pl-9 pr-4 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-700/50 text-gray-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>
      <button
        @click="fetchOrders"
        class="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 transition flex items-center justify-center gap-1"
      >
        <span>🔄</span> Refresh
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="text-center py-12 text-gray-500 dark:text-gray-400">
      <span class="inline-block animate-spin text-2xl mb-2">🔄</span>
      <p class="text-sm">Memuat data SPK servis...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredOrders.length === 0" class="text-center py-12 bg-white dark:bg-gray-800 rounded-xl border border-dashed border-gray-300 dark:border-gray-700 p-8">
      <span class="text-4xl block mb-2">🔧</span>
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">Tidak ada pekerjaan servis</h3>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-sm mx-auto">
        Belum ada SPK dengan status ini. Klik tombol "Terbitkan SPK Baru" untuk mencatat perbaikan unit pelanggan.
      </p>
    </div>

    <!-- SPK Cards Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="order in filteredOrders"
        :key="order.id"
        class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden flex flex-col justify-between"
      >
        <div class="p-4 space-y-3">
          <!-- Card Header: Unit & Status -->
          <div class="flex justify-between items-start">
            <div>
              <div class="flex items-center gap-2">
                <span class="font-bold text-gray-900 dark:text-white text-base">
                  {{ order.unit_name }}
                </span>
                <span v-if="order.unit_identifier" class="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 font-mono text-xs rounded">
                  {{ order.unit_identifier }}
                </span>
              </div>
              <span class="text-xs font-mono text-gray-500 dark:text-gray-400">
                {{ order.order_no }}
              </span>
            </div>
            <span :class="['px-2.5 py-1 rounded-full text-xs font-semibold', statusBadgeClass(order.status)]">
              {{ statusLabel(order.status) }}
            </span>
          </div>

          <!-- Customer Info & WA link -->
          <div class="flex justify-between items-center text-xs text-gray-600 dark:text-gray-300 bg-gray-50 dark:bg-gray-700/30 p-2 rounded-lg">
            <div>
              <p class="font-semibold text-gray-900 dark:text-white">👤 {{ order.customer_name }}</p>
              <p class="text-gray-500 dark:text-gray-400 font-mono">{{ order.customer_phone }}</p>
            </div>
            <a
              :href="getWhatsAppLink(order)"
              target="_blank"
              class="px-2.5 py-1 bg-emerald-100 hover:bg-emerald-200 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200 rounded text-xs font-semibold flex items-center gap-1 transition"
            >
              <span>💬</span> Chat WA
            </a>
          </div>

          <!-- Complaint & Technician -->
          <div class="space-y-1.5 text-xs">
            <div>
              <span class="text-gray-500 dark:text-gray-400">Keluhan:</span>
              <p class="text-gray-800 dark:text-gray-200 font-medium bg-red-50 dark:bg-red-900/20 p-2 rounded border border-red-100 dark:border-red-900/40">
                ⚠️ {{ order.complaint }}
              </p>
            </div>
            <div v-if="order.technician_name" class="flex items-center gap-1 text-gray-600 dark:text-gray-300">
              <span>👨‍🔧 Teknisi:</span>
              <span class="font-semibold text-gray-900 dark:text-white">{{ order.technician_name }}</span>
            </div>
            <div v-if="order.notes" class="text-gray-500 dark:text-gray-400 italic">
              📝 {{ order.notes }}
            </div>
          </div>

          <!-- Cost & Payment Status -->
          <div class="flex justify-between items-center pt-2 border-t border-gray-100 dark:border-gray-700 text-xs">
            <div>
              <span class="text-gray-500 dark:text-gray-400">
                {{ order.final_cost > 0 ? 'Biaya Akhir:' : 'Estimasi Biaya:' }}
              </span>
              <p class="text-sm font-bold text-indigo-600 dark:text-indigo-400">
                {{ formatRupiah(order.final_cost > 0 ? order.final_cost : order.estimated_cost) }}
              </p>
            </div>
            <span
              :class="[
                'px-2 py-0.5 rounded text-[11px] font-semibold',
                order.is_paid
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                  : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
              ]"
            >
              {{ order.is_paid ? '✅ Lunas' : '⏳ Belum Bayar' }}
            </span>
          </div>
        </div>

        <!-- Card Footer / Actions -->
        <div class="px-4 py-3 bg-gray-50 dark:bg-gray-700/50 border-t border-gray-100 dark:border-gray-700 flex flex-wrap items-center justify-between gap-2">
          <!-- Next Status Step Button -->
          <button
            v-if="getNextStatus(order.status)"
            :disabled="updatingId === order.id"
            @click="handleAdvanceStatus(order)"
            class="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg text-xs font-semibold transition flex items-center gap-1 shadow-sm"
          >
            <span>{{ getNextStatus(order.status)!.icon }}</span>
            <span>{{ getNextStatus(order.status)!.label }}</span>
          </button>

          <!-- Pay Button if not paid -->
          <button
            v-if="!order.is_paid && order.status !== 'cancelled'"
            :disabled="updatingId === order.id"
            @click="confirmPayment(order)"
            class="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs font-semibold transition flex items-center gap-1 shadow-sm"
          >
            <span>💳</span> Pelunasan
          </button>

          <!-- Cancel Button -->
          <button
            v-if="order.status === 'received'"
            :disabled="updatingId === order.id"
            @click="cancelOrder(order)"
            class="text-xs text-gray-400 hover:text-red-600 transition"
          >
            Batalkan
          </button>
        </div>
      </div>
    </div>

    <!-- New Order Modal -->
    <ServiceOrderModal
      :show="showNewOrderModal"
      :submitting="submitting"
      @close="showNewOrderModal = false"
      @submit="handleOrderSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { serviceApi, type ServiceOrder, type CreateServiceOrderPayload } from '../api'
import ServiceOrderModal from './ServiceOrderModal.vue'

const orders = ref<ServiceOrder[]>([])
const loading = ref(false)
const submitting = ref(false)
const updatingId = ref<string | null>(null)
const selectedStatus = ref('all')
const searchQuery = ref('')
const showNewOrderModal = ref(false)

const statusOptions = [
  { key: 'all', label: 'Semua', icon: '📋' },
  { key: 'received', label: 'Diterima', icon: '📥' },
  { key: 'diagnosing', label: 'Diagnosa', icon: '🔍' },
  { key: 'working', label: 'Servis', icon: '🔧' },
  { key: 'testing', label: 'Uji Coba', icon: '⚡' },
  { key: 'ready', label: 'Siap Ambil', icon: '✅' },
  { key: 'completed', label: 'Selesai', icon: '🎉' },
]

function countByStatus(status: string): number {
  if (status === 'all') return orders.value.length
  return orders.value.filter(o => o.status === status).length
}

const filteredOrders = computed(() => {
  return orders.value.filter(o => {
    const matchesStatus = selectedStatus.value === 'all' || o.status === selectedStatus.value
    const q = searchQuery.value.toLowerCase().trim()
    const matchesQuery = !q ||
      o.order_no.toLowerCase().includes(q) ||
      o.unit_name.toLowerCase().includes(q) ||
      o.unit_identifier.toLowerCase().includes(q) ||
      o.customer_name.toLowerCase().includes(q) ||
      o.customer_phone.toLowerCase().includes(q)
    return matchesStatus && matchesQuery
  })
})

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    received: 'Unit Diterima',
    diagnosing: 'Proses Diagnosa',
    working: 'Sedang Dikerjakan',
    testing: 'Uji Coba / QC',
    ready: 'Siap Diambil',
    completed: 'Diserahkan ke Pemilik',
    cancelled: 'Dibatalkan',
  }
  return map[status] || status
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'received': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300'
    case 'diagnosing': return 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300'
    case 'working': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
    case 'testing': return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300'
    case 'ready': return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300 ring-1 ring-emerald-500'
    case 'completed': return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
    case 'cancelled': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
    default: return 'bg-gray-100 text-gray-800'
  }
}

function getNextStatus(curr: string): { key: string; label: string; icon: string } | null {
  switch (curr) {
    case 'received': return { key: 'diagnosing', label: 'Diagnosa', icon: '🔍' }
    case 'diagnosing': return { key: 'working', label: 'Mulai Kerja', icon: '🔧' }
    case 'working': return { key: 'testing', label: 'Uji Kelayakan', icon: '⚡' }
    case 'testing': return { key: 'ready', label: 'Unit Selesai (Siap Ambil)', icon: '✅' }
    case 'ready': return { key: 'completed', label: 'Serahkan Unit', icon: '🎉' }
    default: return null
  }
}

function formatRupiah(amountSen: number): string {
  const rupiah = Math.round(amountSen / 100)
  return 'Rp ' + rupiah.toLocaleString('id-ID')
}

function getWhatsAppLink(order: ServiceOrder): string {
  let phone = order.customer_phone.replace(/\D/g, '')
  if (phone.startsWith('0')) phone = '62' + phone.slice(1)
  const cost = order.final_cost > 0 ? order.final_cost : order.estimated_cost
  const text = encodeURIComponent(
    `Halo kak ${order.customer_name}, kami dari bengkel/servis menginformasikan perbaikan unit ${order.unit_name} (${order.order_no}) saat ini berstatus: ${statusLabel(order.status)}. Total biaya: ${formatRupiah(cost)}.`
  )
  return `https://wa.me/${phone}?text=${text}`
}

let debounceTimer: ReturnType<typeof setTimeout>
function debounceSearch() {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    fetchOrders()
  }, 400)
}

async function fetchOrders() {
  loading.value = true
  try {
    const params: { status?: string; search?: string } = {}
    if (selectedStatus.value !== 'all') {
      params.status = selectedStatus.value
    }
    if (searchQuery.value.trim()) {
      params.search = searchQuery.value.trim()
    }
    const res = await serviceApi.getOrders(params)
    if (res.success && Array.isArray(res.data)) {
      orders.value = res.data
    } else {
      orders.value = []
    }
  } catch (e) {
    console.error('Failed to fetch service orders', e)
  } finally {
    loading.value = false
  }
}

async function handleAdvanceStatus(order: ServiceOrder) {
  const next = getNextStatus(order.status)
  if (!next) return

  let finalCostSen: number | undefined = undefined

  if (next.key === 'ready') {
    const currCostRupiah = (order.final_cost > 0 ? order.final_cost : order.estimated_cost) / 100
    const promptVal = prompt(
      `Unit ${order.unit_name} siap diambil.\nMasukkan total biaya perbaikan akhir (Rp) [Notifikasi WhatsApp otomatis akan dikirim ke pelanggan]:`,
      String(currCostRupiah)
    )
    if (promptVal === null) return // cancelled
    const parsed = parseFloat(promptVal)
    if (!isNaN(parsed) && parsed >= 0) {
      finalCostSen = Math.round(parsed * 100)
    }
  }

  updatingId.value = order.id
  try {
    const res = await serviceApi.updateStatus(order.id, next.key, finalCostSen)
    if (res.success && res.data) {
      const idx = orders.value.findIndex(o => o.id === order.id)
      if (idx !== -1) {
        orders.value[idx] = res.data
      }
    } else {
      alert(res.message || 'Gagal mengubah status pekerjaan')
    }
  } catch (e) {
    console.error('Failed to update service order status', e)
  } finally {
    updatingId.value = null
  }
}

async function cancelOrder(order: ServiceOrder) {
  if (!confirm(`Batalkan SPK ${order.order_no} (${order.unit_name})?`)) return
  updatingId.value = order.id
  try {
    const res = await serviceApi.updateStatus(order.id, 'cancelled')
    if (res.success && res.data) {
      const idx = orders.value.findIndex(o => o.id === order.id)
      if (idx !== -1) {
        orders.value[idx] = res.data
      }
    }
  } catch (e) {
    console.error('Failed to cancel SPK', e)
  } finally {
    updatingId.value = null
  }
}

async function confirmPayment(order: ServiceOrder) {
  const cost = order.final_cost > 0 ? order.final_cost : order.estimated_cost
  if (!confirm(`Konfirmasi pelunasan biaya servis ${order.order_no} sebesar ${formatRupiah(cost)}?`)) return
  updatingId.value = order.id
  try {
    const res = await serviceApi.payOrder(order.id, 'cash')
    if (res.success && res.data) {
      const idx = orders.value.findIndex(o => o.id === order.id)
      if (idx !== -1) {
        orders.value[idx] = res.data
      }
    }
  } catch (e) {
    console.error('Failed to confirm service payment', e)
  } finally {
    updatingId.value = null
  }
}

async function handleOrderSubmit(payload: CreateServiceOrderPayload) {
  submitting.value = true
  try {
    const res = await serviceApi.createOrder(payload)
    if (res.success && res.data) {
      showNewOrderModal.value = false
      await fetchOrders()
    } else {
      alert(res.message || 'Gagal menerbitkan SPK')
    }
  } catch (e) {
    console.error('Failed to submit SPK', e)
    alert('Terjadi kesalahan koneksi saat menerbitkan SPK')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchOrders()
})
</script>
