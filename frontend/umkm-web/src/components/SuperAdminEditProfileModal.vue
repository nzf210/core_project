<template>
  <div class="modal-overlay" @click.self="$emit('update:editTarget', null)">
    <div class="modal-card" style="max-width: 540px; max-height: 85vh; overflow-y: auto;">
      <h3 style="margin: 0 0 0.25rem 0;">Edit Profil Tenant</h3>
      <p style="color: var(--text-secondary); font-size: 0.85rem; margin-bottom: 1.25rem;">
        {{ editTarget.name }} ({{ editTarget.owner_username || 'no owner' }})
      </p>

      <div class="logo-upload-section">
        <div class="logo-preview">
          <img v-if="editLogoPreview" :src="editLogoPreview" alt="Logo preview" />
          <img v-else-if="editForm.logo_url" :src="editForm.logo_url.startsWith('http') ? editForm.logo_url : apiBase + editForm.logo_url + '?t=' + Date.now()"
            alt="Current logo" />
          <div v-else class="logo-placeholder">No Logo</div>
        </div>
        <label class="file-input-label">
          <input type="file" accept="image/png,image/jpeg,image/webp" @change="$emit('logo-change', $event)"
            style="display:none" />
          <span class="btn btn-secondary" style="padding: 0.35rem 0.75rem; font-size: 0.8rem; cursor: pointer;">Pilih
            Logo</span>
        </label>
        <button v-if="editLogoFile" class="btn"
          style="background: transparent; color: var(--accent-primary); border: 1px solid var(--border-color); padding: 0.35rem 0.75rem; font-size: 0.8rem;"
          @click="$emit('upload-logo')" :disabled="uploadingLogo">
          {{ uploadingLogo ? 'Uploading...' : 'Upload Logo' }}
        </button>
      </div>

      <div class="form-group"><label>Nama Toko
        <input v-model="editForm.business_name" class="form-control" placeholder="Nama usaha/toko" /></label></div>
      <div class="form-group"><label>Nama Usaha (Tenant)
        <input v-model="editForm.name" class="form-control" /></label></div>
      <div class="form-group"><label>Subdomain
        <input v-model="editForm.subdomain" class="form-control" placeholder="opsional" /></label></div>
      <div class="form-group"><label>Custom Domain
        <input v-model="editForm.custom_domain" class="form-control" placeholder="opsional" /></label></div>
      <div class="form-group"><label>Xendit Merchant ID <span style="color: var(--text-secondary); font-weight: 400; font-size: 0.75rem;">(kosongkan jika pakai SaaS pool)</span>
        <input v-model="editForm.xendit_merchant_id" class="form-control" placeholder="opsional — untuk tenant B2B dengan akun Xendit sendiri" /></label></div>
      <div class="form-group"><label>Nomor WA Toko (CS)
        <input v-model="editForm.wa_number" class="form-control" placeholder="0812..." /></label></div>
      <div class="form-group"><label>Nomor WA Owner (Login)
        <input v-model="editForm.owner_phone" class="form-control" placeholder="0812..." /></label></div>
      <div class="form-group"><label>Alamat Usaha
        <textarea v-model="editForm.business_address" class="form-control" rows="2" placeholder="Alamat lengkap" /></label></div>
      <div class="form-group"><label>Jenis Usaha
        <select v-model="editForm.business_type" class="form-control">
          <option v-for="bt in businessTypes" :key="bt.id" :value="bt.id">{{ bt.name }}</option>
        </select></label></div>
      <div class="form-group"><label>Paket
        <select v-model="editForm.plan" class="form-control">
          <option v-for="plan in planOptions" :key="plan.id" :value="plan.id">{{ plan.name }}</option>
        </select></label></div>
      <div class="form-group"><label>Reset Password Owner <span
            style="color: var(--text-secondary); font-weight: 400; font-size: 0.8rem;">(kosongkan jika tidak ingin
            diubah)</span>
        <input v-model="editForm.new_password" type="password" class="form-control" placeholder="Password baru" /></label></div>

      <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
        <button class="btn btn-secondary" @click="$emit('update:editTarget', null)" :disabled="savingProfile">Batal</button>
        <button class="btn btn-primary" @click="$emit('save')" :disabled="savingProfile">
          {{ savingProfile ? 'Menyimpan...' : 'Simpan' }}
        </button>
      </div>
      <div v-if="profileError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">{{ profileError }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const apiBase = import.meta.env.VITE_API_BASE || ''

defineProps<{
  editTarget: any
  editForm: any
  editLogoFile: any
  editLogoPreview: string
  uploadingLogo: boolean
  savingProfile: boolean
  profileError: string
  businessTypes: any[]
  planOptions: any[]
}>()

defineEmits<{
  'update:editTarget': [value: any]
  'logo-change': [event: Event]
  'upload-logo': []
  'save': []
}>()
</script>

<style scoped>
.logo-upload-section {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
  flex-wrap: wrap;
}

.logo-preview {
  width: 80px;
  height: 80px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  flex-shrink: 0;
}

.logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.logo-placeholder {
  font-size: 0.7rem;
  color: var(--text-secondary);
  text-align: center;
}

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
