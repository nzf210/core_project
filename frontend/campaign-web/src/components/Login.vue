<template>
  <div class="login-container flex items-center justify-center">
    <div class="login-card glass-card">
      <div class="text-center mb-6">
        <h1 class="logo-title">Campaign<span class="text-gradient">Manager</span></h1>
        <p class="text-muted" v-if="initialRole">Mendaftar sebagai <strong>{{ initialRole.replace('_', ' ').toUpperCase() }}</strong></p>
        <p class="text-muted" v-else>Masuk ke sistem manajemen kampanye</p>
      </div>

      <form v-if="!showOTP" @submit.prevent="handleLogin" class="flex flex-col gap-4">
        <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
        
        <div>
          <label v-if="initialRole">Nomor WA / Username</label>
          <label v-else>Username / Nomor WA</label>
          <input v-model="form.username" type="text" class="input-field" placeholder="Masukkan Nomor WA (misal: 0812...)" required />
        </div>
        
        <div>
          <label>Password</label>
          <input v-model="form.password" type="password" class="input-field" placeholder="Masukkan password" required />
        </div>

        <div v-if="initialRole">
          <label>Kode Referral (opsional)</label>
          <input v-model="referralCode" type="text" class="input-field" placeholder="AGEN-XXXXXX" style="text-transform: uppercase;" />
        </div>

        <button type="submit" class="btn-primary mt-2" :disabled="isLoading">
          {{ isLoading ? 'Loading...' : (initialRole ? 'Kirim OTP Pendaftaran' : 'Login') }}
        </button>
        
        <button type="button" class="btn-secondary" @click="$emit('back')" style="margin-top: 0.5rem;">
          Kembali ke Beranda Publik
        </button>
      </form>

      <form v-else @submit.prevent="verifyOTP" class="flex flex-col gap-4">
        <div v-if="errorMsg" class="error-msg">{{ errorMsg }}</div>
        <p class="text-muted text-center" style="margin-bottom: 1rem;">
          Masukkan kode OTP yang dikirimkan ke WhatsApp Anda.
        </p>
        
        <div>
          <label>Kode OTP</label>
          <input v-model="otpCode" type="text" class="input-field text-center" placeholder="6 Digit OTP" required maxlength="6" style="font-size: 1.5rem; letter-spacing: 0.5rem;" />
        </div>

        <button type="submit" class="btn-primary mt-2" :disabled="isLoading">
          {{ isLoading ? 'Loading...' : 'Verifikasi OTP' }}
        </button>
        <button type="button" class="btn-secondary" @click="showOTP = false; errorMsg = ''" style="margin-top: 0.5rem;">
          Ganti Nomor
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { authApi } from '../api'

const props = defineProps<{ initialRole?: string }>()
const emit = defineEmits(['login-success', 'back'])

const form = ref({ username: '', password: '' })
const otpCode = ref('')
const showOTP = ref(false)
const errorMsg = ref('')
const isLoading = ref(false)
const referralCode = ref('')

const handleLogin = async () => {
  errorMsg.value = ''
  isLoading.value = true

  try {
    if (props.initialRole) {
      let res = await authApi.register({
        username: form.value.username, 
        password: form.value.password, 
        email: form.value.username + '@demo.local',
        phoneNumber: form.value.username,
        role: props.initialRole 
      })
      let data = await res.json()
      if (data.success) {
        showOTP.value = true
      } else {
        errorMsg.value = data.message || 'Gagal mengirim OTP.'
      }
    } else {
      let res = await authApi.login(form.value.username, form.value.password)
      let data = await res.json()
      if (data.success && data.data) {
        processLoginData(data.data)
      } else {
        errorMsg.value = data.message || 'Login gagal. Periksa kredensial Anda.'
      }
    }
  } catch (err) {
    errorMsg.value = 'Gagal terhubung ke server.'
  } finally {
    isLoading.value = false
  }
}

const verifyOTP = async () => {
  errorMsg.value = ''
  isLoading.value = true

  try {
    let res = await authApi.verifyOTP(form.value.username, otpCode.value)
    let data = await res.json()
    
    if (data.success) {
      let loginRes = await authApi.login(form.value.username, form.value.password)
      let loginData = await loginRes.json()
      if (loginData.success && loginData.data) {
        processLoginData(loginData.data)
      } else {
        errorMsg.value = "Akun dibuat, tapi gagal login otomatis. Silakan login manual."
        showOTP.value = false
      }
    } else {
      errorMsg.value = data.message || 'OTP salah.'
    }
  } catch (err) {
    errorMsg.value = 'Gagal terhubung ke server.'
  } finally {
    isLoading.value = false
  }
}

const processLoginData = (data: any) => {
  localStorage.setItem('accessToken', data.accessToken || data.data?.accessToken)
  localStorage.setItem('refreshToken', data.refreshToken || data.data?.refreshToken)
  localStorage.setItem('tenantId', data.tenantId || data.data?.tenantId)
  
  try {
    const token = data.accessToken || data.data?.accessToken
    const payload = JSON.parse(atob(token.split('.')[1]))
    localStorage.setItem('userRole', payload.role || props.initialRole || 'admin')
    localStorage.setItem('userName', payload.username || form.value.username)
    localStorage.setItem('isDataVerified', payload.isDataVerified ? 'true' : 'false')
  } catch (e) {
    console.error("JWT Decode error", e)
  }

  // F037: Redeem referral code after successful login (Campaign)
  const rfCode = referralCode.value.trim().toUpperCase()
  if (rfCode) {
    // Store pending code in localStorage, redeem after full auth setup
    localStorage.setItem('pending_referral_code', rfCode)
  }

  emit('login-success')
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  width: 100vw;
  background: var(--bg-primary);
}
.login-card {
  width: 100%;
  max-width: 400px;
  padding: 2.5rem;
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  border: 1px solid var(--border-color);
}
.logo-title {
  font-size: 2rem;
  font-weight: 800;
  margin: 0 0 0.5rem;
}
.text-center { text-align: center; }
.mb-6 { margin-bottom: 1.5rem; }
.mt-2 { margin-top: 0.5rem; }
.text-muted { color: var(--text-muted); font-size: 0.9rem; margin: 0; }
.flex { display: flex; }
.flex-col { flex-direction: column; }
.gap-4 { gap: 1rem; }
.items-center { align-items: center; }
.justify-center { justify-content: center; }
label {
  display: block;
  font-size: 0.85rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: var(--text-secondary);
}
.input-field {
  width: 100%;
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
}
.input-field:focus {
  border-color: var(--accent-primary);
}
.btn-primary {
  width: 100%;
  padding: 0.75rem;
  background: var(--accent-gradient);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  transition: opacity 0.2s;
}
.btn-primary:disabled { opacity: 0.7; cursor: not-allowed; }
.error-msg {
  padding: 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  text-align: center;
}
</style>
