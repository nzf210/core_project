<template>
  <div class="auth-container">
    <div class="auth-card glass-card">
      <h2 class="text-gradient" style="margin-bottom: 1.5rem; text-align: center;">
        {{ isLogin ? 'Login ke Akun Anda' : 'Buat Akun Baru' }}
      </h2>
      
      <form @submit.prevent="handleSubmit">
        <div class="form-group" v-if="!isLogin">
          <label>Nama Lengkap</label>
          <input v-model="form.name" type="text" class="form-control" required placeholder="John Doe" />
        </div>
        
        <div class="form-group">
          <label>Email</label>
          <input v-model="form.email" type="email" class="form-control" required placeholder="email@contoh.com" />
        </div>
        
        <div class="form-group">
          <label>Password</label>
          <input v-model="form.password" type="password" class="form-control" required placeholder="••••••••" />
        </div>

        <button type="submit" class="btn btn-primary w-100" style="margin-top: 1rem;" :disabled="isLoading">
          <span v-if="isLoading">Processing...</span>
          <span v-else>{{ isLogin ? 'Masuk' : 'Daftar' }}</span>
        </button>
      </form>

      <div class="auth-switch">
        <p>
          {{ isLogin ? 'Belum punya akun?' : 'Sudah punya akun?' }}
          <a href="#" @click.prevent="isLogin = !isLogin" class="text-accent">
            {{ isLogin ? 'Daftar sekarang' : 'Login di sini' }}
          </a>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { toast } from 'vue3-toastify'
import api from '../api'

const props = defineProps(['onAuthSuccess'])
const isLogin = ref(true)
const form = ref({ name: '', email: '', password: '' })
const isLoading = ref(false)

const handleSubmit = async () => {
  isLoading.value = true
  
  try {
    const endpoint = isLogin.value ? '/auth/login' : '/auth/register'
    const payload = isLogin.value 
      ? { username: form.value.email, password: form.value.password }
      : { username: form.value.name, email: form.value.email, password: form.value.password }
      
    const response = await api.post(endpoint, payload)
    
    if (response.data.success) {
      if (isLogin.value) {
        toast.success('Login berhasil!')
        props.onAuthSuccess(response.data.data.accessToken)
      } else {
        // Auto login after register
        const loginResponse = await api.post('/auth/login', {
          username: form.value.email,
          password: form.value.password
        })
        if (loginResponse.data.success) {
          toast.success('Pendaftaran dan login berhasil!')
          props.onAuthSuccess(loginResponse.data.data.accessToken)
        }
      }
    } else {
      toast.error(response.data.message || 'Authentication failed')
    }
  } catch (err: any) {
    toast.error(err.response?.data?.message || err.message || 'Network error')
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.auth-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 70vh;
}
.auth-card {
  width: 100%;
  max-width: 400px;
  padding: 2.5rem 2rem;
}
.form-group {
  margin-bottom: 1.25rem;
}
.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--text-secondary);
}
.form-control {
  width: 100%;
  padding: 0.75rem;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
}
.w-100 {
  width: 100%;
}
.auth-switch {
  margin-top: 1.5rem;
  text-align: center;
  font-size: 0.875rem;
}
.text-accent {
  color: var(--accent-primary);
  text-decoration: none;
  font-weight: 500;
}
.text-accent:hover {
  text-decoration: underline;
}
.text-accent:hover {
  text-decoration: underline;
}
</style>
