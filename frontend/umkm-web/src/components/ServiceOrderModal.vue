<template>
  <div v-if="show" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto p-6 space-y-4">
      <div class="flex justify-between items-center border-b border-gray-100 dark:border-gray-700 pb-3">
        <h3 class="text-lg font-bold text-gray-900 dark:text-white flex items-center gap-2">
          <span>🔧</span> Buat Surat Perintah Kerja (SPK) Servis Baru
        </h3>
        <button @click="$emit('close')" class="text-gray-400 hover:text-gray-600 text-xl font-bold">&times;</button>
      </div>

      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Customer Details -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="svcCustName" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Nama Pelanggan *</label>
            <input
              id="svcCustName"
              v-model="form.customer_name"
              type="text"
              required
              placeholder="Contoh: Budi Santoso"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
            />
          </div>
          <div>
            <label for="svcCustPhone" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">No. WhatsApp Pelanggan *</label>
            <input
              id="svcCustPhone"
              v-model="form.customer_phone"
              type="tel"
              required
              placeholder="Contoh: 081234567890"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
            />
          </div>
        </div>

        <!-- Unit Details -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="svcUnitName" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Nama Unit / Kendaraan / Alat *</label>
            <input
              id="svcUnitName"
              v-model="form.unit_name"
              type="text"
              required
              placeholder="Misal: Honda Vario 125 / Laptop Asus"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
            />
          </div>
          <div>
            <label for="svcUnitIdent" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">No. Polisi / Serial Number</label>
            <input
              id="svcUnitIdent"
              v-model="form.unit_identifier"
              type="text"
              placeholder="Misal: B 1234 XYZ / SN: 994821"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
            />
          </div>
        </div>

        <!-- Complaint -->
        <div>
          <label for="svcComplaint" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Keluhan / Gejala Kerusakan *</label>
          <textarea
            id="svcComplaint"
            v-model="form.complaint"
            rows="2"
            required
            placeholder="Deskripsikan masalah (misal: Rem depan bunyi berdecit keras, tarikan berat saat digas)"
            class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
          ></textarea>
        </div>

        <!-- Technician & Estimate -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label for="svcTechName" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Teknisi Penanggung Jawab</label>
            <input
              id="svcTechName"
              v-model="form.technician_name"
              type="text"
              placeholder="Contoh: Mas Joko"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
            />
          </div>
          <div>
            <label for="svcEstCost" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Estimasi Biaya (Rp)</label>
            <input
              id="svcEstCost"
              v-model.number="tempEstRupiah"
              type="number"
              min="0"
              step="1000"
              placeholder="Misal: 150000"
              class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
            />
          </div>
        </div>

        <div>
          <label for="svcNotes" class="block text-xs font-semibold text-gray-700 dark:text-gray-300 mb-1">Catatan Tambahan / Perlengkapan Bawaan</label>
          <textarea
            id="svcNotes"
            v-model="form.notes"
            rows="2"
            placeholder="Misal: Helm ditinggal di unit, charger dan tas laptop dibawa"
            class="w-full px-3 py-2 border rounded-lg border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
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
            class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold rounded-lg text-sm transition flex items-center gap-1 shadow"
          >
            <span v-if="submitting">Membuat SPK...</span>
            <span v-else>Terbitkan SPK</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CreateServiceOrderPayload } from '../api'

defineProps<{
  show: boolean
  submitting: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', payload: CreateServiceOrderPayload): void
}>()

const tempEstRupiah = ref<number | null>(null)

const form = ref({
  customer_name: '',
  customer_phone: '',
  unit_name: '',
  unit_identifier: '',
  complaint: '',
  technician_name: '',
  notes: '',
})

function handleSubmit() {
  emit('submit', {
    customer_name: form.value.customer_name.trim(),
    customer_phone: form.value.customer_phone.trim(),
    unit_name: form.value.unit_name.trim(),
    unit_identifier: form.value.unit_identifier.trim(),
    complaint: form.value.complaint.trim(),
    technician_name: form.value.technician_name.trim(),
    estimated_cost: tempEstRupiah.value ? Math.round(tempEstRupiah.value * 100) : 0,
    notes: form.value.notes.trim(),
  })
}
</script>
