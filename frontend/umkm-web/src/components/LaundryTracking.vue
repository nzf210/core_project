<template>
  <div class="p-6 max-w-7xl mx-auto space-y-6">
    <!-- Header & Quick Stats -->
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <span>🧺</span> Tracking Operasional Laundry
        </h1>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Kelola siklus cucian pelanggan, lokasi rak, status bayar, dan notifikasi WA otomatis
        </p>
      </div>
      <div class="flex items-center gap-3">
        <button
          @click="showNewOrderModal = true"
          class="px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg shadow flex items-center gap-2 transition"
        >
          <span>➕</span> Terima Cucian Baru
        </button>
      </div>
    </div>

    <!-- Status Tabs / Counters -->
    <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
      <button
        v-for="s in statusOptions"
        :key="s.key"
        @click="selectedStatus = s.key; fetchOrders()"
        :class="[
          'p-3 rounded-xl border text-left transition flex flex-col justify-between',
          selectedStatus === s.key
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 ring-2 ring-blue-500'
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
        <span class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-gray-400">🔍</span>
        <input
          v-model="searchQuery"
          @input="debounceSearch"
          type="text"
          placeholder="Cari nomor nota (LND-...), nama pelanggan, atau no WA..."
          class="w-full pl-9 pr-4 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-700/50 text-gray-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
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
      <p class="text-sm">Memuat data order laundry...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredOrders.length === 0" class="text-center py-12 bg-white dark:bg-gray-800 rounded-xl border border-dashed border-gray-300 dark:border-gray-700 p-8">
      <span class="text-4xl block mb-2">🧺</span>
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">Tidak ada order laundry</h3>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-sm mx-auto">
        Belum ada cucian dengan status ini. Klik tombol "Terima Cucian Baru" untuk mencatat order pertama.
      </p>
    </div>

    <!-- Order Grid Cards -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="order in filteredOrders"
        :key="order.id"
        class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-5 shadow-sm flex flex-col justify-between hover:shadow-md transition"
      >
        <div>
          <!-- Card Header -->
          <div class="flex items-start justify-between gap-2 border-b border-gray-100 dark:border-gray-700 pb-3">
            <div>
              <span class="text-xs font-mono font-bold text-blue-600 dark:text-blue-400">{{ order.order_no }}</span>
              <h3 class="text-base font-bold text-gray-900 dark:text-white mt-0.5">{{ order.customer_name }}</h3>
              <p class="text-xs text-gray-500 dark:text-gray-400 flex items-center gap-1 mt-0.5">
                <span>📱</span> {{ order.customer_phone }}
              </p>
            </div>
            <span :class="['px-2.5 py-1 rounded-full text-xs font-semibold capitalize', statusBadgeClass(order.status)]">
              {{ statusLabel(order.status) }}
            </span>
          </div>

          <!-- Card Details -->
          <div class="py-3 space-y-2 text-sm text-gray-600 dark:text-gray-300">
            <div class="flex justify-between items-center text-xs">
              <span class="text-gray-500 dark:text-gray-400">Layanan:</span>
              <span class="font-medium capitalize">{{ formatService(order) }}</span>
            </div>
            <div v-if="order.rack_location" class="flex justify-between items-center text-xs">
              <span class="text-gray-500 dark:text-gray-400">Lokasi Rak/Keranjang:</span>
              <span class="font-semibold text-gray-800 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">
                📦 {{ order.rack_location }}
              </span>
            </div>
            <div class="flex justify-between items-center text-xs">
              <span class="text-gray-500 dark:text-gray-400">Biaya:</span>
              <div class="text-right">
                <span class="font-bold text-gray-900 dark:text-white">{{ formatRupiah(order.total_amount) }}</span>
                <span
                  :class="[
                    'ml-1.5 px-1.5 py-0.5 rounded text-[10px] font-semibold',
                    order.is_paid ? 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300' : 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300'
                  ]"
                >
                  {{ order.is_paid ? 'LUNAS' : 'BELUM LUNAS' }}
                </span>
              </div>
            </div>
            <div v-if="order.notes" class="text-xs text-gray-500 dark:text-gray-400 italic bg-gray-50 dark:bg-gray-700/30 p-2 rounded">
              "{{ order.notes }}"
            </div>
          </div>
        </div>

        <!-- Card Actions -->
        <div class="pt-3 border-t border-gray-100 dark:border-gray-700 space-y-2">
          <!-- Next Status Trigger -->
          <div class="flex items-center gap-2">
            <button
              v-if="getNextStatus(order.status)"
              @click="updateStatus(order, getNextStatus(order.status)!.key)"
              :disabled="updatingId === order.id"
              class="flex-1 py-1.5 px-3 bg-blue-50 hover:bg-blue-100 dark:bg-blue-900/30 dark:hover:bg-blue-900/50 text-blue-700 dark:text-blue-300 rounded-lg text-xs font-semibold transition flex items-center justify-center gap-1"
            >
              <span>{{ getNextStatus(order.status)!.icon }}</span>
              <span>Lanjut: {{ getNextStatus(order.status)!.label }}</span>
            </button>
            <button
              v-if="!order.is_paid"
              @click="confirmPayment(order)"
              :disabled="updatingId === order.id"
              class="py-1.5 px-3 bg-green-600 hover:bg-green-700 text-white rounded-lg text-xs font-semibold transition"
            >
              💰 Pelunasan
            </button>
          </div>

          <!-- Secondary Actions -->
          <div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 pt-1">
            <a
              :href="getWhatsAppLink(order)"
              target="_blank"
              class="hover:text-green-600 dark:hover:text-green-400 flex items-center gap-1 transition"
            >
              <span>💬</span> Hubungi WA
            </a>
            <button
              v-if="order.status !== 'cancelled' && order.status !== 'completed'"
              @click="updateStatus(order, 'cancelled')"
              class="hover:text-red-500 transition text-[11px]"
            >
              Batalkan
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Form Component -->
    <LaundryOrderModal
      :show="showNewOrderModal"
      :submitting="submitting"
      @close="showNewOrderModal = false"
      @submit="handleOrderSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { laundryApi, type LaundryOrder, type CreateLaundryOrderPayload } from '../api'
import { formatRupiah } from '../composables/useCurrency'
import LaundryOrderModal from './LaundryOrderModal.vue'

const orders = ref<LaundryOrder[]>([])
const loading = ref(false)
const submitting = ref(false)
const updatingId = ref<string | null>(null)
const selectedStatus = ref<string>('all')
const searchQuery = ref('')
const showNewOrderModal = ref(false)

const statusOptions = [
  { key: 'all', label: 'Semua', icon: '📋' },
  { key: 'received', label: 'Antre', icon: '📥' },
  { key: 'washing', label: 'Cuci', icon: '🫧' },
  { key: 'drying', label: 'Kering', icon: '☀️' },
  { key: 'ironing', label: 'Setrika', icon: '👔' },
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
      o.customer_name.toLowerCase().includes(q) ||
      o.customer_phone.toLowerCase().includes(q)
    return matchesStatus && matchesQuery
  })
})

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    received: 'Diterima',
    washing: 'Dicuci',
    drying: 'Pengeringan',
    ironing: 'Setrika / Packing',
    ready: 'Siap Diambil',
    completed: 'Selesai Diambil',
    cancelled: 'Dibatalkan',
  }
  return map[status] || status
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'received': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300'
    case 'washing': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
    case 'drying': return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300'
    case 'ironing': return 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300'
    case 'ready': return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 ring-1 ring-green-500'
    case 'completed': return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
    case 'cancelled': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
    default: return 'bg-gray-100 text-gray-800'
  }
}

function formatService(o: LaundryOrder): string {
  if (o.service_type === 'kiloan') {
    const kg = (o.weight_grams / 1000).toFixed(1)
    return `Kiloan (${kg} kg)`
  }
  if (o.service_type === 'satuan') {
    return `Satuan (${o.item_count} pcs)`
  }
  return o.service_type
}

function getNextStatus(curr: string): { key: string; label: string; icon: string } | null {
  switch (curr) {
    case 'received': return { key: 'washing', label: 'Cuci', icon: '🫧' }
    case 'washing': return { key: 'drying', label: 'Keringkan', icon: '☀️' }
    case 'drying': return { key: 'ironing', label: 'Setrika', icon: '👔' }
    case 'ironing': return { key: 'ready', label: 'Siap Ambil', icon: '✅' }
    case 'ready': return { key: 'completed', label: 'Diserahkan', icon: '🎉' }
    default: return null
  }
}

function getWhatsAppLink(order: LaundryOrder): string {
  let phone = order.customer_phone.replace(/\D/g, '')
  if (phone.startsWith('0')) phone = '62' + phone.slice(1)
  const text = encodeURIComponent(`Halo Kak ${order.customer_name}, perihal cucian Anda no nota ${order.order_no} di Laundry kami...`)
  return `https://wa.me/${phone}?text=${text}`
}

let searchTimer: any = null
function debounceSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {}, 300)
}

async function fetchOrders() {
  loading.value = true
  try {
    const res = await laundryApi.getOrders({
      status: selectedStatus.value === 'all' ? undefined : selectedStatus.value,
      search: searchQuery.value || undefined,
    })
    if (res.success && res.data) {
      orders.value = res.data
    }
  } catch (e) {
    console.error('Failed to fetch laundry orders', e)
  } finally {
    loading.value = false
  }
}

async function updateStatus(order: LaundryOrder, nextStatus: string) {
  updatingId.value = order.id
  try {
    const res = await laundryApi.updateStatus(order.id, nextStatus)
    if (res.success && res.data) {
      const idx = orders.value.findIndex(o => o.id === order.id)
      if (idx !== -1) {
        orders.value[idx] = res.data
      }
    }
  } catch (e) {
    console.error('Failed to update status', e)
  } finally {
    updatingId.value = null
  }
}

async function confirmPayment(order: LaundryOrder) {
  if (!confirm(`Konfirmasi pelunasan untuk nota ${order.order_no} sebesar ${formatRupiah(order.total_amount)}?`)) return
  updatingId.value = order.id
  try {
    const res = await laundryApi.payOrder(order.id, 'cash')
    if (res.success && res.data) {
      const idx = orders.value.findIndex(o => o.id === order.id)
      if (idx !== -1) {
        orders.value[idx] = res.data
      }
    }
  } catch (e) {
    console.error('Failed to confirm payment', e)
  } finally {
    updatingId.value = null
  }
}

async function handleOrderSubmit(payload: CreateLaundryOrderPayload) {
  submitting.value = true
  try {
    const res = await laundryApi.createOrder(payload)
    if (res.success && res.data) {
      showNewOrderModal.value = false
      await fetchOrders()
    } else {
      alert(res.message || 'Gagal membuat order laundry')
    }
  } catch (e) {
    console.error('Failed to submit order', e)
    alert('Terjadi kesalahan koneksi saat membuat order laundry')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchOrders()
})
</script>
