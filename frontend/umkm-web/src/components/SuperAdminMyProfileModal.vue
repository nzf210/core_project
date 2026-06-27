<template>
  <div class="modal-overlay" @click.self="$emit('update:showMyProfile', false)">
    <div class="modal-card" style="max-width: 440px;">
      <h3 style="margin: 0 0 1.25rem 0;">Edit Profil Saya</h3>

      <div class="form-group"><label>Username
        <input v-model="myProfile.username" class="form-control" /></label></div>
      <div class="form-group"><label>Nomor HP
        <input v-model="myProfile.phone_number" class="form-control" placeholder="0812..." /></label></div>
      <div class="form-group"><label>Password Lama
        <input v-model="myProfile.old_password" type="password" class="form-control" placeholder="(untuk ganti password)" /></label></div>
      <div class="form-group"><label>Password Baru
        <input v-model="myProfile.new_password" type="password" class="form-control" placeholder="(kosongkan jika tidak diganti)" /></label></div>

      <div style="display: flex; gap: 0.75rem; justify-content: flex-end; margin-top: 1.5rem;">
        <button class="btn btn-secondary" @click="$emit('update:showMyProfile', false)" :disabled="savingMyProfile">Batal</button>
        <button class="btn btn-primary" @click="$emit('save')" :disabled="savingMyProfile">
          {{ savingMyProfile ? 'Menyimpan...' : 'Simpan' }}
        </button>
      </div>
      <div v-if="myProfileError" style="margin-top: 0.75rem; color: #ef4444; font-size: 0.85rem;">
        {{ myProfileError }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  showMyProfile: boolean
  myProfile: any
  savingMyProfile: boolean
  myProfileError: string
}>()

defineEmits<{
  'update:showMyProfile': [value: boolean]
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
