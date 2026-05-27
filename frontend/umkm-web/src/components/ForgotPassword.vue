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
          <p class="text-muted">Masukkan email Anda untuk menerima token reset kata sandi.</p>
        </div>

        <form v-if="!success" @submit.prevent="handleForgot">
          <div class="form-group" style="margin-bottom: 2rem;">
            <label>Email Terdaftar</label>
            <input 
              v-model="email" 
              type="email" 
              class="form-control" 
              placeholder="Masukkan email Anda" 
              required 
            />
          </div>

          <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem;" :disabled="loading">
            {{ loading ? 'Memproses...' : 'Kirim Token Reset' }}
          </button>

          <div v-if="errorMsg" class="error-msg text-center" style="margin-top: 1rem; color: #ef4444; font-size: 0.875rem;">
            {{ errorMsg }}
          </div>
        </form>

        <div v-else class="text-center">
          <div style="color: #10b981; font-size: 3rem; margin-bottom: 1rem;">✓</div>
          <h3>Berhasil!</h3>
          <p class="text-muted" style="margin-bottom: 2rem;">Jika email tersebut terdaftar, kami telah membuat token reset (cek log backend).</p>
          <button class="btn btn-outline" style="width: 100%; padding: 0.75rem;" @click="router.push({ path: '/reset-password', query: { email } })">
            Lanjutkan ke Reset Sandi
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

const email = ref('')
const loading = ref(false)
const errorMsg = ref('')
const success = ref(false)

const handleForgot = async () => {
  loading.value = true
  errorMsg.value = ''

  try {
    const data = await api.forgotPassword(email.value)
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
  min-height: 80vh;
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
