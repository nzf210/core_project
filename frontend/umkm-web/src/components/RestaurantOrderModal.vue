<template>
  <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto p-6 space-y-4">
      <div class="flex justify-between items-center border-b border-gray-100 dark:border-gray-700 pb-3">
        <h3 class="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <span>🍽️</span> Pesanan Baru / Tiket Dapur (KOT)
        </h3>
        <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600 text-xl font-bold">&times;</button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="restoTableNum" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Nomor / Nama Meja *</label>
            <input
              id="restoTableNum"
              v-model="form.table_number"
              type="text"
              required
              placeholder="Contoh: Meja 05 / VIP-1"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-amber-500 outline-none"
            />
          </div>
          <div>
            <label for="restoCustName" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Nama Tamu</label>
            <input
              id="restoCustName"
              v-model="form.customer_name"
              type="text"
              placeholder="Contoh: Pak Budi"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-amber-500 outline-none"
            />
          </div>
        </div>

        <div>
          <label for="restoCustPhone" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">No. WhatsApp Tamu (Opsional)</label>
          <input
            id="restoCustPhone"
            v-model="form.customer_phone"
            type="tel"
            placeholder="Contoh: 081234567890"
            class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-amber-500 outline-none"
          />
        </div>

        <!-- Order Items -->
        <div class="space-y-2 pt-2 border-t border-gray-100 dark:border-gray-700">
          <div class="flex justify-between items-center">
            <label class="block text-xs font-semibold text-gray-700 dark:text-gray-300">Rincian Menu Dapur *</label>
            <button
              type="button"
              @click="addItem"
              class="text-xs text-amber-600 dark:text-amber-400 font-semibold hover:underline flex items-center gap-1"
            >
              <span>➕</span> Tambah Menu
            </button>
          </div>

          <div
            v-for="(item, idx) in itemsList"
            :key="idx"
            class="p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-2 border border-gray-200 dark:border-gray-600"
          >
            <div class="flex gap-2">
              <input
                v-model="item.name"
                type="text"
                required
                placeholder="Nama menu (misal: Nasi Goreng)"
                class="flex-1 px-2.5 py-1.5 border rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-amber-500 outline-none"
              />
              <input
                v-model.number="item.qty"
                type="number"
                min="1"
                required
                placeholder="Porsi"
                class="w-16 px-2 py-1.5 border rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm text-center focus:ring-2 focus:ring-amber-500 outline-none"
              />
              <input
                v-model.number="item.priceRupiah"
                type="number"
                min="0"
                step="500"
                required
                placeholder="Rp Harga"
                class="w-28 px-2 py-1.5 border rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-amber-500 outline-none"
              />
              <button
                v-if="itemsList.length > 1"
                type="button"
                @click="removeItem(idx)"
                class="text-red-500 hover:text-red-700 px-1 text-sm font-bold"
                title="Hapus baris"
              >
                ✕
              </button>
            </div>
            <input
              v-model="item.notes"
              type="text"
              placeholder="Catatan pesanan dapur (misal: pedas sedang, tanpa daun bawang)"
              class="w-full px-2.5 py-1 border rounded border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-800 dark:text-gray-200 text-xs focus:ring-2 focus:ring-amber-500 outline-none"
            />
          </div>

          <div class="flex justify-between items-center text-sm font-bold text-gray-900 dark:text-white pt-1 px-1">
            <span>Estimasi Total:</span>
            <span class="text-amber-600 dark:text-amber-400">Rp {{ calculateTotal().toLocaleString('id-ID') }}</span>
          </div>
        </div>

        <div>
          <label for="restoNotes" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Catatan Tambahan untuk Kasir / Dapur</label>
          <textarea
            id="restoNotes"
            v-model="form.notes"
            rows="2"
            placeholder="Contoh: Tamu minta dihidangkan bertahap"
            class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-amber-500 outline-none"
          ></textarea>
        </div>

        <div class="flex justify-end gap-2 pt-3 border-t border-gray-100 dark:border-gray-700">
          <button
            type="button"
            @click="$emit('close')"
            class="px-4 py-2 border rounded-lg text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Batal
          </button>
          <button
            type="submit"
            :disabled="submitting"
            class="px-5 py-2 bg-amber-600 hover:bg-amber-700 text-white font-semibold rounded-lg text-sm transition flex items-center gap-1 shadow"
          >
            <span v-if="submitting">Mengirim ke Dapur...</span>
            <span v-else>Kirim ke Dapur (KOT)</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CreateRestaurantOrderPayload } from '../api'

defineProps<{
  show: boolean
  submitting: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', payload: CreateRestaurantOrderPayload): void
}>()

interface ItemRow {
  name: string
  qty: number
  priceRupiah: number | null
  notes?: string
}

const itemsList = ref<ItemRow[]>([
  { name: '', qty: 1, priceRupiah: null, notes: '' },
])

const form = ref<{
  table_number: string
  customer_name: string
  customer_phone: string
  notes: string
}>({
  table_number: '',
  customer_name: '',
  customer_phone: '',
  notes: '',
})

function addItem() {
  itemsList.value.push({ name: '', qty: 1, priceRupiah: null, notes: '' })
}

function removeItem(idx: number) {
  itemsList.value.splice(idx, 1)
}

function calculateTotal(): number {
  return itemsList.value.reduce((acc, it) => acc + (it.qty || 1) * (it.priceRupiah || 0), 0)
}

function handleSubmit() {
  const formattedItems = itemsList.value
    .filter(it => it.name.trim() !== '')
    .map(it => ({
      name: it.name.trim(),
      qty: it.qty || 1,
      price: Math.round((it.priceRupiah || 0) * 100), // convert to sen
      notes: it.notes?.trim() || '',
    }))

  if (formattedItems.length === 0) {
    alert('Mohon isi minimal 1 item menu pesanan')
    return
  }

  emit('submit', {
    table_number: form.value.table_number.trim(),
    customer_name: form.value.customer_name.trim(),
    customer_phone: form.value.customer_phone.trim(),
    items: formattedItems,
    notes: form.value.notes.trim(),
  })
}
</script>
