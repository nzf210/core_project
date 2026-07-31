<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = 'Username dan password wajib diisi'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await fetch('/api/superadmin/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    const data = await res.json()
    if (!res.ok) {
      error.value = data.message || 'Login gagal'
      return
    }
    const token = data.data?.accessToken || data.accessToken || data.data?.access_token || data.access_token
    if (!token) {
      error.value = 'Token tidak ditemukan dalam response'
      return
    }
    // Validate role
    try {
      const payload = JSON.parse(atob(token.split('.')[1]))
      if (payload.role !== 'superadmin') {
        error.value = 'Akses ditolak. Hanya superadmin yang dapat masuk.'
        return
      }
    } catch {
      error.value = 'Token tidak valid'
      return
    }
    localStorage.setItem('access_token', token)
    router.push('/')
  } catch (e: any) {
    error.value = e.message || 'Gagal terhubung ke server'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <div class="brand">
        <span class="brand-icon">⚡</span>
        <h1>WCH Superadmin</h1>
        <p class="brand-sub">Platform Control Center</p>
      </div>

      <form @submit.prevent="handleLogin" class="form">
        <div class="field">
          <label for="username">Username</label>
          <input
            id="username"
            v-model="username"
            type="text"
            placeholder="superadmin"
            autocomplete="username"
            :disabled="loading"
            required
          />
        </div>
        <div class="field">
          <label for="password">Password</label>
          <input
            id="password"
            v-model="password"
            type="password"
            placeholder="••••••••"
            autocomplete="current-password"
            :disabled="loading"
            required
          />
        </div>

        <div v-if="error" class="error-box">
          <span>⚠️</span> {{ error }}
        </div>

        <button type="submit" class="btn-login" :disabled="loading">
          <span v-if="loading" class="spinner">⏳</span>
          <span v-else>Masuk →</span>
        </button>
      </form>

      <p class="footer-note">Hanya untuk administrator WCH Platform</p>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  background-image: radial-gradient(ellipse at 50% 0%, rgba(59, 130, 246, 0.08) 0%, transparent 60%);
}

.login-card {
  width: 100%;
  max-width: 400px;
  background: var(--card);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 40px 36px;
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.4);
}

.brand {
  text-align: center;
  margin-bottom: 32px;
}

.brand-icon {
  font-size: 36px;
  display: block;
  margin-bottom: 10px;
}

.brand h1 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
  margin-bottom: 4px;
}

.brand-sub {
  font-size: 13px;
  color: var(--muted);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 13px;
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.field input {
  width: 100%;
  font-size: 15px;
  padding: 10px 14px;
  border-radius: 8px;
  transition: border-color 0.2s;
}

.error-box {
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: var(--danger);
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13px;
  display: flex;
  gap: 8px;
  align-items: center;
}

.btn-login {
  width: 100%;
  padding: 12px;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  transition: background 0.2s, transform 0.1s;
  margin-top: 4px;
}

.btn-login:hover:not(:disabled) {
  background: #2563eb;
  transform: translateY(-1px);
}

.btn-login:active:not(:disabled) {
  transform: translateY(0);
}

.footer-note {
  text-align: center;
  font-size: 12px;
  color: var(--muted);
  margin-top: 24px;
}
</style>
