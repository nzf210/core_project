<template>
  <div class="p-6 max-w-7xl mx-auto space-y-6">
    <!-- Header & Quick Stats -->
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <span>🍽️</span> Kitchen Display & Pesanan Meja (KOT)
        </h1>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          Kelola alur tiket pesanan dapur, antrean masak koki, dan penyajian meja secara real-time
        </p>
      </div>
      <div class="flex items-center gap-3">
        <button
          @click="showNewOrderModal = true"
          class="px-4 py-2.5 bg-amber-600 hover:bg-amber-700 text-white font-medium rounded-lg shadow flex items-center gap-2 transition"
        >
          <span>➕</span> Pesanan Meja Baru
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
            ? 'border-amber-500 bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 ring-2 ring-amber-500'
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
        <label for="kotSearchInput" class="sr-only">Cari Pesanan Resto</label>
        <span class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-gray-400">🔍</span>
        <input
          id="kotSearchInput"
          v-model="searchQuery"
          @input="debounceSearch"
          type="text"
          placeholder="Cari nomor meja, nota (KOT-...), atau nama tamu..."
          class="w-full pl-9 pr-4 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-700/50 text-gray-900 dark:text-white text-sm focus:outline-none focus:ring-2 focus:ring-amber-500"
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
      <p class="text-sm">Memuat tiket pesanan dapur...</p>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredOrders.length === 0" class="text-center py-12 bg-white dark:bg-gray-800 rounded-xl border border-dashed border-gray-300 dark:border-gray-700 p-8">
      <span class="text-4xl block mb-2">🍽️</span>
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">Tidak ada pesanan aktif</h3>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-sm mx-auto">
        Belum ada pesanan dengan status ini. Klik tombol "Pesanan Meja Baru" untuk membuat tiket dapur.
      </p>
    </div>

    <!-- Orders Cards Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div
        v-for="order in filteredOrders"
        :key="order.id"
        class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm overflow-hidden flex flex-col justify-between"
      >
        <div class="p-4 space-y-3">
          <!-- Card Header: Table & Status -->
          <div class="flex justify-between items-start">
            <div>
              <div class="flex items-center gap-2">
                <span class="px-2.5 py-1 bg-amber-100 dark:bg-amber-900/40 text-amber-800 dark:text-amber-200 font-bold text-sm rounded-lg">
                  📍 {{ order.table_number }}
                </span>
                <span class="text-xs font-mono text-gray-500 dark:text-gray-400">
                  {{ order.order_no }}
                </span>
              </div>
              <p v-if="order.customer_name" class="text-xs text-gray-600 dark:text-gray-300 mt-1 font-medium">
                Tamu: {{ order.customer_name }}
              </p>
            </div>
            <span :class="['px-2.5 py-1 rounded-full text-xs font-semibold', statusBadgeClass(order.status)]">
              {{ statusLabel(order.status) }}
            </span>
          </div>

          <!-- Items Ordered -->
          <div class="bg-gray-50 dark:bg-gray-700/30 rounded-lg p-2.5 space-y-1.5 text-xs">
            <div
              v-for="(it, i) in order.items"
              :key="i"
              class="flex justify-between items-start text-gray-800 dark:text-gray-200"
            >
              <div>
                <span class="font-bold text-amber-600 dark:text-amber-400 mr-1">{{ it.qty }}x</span>
                <span class="font-medium">{{ it.name }}</span>
                <p v-if="it.notes" class="text-gray-500 dark:text-gray-400 italic text-[11px] pl-4">
                  Note: {{ it.notes }}
                </p>
              </div>
              <span class="text-gray-500 dark:text-gray-400 font-mono">{{ formatRupiah(it.price * it.qty) }}</span>
            </div>
          </div>

          <!-- Notes -->
          <p v-if="order.notes" class="text-xs text-amber-700 dark:text-amber-300 bg-amber-50 dark:bg-amber-900/20 px-2 py-1 rounded">
            📌 {{ order.notes }}
          </p>

          <!-- Total & Payment Status -->
          <div class="flex justify-between items-center pt-2 border-t border-gray-100 dark:border-gray-700 text-xs">
            <div>
              <span class="text-gray-500 dark:text-gray-400">Total Tagihan:</span>
              <p class="text-sm font-bold text-gray-900 dark:text-white">{{ formatRupiah(order.total_amount) }}</p>
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
            @click="updateStatus(order, getNextStatus(order.status)!.key)"
            class="px-3 py-1.5 bg-amber-600 hover:bg-amber-700 text-white rounded-lg text-xs font-semibold transition flex items-center gap-1 shadow-sm"
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
            <span>💳</span> Bayar Kasir
          </button>

          <!-- Cancel Button -->
          <button
            v-if="order.status === 'pending'"
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
    <RestaurantOrderModal
      :show="showNewOrderModal"
      :submitting="submitting"
      @close="showNewOrderModal = false"
      @submit="handleOrderSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { restaurantApi, type RestaurantOrder, type CreateRestaurantOrderPayload } from '../api'
import RestaurantOrderModal from './RestaurantOrderModal.vue'

const orders = ref<RestaurantOrder[]>([])
const loading = ref(false)
const submitting = ref(false)
const updatingId = ref<string | null>(null)
const selectedStatus = ref('all')
const searchQuery = ref('')
const showNewOrderModal = ref(false)

const statusOptions = [
  { key: 'all', label: 'Semua', icon: '📋' },
  { key: 'pending', label: 'Masuk', icon: '📥' },
  { key: 'cooking', label: 'Dimasak', icon: '🍳' },
  { key: 'ready_to_serve', label: 'Siap Saji', icon: '🛎️' },
  { key: 'served', label: 'Tersaji', icon: '🍽️' },
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
      o.table_number.toLowerCase().includes(q) ||
      o.customer_name.toLowerCase().includes(q)
    return matchesStatus && matchesQuery
  })
})

function statusLabel(status: string): string {
  const map: Record<string, string> = {
    pending: 'Pesanan Masuk',
    cooking: 'Sedang Dimasak',
    ready_to_serve: 'Siap Disajikan',
    served: 'Telah Disajikan',
    completed: 'Selesai / Lunas',
    cancelled: 'Dibatalkan',
  }
  return map[status] || status
}

function statusBadgeClass(status: string): string {
  switch (status) {
    case 'pending': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300'
    case 'cooking': return 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300 animate-pulse'
    case 'ready_to_serve': return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-300 ring-1 ring-emerald-500'
    case 'served': return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
    case 'completed': return 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
    case 'cancelled': return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
    default: return 'bg-gray-100 text-gray-800'
  }
}

function getNextStatus(curr: string): { key: string; label: string; icon: string } | null {
  switch (curr) {
    case 'pending': return { key: 'cooking', label: 'Mulai Masak', icon: '🍳' }
    case 'cooking': return { key: 'ready_to_serve', label: 'Makanan Siap', icon: '🛎️' }
    case 'ready_to_serve': return { key: 'served', label: 'Antar ke Meja', icon: '🍽️' }
    case 'served': return { key: 'completed', label: 'Tutup Meja', icon: '🎉' }
    default: return null
  }
}

function formatRupiah(amountSen: number): string {
  const rupiah = Math.round(amountSen / 100)
  return 'Rp ' + rupiah.toLocaleString('id-ID')
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
    const res = await restaurantApi.getOrders(params)
    if (res.success && Array.isArray(res.data)) {
      orders.value = res.data
    } else {
      orders.value = []
    }
  } catch (e) {
    console.error('Failed to fetch restaurant orders', e)
  } finally {
    loading.value = false
  }
}

async function updateStatus(order: RestaurantOrder, newStatus: string) {
  updatingId.value = order.id
  try {
    const res = await restaurantApi.updateStatus(order.id, newStatus)
    if (res.success && res.data) {
      const idx = orders.value.findIndex(o => o.id === order.id)
      if (idx !== -1) {
        orders.value[idx] = res.data
      }
    }
  } catch (e) {
    console.error('Failed to update restaurant order status', e)
  } finally {
    updatingId.value = null
  }
}

async function cancelOrder(order: RestaurantOrder) {
  if (!confirm(`Batalkan pesanan ${order.order_no} (${order.table_number})?`)) return
  await updateStatus(order, 'cancelled')
}

async function confirmPayment(order: RestaurantOrder) {
  if (!confirm(`Konfirmasi pembayaran kasir untuk meja ${order.table_number} (${order.order_no}) sebesar ${formatRupiah(order.total_amount)}?`)) return
  updatingId.value = order.id
  try {
    const res = await restaurantApi.payOrder(order.id, 'cash')
    if (res.success && res.data) {
      const idx = orders.value.findIndex(o => o.id === order.id)
      if (idx !== -1) {
        orders.value[idx] = res.data
      }
    }
  } catch (e) {
    console.error('Failed to confirm restaurant payment', e)
  } finally {
    updatingId.value = null
  }
}

async function handleOrderSubmit(payload: CreateRestaurantOrderPayload) {
  submitting.value = true
  try {
    const res = await restaurantApi.createOrder(payload)
    if (res.success && res.data) {
      showNewOrderModal.value = false
      await fetchOrders()
    } else {
      alert(res.message || 'Gagal membuat pesanan meja')
    }
  } catch (e) {
    console.error('Failed to submit restaurant order', e)
    alert('Terjadi kesalahan koneksi saat membuat pesanan meja')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchOrders()
})
</script>
