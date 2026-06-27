<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'

const router = useRouter()
const loading = ref(false)
const error = ref('')

const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

async function submit() {
  if (form.newPassword !== form.confirmPassword) {
    error.value = 'Password baru tidak cocok'
    return
  }
  if (form.newPassword.length < 8) {
    error.value = 'Password minimal 8 karakter'
    return
  }

  loading.value = true
  error.value = ''
  try {
    const res = await api.forceChangePassword(form.oldPassword, form.newPassword)
    if (res.success) {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('tenant_id')
      localStorage.removeItem('user_id')
      localStorage.removeItem('role')
      router.push('/login')
    } else {
      error.value = res.message || 'Gagal mengubah password'
    }
  } catch (e: any) {
    error.value = e?.message || 'Gagal terhubung ke server'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="force-password-change">
    <div class="form-container">
      <h2>Ubah Password Wajib</h2>
      <p class="subtitle">
        Password Anda sudah direset ke default.
        Silakan masukkan password baru untuk menggantinya.
      </p>

      <div v-if="error" class="error-msg">{{ error }}</div>

      <form @submit.prevent="submit">
        <div class="form-group">
          <label for="oldPassword">Password Lama (Default)</label>
          <input
            id="oldPassword"
            v-model="form.oldPassword"
            type="password"
            placeholder="Masukkan password lama"
            required
          />
        </div>

        <div class="form-group">
          <label for="newPassword">Password Baru</label>
          <input
            id="newPassword"
            v-model="form.newPassword"
            type="password"
            placeholder="Min. 8 karakter"
            required
            minlength="8"
          />
        </div>

        <div class="form-group">
          <label for="confirmPassword">Konfirmasi Password Baru</label>
          <input
            id="confirmPassword"
            v-model="form.confirmPassword"
            type="password"
            placeholder="Ulangi password baru"
            required
          />
        </div>

        <button type="submit" class="btn-submit" :disabled="loading">
          {{ loading ? 'Sedang menyimpan...' : 'Simpan Password Baru' }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.force-password-change {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 20px;
  background: #f5f5f5;
}
.form-container {
  background: #fff;
  padding: 32px;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.08);
  width: 100%;
  max-width: 420px;
}
h2 {
  margin: 0 0 8px;
  font-size: 22px;
  color: #333;
}
.subtitle {
  color: #666;
  font-size: 14px;
  margin-bottom: 24px;
  line-height: 1.5;
}
.form-group {
  margin-bottom: 16px;
}
.form-group label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: #444;
  margin-bottom: 6px;
}
.form-group input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 14px;
  box-sizing: border-box;
}
.form-group input:focus {
  outline: none;
  border-color: #4f46e5;
  box-shadow: 0 0 0 3px rgba(79,70,229,0.1);
}
.btn-submit {
  width: 100%;
  padding: 12px;
  background: #dc2626;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 8px;
}
.btn-submit:hover { background: #b91c1c; }
.btn-submit:disabled { opacity: 0.6; cursor: not-allowed; }
.error-msg {
  background: #fef2f2;
  color: #dc2626;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: 13px;
  margin-bottom: 16px;
  border: 1px solid #fecaca;
}
</style>