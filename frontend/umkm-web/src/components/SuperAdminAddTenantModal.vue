<template>
  <div class="modal-overlay" @click.self="$emit('update:showAddTenant', false)">
    <div class="modal-card" style="max-width: 450px;">
      <h3 style="margin: 0 0 1.25rem 0;">Tambah Tenant Baru</h3>

      <div class="form-group">
        <label>Nama Usaha (Tenant)
          <input v-model="formData.name" type="text" class="form-control" placeholder="cth: Toko Sembako Budi" /></label>
      </div>

      <div class="form-group">
        <label>Username Pemilik
          <input v-model="formData.username" @input="formData.username = formData.username.replace(/ /g, '_').toLowerCase()" type="text" class="form-control" placeholder="cth: budi_sembako" /></label>
      </div>

      <div class="form-group">
        <label>Email Pemilik
          <input v-model="formData.email" type="email" class="form-control" placeholder="cth: budi@contoh.com" /></label>
      </div>

      <div class="form-group">
        <label>Nomor WhatsApp (Aktif)
          <input v-model="formData.phone_number" type="text" class="form-control" placeholder="cth: 081234567890" /></label>
      </div>

      <div class="form-group">
        <label>Subdomain (Opsional)
          <input v-model="formData.subdomain" type="text" class="form-control" placeholder="cth: Sembako.saas.com" /></label>
      </div>

      <div class="form-group">
        <label>Custom Domain (Opsional)
          <input v-model="formData.custom_domain" type="text" class="form-control" placeholder="cth: www.tokosembako.com" /></label>
      </div>

      <div class="form-group">
        <label>Pilih Paket Langganan
          <select v-model="formData.plan" class="form-control">
            <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }} (Rp {{ (plan.price_monthly/100).toLocaleString('id-ID') }})</option>
          </select></label>
      </div>

      <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
        <button class="btn btn-secondary" @click="$emit('close')">Batal</button>
        <button class="btn btn-primary" @click="$emit('save')">Simpan</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  showAddTenant: boolean
  formData: any
  planOptions: any[]
}>()

defineEmits<{
  'update:showAddTenant': [value: boolean]
  'close': []
  'save': []
}>()
</script>

<style scoped>
.form-group {
  margin-bottom: 0.75rem;
}

.form-group label {
  display: block;
  font-size: 0.8rem;
  font-weight: 500;
  margin-bottom: 0.3rem;
  color: var(--text-primary);
}

.form-control {
  width: 100%;
  padding: 0.6rem 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-family: inherit;
  font-size: 0.875rem;
  box-sizing: border-box;
}

.form-control:focus {
  outline: none;
  border-color: var(--accent-primary);
}
</style>
