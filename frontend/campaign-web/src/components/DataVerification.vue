<template>
  <div class="verification-container flex items-center justify-center">
    <div class="glass-card verification-card">
      <div class="text-center mb-6">
        <h2 style="color: var(--accent-primary); margin-bottom: 0.5rem;">Verifikasi Data Kandidat</h2>
        <p class="text-muted">Untuk menggunakan fitur pemenangan, Anda harus memverifikasi identitas Anda terlebih dahulu.</p>
      </div>

      <form @submit.prevent="submitVerification" class="flex flex-col gap-4">
        <div>
          <label>Unggah Foto KTP</label>
          <input type="file" accept="image/*" class="input-field" required style="padding: 0.5rem;" />
        </div>
        
        <div>
          <label>Partai Pengusung / Independen</label>
          <input v-model="form.partai" type="text" class="input-field" placeholder="Misal: Partai ABC" required />
        </div>

        <div>
          <label>Daerah Pemilihan (Dapil)</label>
          <input v-model="form.dapil" type="text" class="input-field" placeholder="Misal: Jawa Barat I" required />
        </div>

        <button type="submit" class="btn-primary mt-4" :disabled="isLoading">
          {{ isLoading ? 'Memverifikasi...' : 'Kirim Verifikasi' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { authApi } from '../api'

const emit = defineEmits(['verified'])
const isLoading = ref(false)
const form = ref({ partai: '', dapil: '', ktp: 'dummy-file-data' })

const submitVerification = async () => {
  isLoading.value = true
  try {
    const res = await authApi.verifyData(
      localStorage.getItem('accessToken') || '',
      form.value
    )
    const data = await res.json()
    if (data.success) {
      localStorage.setItem('accessToken', data.data.accessToken)
      localStorage.setItem('refreshToken', data.data.refreshToken)
      localStorage.setItem('isDataVerified', 'true')
      alert("Verifikasi Berhasil! Anda sekarang dapat mengakses semua fitur pemenangan.")
      emit('verified')
    } else {
      alert("Verifikasi gagal: " + data.message)
    }
  } catch (err) {
    alert("Terjadi kesalahan sistem")
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.verification-container {
  min-height: 80vh;
  padding: 2rem;
}
.verification-card {
  width: 100%;
  max-width: 500px;
  padding: 3rem;
  animation: slideUp 0.5s ease-out;
}
.input-field {
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--surface-1);
  color: var(--text-primary);
  outline: none;
  width: 100%;
  margin-top: 0.25rem;
}
.input-field:focus {
  border-color: var(--accent-primary);
}
.btn-primary {
  background: var(--accent-gradient);
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-weight: 600;
  width: 100%;
}
.btn-primary:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}
.mb-6 { margin-bottom: 1.5rem; }
.mt-4 { margin-top: 1rem; }
.text-center { text-align: center; }
.flex { display: flex; }
.flex-col { flex-direction: column; }
.gap-4 { gap: 1rem; }
.items-center { align-items: center; }
.justify-center { justify-content: center; }

@keyframes slideUp {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
