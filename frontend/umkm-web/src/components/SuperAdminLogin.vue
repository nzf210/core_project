<template>
  <div class="auth-split-container">
    <div class="auth-image-side">
      <div class="glass-overlay">
        <h1 class="text-accent-primary">Super Admin</h1>
        <p>Kelola seluruh tenant dan WhatsApp Verifier</p>
      </div>
    </div>

    <div class="auth-form-side">
      <div class="auth-form-wrapper glass-card animate-fade-in">
        <div style="margin-bottom: 2rem;">
          <h2>Login Super Admin</h2>
          <p class="text-muted">Akses panel kontrol super administrator</p>
        </div>

        <form @submit.prevent="handleLogin">
          <div class="form-group">
            <label>Username</label>
            <input v-model="username" type="text" class="form-control" placeholder="Username superadmin" required />
          </div>

          <div class="form-group" style="margin-bottom: 2rem;">
            <label>Password</label>
            <div style="position: relative;">
              <input v-model="password" :type="showPassword ? 'text' : 'password'" class="form-control"
                placeholder="Masukkan password" required style="padding-right: 2.5rem;" />
              <button type="button" @click="showPassword = !showPassword"
                style="position: absolute; right: 0.75rem; top: 50%; transform: translateY(-50%); background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0; display: flex;">
                <svg v-if="!showPassword" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"></path>
                  <circle cx="12" cy="12" r="3"></circle>
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M9.88 9.88a3 3 0 1 0 4.24 4.24"></path>
                  <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68"></path>
                  <path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61"></path>
                  <line x1="2" y1="2" x2="22" y2="22"></line>
                </svg>
              </button>
            </div>
          </div>

          <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem;" :disabled="loading">
            {{ loading ? 'Memeriksa...' : 'Masuk' }}
          </button>
        </form>

        <div v-if="errorMsg" class="error-msg" style="margin-top: 1rem; color: #ef4444; font-size: 0.875rem; text-align: center;">
          {{ errorMsg }}
        </div>

        <p class="text-center" style="margin-top: 2rem; color: var(--text-secondary);">
          <router-link to="/login" style="color: var(--accent-primary);">← Kembali ke login biasa</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { superadminApi } from '../superadminApi'
import { sanitizeJWT, sanitizeUUID, sanitizeRole } from '../api'

const router = useRouter()

const username = ref('')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const errorMsg = ref('')

const handleLogin = async () => {
  loading.value = true
  errorMsg.value = ''

  try {
    const data = await superadminApi.login(username.value, password.value)

    if (data.success && data.data) {
      localStorage.setItem('access_token', sanitizeJWT(data.data.accessToken))
      localStorage.setItem('refresh_token', sanitizeJWT(data.data.refreshToken))
      localStorage.setItem('tenant_id', sanitizeUUID(data.data.tenantId))

      const role = sanitizeRole(data.data.role)
      if (!role) {
        errorMsg.value = 'Invalid role data'
        return
      }
      localStorage.setItem('role', role)

      router.push('/superadmin')
    } else {
      errorMsg.value = data.message || 'Login gagal. Periksa kredensial Anda.'
    }
  } catch (e) {
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
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.8), rgba(59, 130, 246, 0.9)), url('https://images.unsplash.com/photo-1558494949-ef010cbdcc31?auto=format&fit=crop&q=80');
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
  background: linear-gradient(to top, rgba(15, 23, 42, 0.9), transparent);
  color: white;
}

.glass-overlay h1 {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.auth-form-side {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: var(--bg-color);
}

.auth-form-wrapper {
  width: 100%;
  max-width: 450px;
  padding: 1.5rem;
}

@media (min-width: 480px) {
  .auth-form-side {
    padding: 2rem;
  }
  .auth-form-wrapper {
    padding: 2.5rem;
  }
}

@media (max-width: 480px) {
  .auth-split-container {
    border-radius: 0;
  }
}
</style>
