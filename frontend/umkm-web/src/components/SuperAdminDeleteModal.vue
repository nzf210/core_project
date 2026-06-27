<template>
  <div class="modal-overlay" @click.self="$emit('update:deleteTarget', null)">
    <div class="modal-card">
      <h3 style="margin: 0 0 0.75rem 0;">Hapus Tenant?</h3>
      <p style="margin-bottom: 0.5rem;">Anda akan menghapus:</p>
      <p style="margin-bottom: 1.5rem; font-weight: 600;">{{ deleteTarget.name }} ({{ deleteTarget.owner_username ||
        'no owner' }})</p>
      <p style="margin-bottom: 1.5rem; color: #ef4444; font-size: 0.85rem;">
        ⚠️ Semua data: users, produk, jurnal, transaksi, dan data tenant akan dihapus permanen.
      </p>
      <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
        <button class="btn btn-secondary" @click="$emit('update:deleteTarget', null)"
          :disabled="deleting">Batal</button>
        <button class="btn" style="background: #ef4444; color: white; border: none;" @click="$emit('confirm')"
          :disabled="deleting">
          {{ deleting ? 'Menghapus...' : 'Hapus Permanen' }}
        </button>
      </div>
      <div v-if="deleteError" style="margin-top: 1rem; color: #ef4444; font-size: 0.85rem;">{{ deleteError }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  deleteTarget: any
  deleting: boolean
  deleteError: string
}>()

defineEmits<{
  'update:deleteTarget': [value: any]
  'confirm': []
}>()
</script>
