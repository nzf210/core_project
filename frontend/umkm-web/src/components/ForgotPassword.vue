<template>
  <div class="auth-split-container">
    <div class="auth-image-side">
      <div class="glass-overlay">
        <h1 class="text-gradient">WCH UMKM</h1>
        <p>Dashboard terpadu dengan asisten AI akuntansi pertama di Indonesia.</p>
      </div>
    </div>
    
    <div class="auth-form-side">
      <div class="auth-form-wrapper glass-card animate-fade-in">
        <div style="margin-bottom: 2.5rem;">
          <h2>Lupa Kata Sandi</h2>
          <p class="text-muted">Masukkan username dan nomor HP terdaftar untuk mereset password.</p>
        </div>

        <form v-if="!success" @submit.prevent="handleForgot">
          <div class="form-group">
            <label>Username</label>
            <input
              v-model="username"
              type="text"
              class="form-control"
              placeholder="Masukkan username Anda"
              required
            />
          </div>

          <div class="form-group" style="margin-bottom: 2rem;">
            <label>Nomor HP Terdaftar</label>
            <input
              v-model="phoneNumber"
              type="tel"
              class="form-control"
              placeholder="Masukkan nomor HP terdaftar"
              required
            />
          </div>

          <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem;" :disabled="loading">
            {{ loading ? 'Memproses...' : 'Reset Password' }}
          </button>

          <div v-if="errorMsg" class="error-msg text-center" style="margin-top: 1rem; color: #ef4444; font-size: 0.875rem;">
            {{ errorMsg }}
          </div>
        </form>

        <div v-else class="text-center">
          <div style="color: #10b981; font-size: 3rem; margin-bottom: 1rem;">✓</div>
          <h3>Berhasil!</h3>
          <p class="text-muted" style="margin-bottom: 2rem;">Password telah direset ke default. Silakan login dan ubah password Anda.</p>
          <button class="btn btn-primary" style="width: 100%; padding: 0.75rem;" @click="router.push('/login')">
            Kembali ke Login
          </button>
        </div>

        <p class="text-center" style="margin-top: 2rem; color: var(--text-secondary);">
          Ingat kata sandi Anda? <router-link to="/login" style="color: var(--accent-primary);">Masuk di sini</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'

const router = useRouter()

const username = ref('')
const phoneNumber = ref('')
const loading = ref(false)
const errorMsg = ref('')
const success = ref(false)

const handleForgot = async () => {
  loading.value = true
  errorMsg.value = ''

  try {
    const data = await api.resetPasswordDefault(username.value, phoneNumber.value)
    if (data.success) {
      success.value = true
    } else {
      errorMsg.value = data.message || 'Gagal memproses permintaan.'
    }
  } catch (e) {
    console.error(e)
    errorMsg.value = 'Terjadi kesalahan jaringan.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-split-container {
  display: flex;
  min-height: calc(100vh - 4rem);
  min-height: calc(100dvh - 4rem);
  border-radius: var(--radius-xl);
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  background: var(--surface-0);
}

.auth-image-side {
  flex: 1;
  background: linear-gradient(135deg, rgba(59,130,246,0.8), rgba(30,58,138,0.9)), url('https://images.unsplash.com/photo-1554224155-6726b3ff858f?auto=format&fit=crop&q=80');
  background-size: cover;
  background-position: center;
  background-blend-mode: overlay;
  position: relative;
  display: none;
}

@media (min-width: 768px) {
  .auth-image-side {
    display: block;
  }
}

.glass-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 3rem;
  background: linear-gradient(to top, rgba(15,23,42,0.9), transparent);
  color: white;
}

.auth-form-side {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background: var(--bg-color);
}

.auth-form-wrapper {
  width: 100%;
  max-width: 450px;
  padding: 2.5rem;
}
</style>
