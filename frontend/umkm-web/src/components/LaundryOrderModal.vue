<template>
  <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto p-6 space-y-4">
      <div class="flex justify-between items-center border-b border-gray-100 dark:border-gray-700 pb-3">
        <h3 class="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <span>🧺</span> Terima Cucian Baru
        </h3>
        <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600 text-xl font-bold">&times;</button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div>
          <label for="lndCustomerName" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Nama Pelanggan *</label>
          <input
            id="lndCustomerName"
            v-model="form.customer_name"
            type="text"
            required
            placeholder="Contoh: Budi Santoso"
            class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none"
          />
        </div>

        <div>
          <label for="lndCustomerPhone" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">No. WhatsApp Pelanggan *</label>
          <input
            id="lndCustomerPhone"
            v-model="form.customer_phone"
            type="tel"
            required
            placeholder="Contoh: 081234567890"
            class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="lndServiceType" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Jenis Layanan</label>
            <select
              id="lndServiceType"
              v-model="form.service_type"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none"
            >
              <option value="kiloan">Cuci Kiloan</option>
              <option value="satuan">Cuci Satuan</option>
              <option value="dry_clean">Dry Clean</option>
            </select>
          </div>
          <div>
            <label for="lndServiceAmount" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">
              {{ form.service_type === 'kiloan' ? 'Berat (kg)' : 'Jumlah (pcs)' }}
            </label>
            <input
              v-if="form.service_type === 'kiloan'"
              id="lndServiceAmount"
              v-model.number="tempWeightKg"
              type="number"
              step="0.1"
              min="0"
              placeholder="Misal: 3.5"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none"
            />
            <input
              v-else
              id="lndServiceAmount"
              v-model.number="form.item_count"
              type="number"
              min="1"
              placeholder="Misal: 2"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none"
            />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="lndRackLocation" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Nomor Rak/Keranjang</label>
            <input
              id="lndRackLocation"
              v-model="form.rack_location"
              type="text"
              placeholder="Contoh: Rak A-02"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none"
            />
          </div>
          <div>
            <label for="lndTotalAmount" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Total Biaya (Rupiah) *</label>
            <input
              id="lndTotalAmount"
              v-model.number="tempRupiahAmount"
              type="number"
              required
              min="0"
              placeholder="Misal: 25000"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none font-bold"
            />
          </div>
        </div>

        <div class="p-3 bg-gray-50 dark:bg-gray-700/40 rounded-lg space-y-2">
          <div class="flex items-center gap-2">
            <input
              id="isPaidCheckboxModal"
              v-model="form.is_paid"
              type="checkbox"
              class="rounded text-blue-600 focus:ring-blue-500 h-4 w-4"
            />
            <label for="isPaidCheckboxModal" class="text-xs font-medium text-gray-800 dark:text-gray-200">
              Sudah dibayar lunas sekarang (Otomatis catat jurnal kas)
            </label>
          </div>
          <div v-if="form.is_paid" class="pt-2">
            <label for="lndPaymentMethod" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Metode Pembayaran</label>
            <select
              id="lndPaymentMethod"
              v-model="form.payment_method"
              class="w-full px-3 py-1.5 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-xs outline-none"
            >
              <option value="cash">Tunai (Cash)</option>
              <option value="qris">QRIS</option>
              <option value="transfer">Transfer Bank</option>
            </select>
          </div>
        </div>

        <div>
          <label for="lndNotes" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Catatan Tambahan (Opsional)</label>
          <textarea
            id="lndNotes"
            v-model="form.notes"
            rows="2"
            placeholder="Contoh: Pisahkan pakaian putih, parfum lavender"
            class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 outline-none"
          ></textarea>
        </div>

        <div class="flex items-center justify-end gap-3 pt-3 border-t border-gray-100 dark:border-gray-700">
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
            class="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white font-semibold rounded-lg text-sm transition flex items-center gap-1"
          >
            <span v-if="submitting">Menyimpan...</span>
            <span v-else>Simpan Order</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CreateLaundryOrderPayload } from '../api'

defineProps<{
  show: boolean
  submitting: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', payload: CreateLaundryOrderPayload): void
}>()

const tempWeightKg = ref<number | null>(null)
const tempRupiahAmount = ref<number | null>(null)

const form = ref<CreateLaundryOrderPayload>({
  customer_name: '',
  customer_phone: '',
  service_type: 'kiloan',
  weight_grams: 0,
  item_count: 0,
  rack_location: '',
  total_amount: 0,
  is_paid: false,
  payment_method: 'cash',
  notes: '',
})

function handleSubmit() {
  if (form.value.service_type === 'kiloan') {
    form.value.weight_grams = Math.round((tempWeightKg.value || 0) * 1000)
  }
  form.value.total_amount = Math.round((tempRupiahAmount.value || 0) * 100)
  emit('submit', { ...form.value })
}
</script>
