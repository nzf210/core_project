<template>
  <div class="auth-split-container">
    <div class="auth-image-side">
      <div class="glass-overlay">
        <h1 class="text-gradient">{{ businessName }}</h1>
        <p>Dashboard terpadu dengan asisten AI akuntansi pertama di Indonesia.</p>
      </div>
    </div>

    <div class="auth-form-side">
      <div class="auth-form-wrapper glass-card animate-fade-in">
        <div
          style="margin-bottom: 2rem; display: flex; flex-direction: column; align-items: center; text-align: center;">
          <img v-if="logoUrl" :src="API_BASE + logoUrl" alt="Logo"
            style="max-height: 80px; margin-bottom: 1rem; border-radius: var(--radius-sm);" />
          <h2 style="margin: 0; margin-bottom: 0.5rem;">{{ businessName === 'WCH UMKM' ? 'Selamat Datang Kembali' :
            'Selamat Datang di ' + businessName }}</h2>
          <p class="text-muted">Masuk ke dashboard keuangan Anda</p>
        </div>

        <div class="login-tabs">
          <button :class="['tab-btn', { active: loginMode === 'wa' }]" @click="loginMode = 'wa'">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2">
              <path d="M3 21l1.65-3.8a9 9 0 1 1 3.4 2.9L3 21" />
              <path d="M9 10a.5.5 0 0 0 1 0V9a.5.5 0 0 0-1 0v1Zm0 0a5 5 0 0 0 5 5h1a.5.5 0 0 0 0-1h-1a.5.5 0 0 0 0 1" />
            </svg>
            WhatsApp
          </button>
          <button :class="['tab-btn', { active: loginMode === 'telegram' }]" @click="loginMode = 'telegram'">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2">
              <path d="M21.5 2.5 2.5 10.5 10 14l3-10 8.5-1.5Z" />
              <path d="M10 14v6.5l4-3" />
              <path d="m2.5 10.5 19-8" />
            </svg>
            Telegram
          </button>
          <button :class="['tab-btn', { active: loginMode === 'password' }]" @click="loginMode = 'password'">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
              <path d="M7 11V7a5 5 0 0 1 10 0v4" />
            </svg>
            Password
          </button>
        </div>

        <!-- WA Login Form -->
        <form v-if="loginMode === 'wa'"
          @submit.prevent="phoneStep === 'input' ? handlePhoneLogin() : handleVerifyPhoneLogin()">
          <div v-if="phoneStep === 'input'" class="form-group">
            <label>Nomor WhatsApp</label>
            <input v-model="phoneNumber" type="text" class="form-control" placeholder="Contoh: 081234567890" required />
          </div>

          <div v-if="phoneStep === 'verify'" class="form-group">
            <label>Kode OTP (dikirim ke {{ phoneNumber }})</label>
            <input v-model="phoneOTP" type="text" class="form-control" placeholder="Masukkan 6 digit OTP" maxlength="6"
              required />
            <p style="margin-top: 0.5rem; font-size: 0.8rem; color: var(--text-secondary);">
              <a href="#" @click.prevent="phoneStep = 'input'" style="color: var(--accent-primary);">Ganti nomor</a>
            </p>
          </div>

          <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem;" :disabled="loading">
            {{ loading ? 'Memproses...' : (phoneStep === 'input' ? 'Kirim OTP' : 'Verifikasi & Masuk') }}
          </button>
        </form>

        <!-- Telegram Login Form -->
        <form v-if="loginMode === 'telegram'"
          @submit.prevent="telegramStep === 'input' ? handleTelegramLogin() : handleTelegramVerifyLogin()">
          <div v-if="telegramStep === 'input'" class="form-group">
            <label>Chat ID Telegram</label>
            <input v-model="telegramChatId" type="text" class="form-control" placeholder="cth: 123456789" required />
            <p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">
              Chat bot kami di <a href="https://t.me/WCHBot" target="_blank"
                style="color: var(--accent-primary);">@core_tesbot</a> — kirim /start untuk dapat Chat ID
            </p>
          </div>

          <div v-if="telegramStep === 'input'" class="form-group">
            <label>Nomor WhatsApp (terdaftar)</label>
            <input v-model="telegramPhone" type="text" class="form-control" placeholder="cth: 081234567890" required />
          </div>

          <div v-if="telegramStep === 'verify'" class="form-group">
            <label>Kode OTP (dikirim ke Telegram Anda)</label>
            <input v-model="telegramOTP" type="text" class="form-control" placeholder="Masukkan 6 digit OTP"
              maxlength="6" required />
            <p style="margin-top: 0.5rem; font-size: 0.8rem; color: var(--text-secondary);">
              <a href="#" @click.prevent="telegramStep = 'input'" style="color: var(--accent-primary);">Ganti Chat ID /
                nomor</a>
            </p>
          </div>

          <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem;" :disabled="loading">
            {{ loading ? 'Memproses...' : (telegramStep === 'input' ? 'Kirim OTP via Telegram' : 'Verifikasi & Masuk')
            }}
          </button>
        </form>

        <!-- Password Login Form -->
        <form v-if="loginMode === 'password'" @submit.prevent="handleLogin">
          <div class="form-group">
            <label>Username / Email</label>
            <input v-model="username" type="text" class="form-control" placeholder="Masukkan username atau email"
              required />
          </div>

          <div class="form-group" style="margin-bottom: 2rem;">
            <div class="flex items-center justify-between"
              style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
              <label>Password</label>
              <router-link to="/forgot-password" style="font-size: 0.8rem; color: var(--accent-primary);">Lupa
                Password?</router-link>
            </div>
            <div style="position: relative;">
              <input v-model="password" :type="showPassword ? 'text' : 'password'" class="form-control"
                placeholder="Masukkan password" required style="padding-right: 2.5rem;" />
              <button type="button" @click="showPassword = !showPassword"
                style="position: absolute; right: 0.75rem; top: 50%; transform: translateY(-50%); background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0; display: flex;">
                <svg v-if="!showPassword" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24"
                  fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"></path>
                  <circle cx="12" cy="12" r="3"></circle>
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
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

        <div v-if="errorMsg" class="error-msg text-center"
          style="margin-top: 1rem; color: #ef4444; font-size: 0.875rem;">
          {{ errorMsg }}
        </div>
        <div v-if="successMsg" class="success-msg text-center"
          style="margin-top: 1rem; color: #10b981; font-size: 0.875rem;">
          {{ successMsg }}
        </div>

        <p class="text-center" style="margin-top: 2rem; color: var(--text-secondary);">
          Belum punya akun? <router-link to="/register" style="color: var(--accent-primary);">Daftar di
            sini</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, API_BASE } from '../api'

const router = useRouter()

const loginMode = ref<'wa' | 'telegram' | 'password'>('wa')
const phoneStep = ref<'input' | 'verify'>('input')
const telegramStep = ref<'input' | 'verify'>('input')

const businessName = ref(localStorage.getItem('active_domain_business_name') || 'WCH UMKM')
const logoUrl = ref(localStorage.getItem('active_domain_logo_url') || '')

// WA login
const phoneNumber = ref('')
const phoneOTP = ref('')

// Telegram login
const telegramChatId = ref('')
const telegramPhone = ref('')
const telegramOTP = ref('')

// Password login
const username = ref('')
const password = ref('')
const showPassword = ref(false)

const loading = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const handlePhoneLogin = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const data = await api.phoneLogin(phoneNumber.value)
    if (data.success) {
      phoneStep.value = 'verify'
      successMsg.value = data.message || 'OTP telah dikirim ke WhatsApp Anda.'
    } else {
      errorMsg.value = data.message || 'Nomor tidak terdaftar.'
    }
  } catch (e) {
    errorMsg.value = 'Terjadi kesalahan jaringan.'
  } finally {
    loading.value = false
  }
}

const handleVerifyPhoneLogin = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const data = await api.verifyPhoneLogin(phoneNumber.value, phoneOTP.value)
    if (data.success && data.data) {
      localStorage.setItem('access_token', data.data.accessToken)
      localStorage.setItem('refresh_token', data.data.refreshToken)
      localStorage.setItem('tenant_id', data.data.tenantId)
      localStorage.setItem('role', data.data.role)

      try {
        const profileRes = await api.get('/api/profile')
        if (profileRes.success && profileRes.data) {
          localStorage.setItem('plan', profileRes.data.plan || 'lite')
          if (profileRes.data.business_name || data.data.role !== 'owner') {
            localStorage.setItem('onboarding_completed', 'true')
          }
          if (typeof profileRes.data.is_frozen === 'boolean') {
            sessionStorage.setItem('subscription_status', profileRes.data.is_frozen ? 'frozen' : 'active')
          } else {
            sessionStorage.setItem('subscription_status', 'active')
          }
        }
      } catch (err) {
        console.error('Failed to check profile for onboarding status')
      }

      router.push('/')
    } else {
      errorMsg.value = data.message || 'OTP tidak valid.'
    }
  } catch (e) {
    errorMsg.value = 'Terjadi kesalahan jaringan.'
  } finally {
    loading.value = false
  }
}

const handleTelegramLogin = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const data = await api.telegramLogin(telegramChatId.value, telegramPhone.value)
    if (data.success) {
      telegramStep.value = 'verify'
      successMsg.value = data.message || 'OTP telah dikirim ke Telegram Anda.'
    } else {
      errorMsg.value = data.message || 'Nomor tidak terdaftar atau Chat ID tidak valid.'
    }
  } catch (e) {
    errorMsg.value = 'Terjadi kesalahan jaringan.'
  } finally {
    loading.value = false
  }
}

const handleTelegramVerifyLogin = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const data = await api.verifyPhoneLogin(telegramPhone.value, telegramOTP.value)
    if (data.success && data.data) {
      localStorage.setItem('access_token', data.data.accessToken)
      localStorage.setItem('refresh_token', data.data.refreshToken)
      localStorage.setItem('tenant_id', data.data.tenantId)
      localStorage.setItem('role', data.data.role)

      try {
        const profileRes = await api.get('/api/profile')
        if (profileRes.success && profileRes.data) {
          localStorage.setItem('plan', profileRes.data.plan || 'lite')
          if (profileRes.data.business_name || data.data.role !== 'owner') {
            localStorage.setItem('onboarding_completed', 'true')
          }
          if (typeof profileRes.data.is_frozen === 'boolean') {
            sessionStorage.setItem('subscription_status', profileRes.data.is_frozen ? 'frozen' : 'active')
          } else {
            sessionStorage.setItem('subscription_status', 'active')
          }
        }
      } catch (err) {
        console.error('Failed to check profile for onboarding status')
      }

      router.push('/')
    } else {
      errorMsg.value = data.message || 'OTP tidak valid.'
    }
  } catch (e) {
    errorMsg.value = 'Terjadi kesalahan jaringan.'
  } finally {
    loading.value = false
  }
}

const handleLogin = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const data = await api.login(username.value, password.value)

    if (data.success && data.data) {
      localStorage.setItem('access_token', data.data.accessToken)
      localStorage.setItem('refresh_token', data.data.refreshToken)
      localStorage.setItem('tenant_id', data.data.tenantId)
      localStorage.setItem('role', data.data.role)

      try {
        const profileRes = await api.get('/api/profile')
        if (profileRes.success && profileRes.data) {
          localStorage.setItem('plan', profileRes.data.plan || 'lite')
          if (profileRes.data.business_name || data.data.role !== 'owner') {
            localStorage.setItem('onboarding_completed', 'true')
          }
          if (typeof profileRes.data.is_frozen === 'boolean') {
            sessionStorage.setItem('subscription_status', profileRes.data.is_frozen ? 'frozen' : 'active')
          } else {
            sessionStorage.setItem('subscription_status', 'active')
          }
        }
      } catch (err) {
        console.error('Failed to check profile for onboarding status')
      }

      router.push('/')
    } else {
      errorMsg.value = data.message || 'Login gagal, periksa kredensial Anda.'
    }
  } catch (e) {
    errorMsg.value = 'Terjadi kesalahan jaringan ke server Auth.'
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
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.8), rgba(30, 58, 138, 0.9)), url('https://images.unsplash.com/photo-1554224155-6726b3ff858f?auto=format&fit=crop&q=80');
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
    padding: 2rem;
    max-width: 480px;
  }
}

@media (min-width: 640px) {
  .auth-form-wrapper {
    padding: 2.5rem;
    max-width: 520px;
  }
}

.login-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  background: var(--bg-tertiary);
  border-radius: var(--radius-sm);
  padding: 0.25rem;
}

.tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  padding: 0.6rem 0.35rem;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font-family: inherit;
  font-size: 0.78rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  overflow: hidden;
}

@media (min-width: 480px) {
  .tab-btn {
    gap: 0.5rem;
    padding: 0.6rem 0.65rem;
    font-size: 0.82rem;
  }
}

@media (min-width: 640px) {
  .tab-btn {
    padding: 0.6rem 1rem;
    font-size: 0.88rem;
  }
}

.tab-btn.active {
  background: var(--surface-0);
  color: var(--accent-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.tab-btn:hover:not(.active) {
  color: var(--text-primary);
}

@media (max-width: 480px) {
  .auth-split-container {
    border-radius: 0;
  }

  .tab-btn svg {
    display: none;
  }
}
</style>