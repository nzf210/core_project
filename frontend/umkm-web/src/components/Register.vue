<template>
  <div class="auth-split-container">
    <div class="auth-image-side">
      <div class="glass-overlay">
        <h1 class="text-gradient">WCH UMKM</h1>
        <p>Bergabunglah dengan ribuan UMKM lain yang telah mentransformasi bisnis mereka dengan bantuan AI.</p>
      </div>
    </div>

    <div class="auth-form-side">
      <div class="auth-form-wrapper glass-card animate-fade-in">
        <div style="margin-bottom: 2rem;">
          <h2>Daftar Akun Baru</h2>
          <p class="text-muted">Mulai kelola keuangan dan AI Chatbot Anda sekarang</p>
        </div>

        <form @submit.prevent="step === 'form' ? handleRegister() : handleVerifyOTP()">
          <!-- Channel Toggle -->
          <div v-if="step === 'form'" class="login-tabs" style="margin-bottom: 1.5rem;">
            <button type="button" :class="['tab-btn', { active: registerChannel === 'wa' }]"
              @click="registerChannel = 'wa'">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2">
                <path d="M3 21l1.65-3.8a9 9 0 1 1 3.4 2.9L3 21" />
                <path
                  d="M9 10a.5.5 0 0 0 1 0V9a.5.5 0 0 0-1 0v1Zm0 0a5 5 0 0 0 5 5h1a.5.5 0 0 0 0-1h-1a.5.5 0 0 0 0 1" />
              </svg>
              WhatsApp
            </button>
            <button type="button" :class="['tab-btn', { active: registerChannel === 'telegram' }]"
              @click="registerChannel = 'telegram'">
              <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2">
                <path d="M21.5 2.5 2.5 10.5 10 14l3-10 8.5-1.5Z" />
                <path d="M10 14v6.5l4-3" />
                <path d="m2.5 10.5 19-8" />
              </svg>
              Telegram
            </button>
          </div>

          <!-- Step 1: Form Input -->
          <template v-if="step === 'form'">
            <div class="form-group">
              <label>Nama Usaha (Toko/Perusahaan)</label>
              <input v-model="formData.name" type="text" class="form-control" placeholder="cth: Kedai Kopi Senja"
                required />
            </div>

            <div class="form-group">
              <label>Username Pemilik</label>
              <input v-model="formData.username"
                @input="formData.username = formData.username.replace(/ /g, '_').toLowerCase()" type="text"
                class="form-control" placeholder="cth: owner_kopi" required />
            </div>

            <div class="form-group">
              <label>Nomor WhatsApp <span style="color: var(--accent-primary);">*</span></label>
              <input v-model="formData.phone_number" type="text" class="form-control" placeholder="cth: 081234567890"
                required />
              <p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">
                {{ registerChannel === 'telegram' ? 'Kode OTP akan dikirim ke Telegram Anda' : `Kode OTP akan dikirim ke
                WhatsApp Anda` }}
              </p>
            </div>

            <div v-if="registerChannel === 'telegram'" class="form-group">
              <label>Chat ID Telegram <span style="color: var(--accent-primary);">*</span></label>
              <input v-model="formData.telegramChatId" type="text" class="form-control" placeholder="cth: 123456789"
                required />
              <p style="font-size: 0.75rem; color: var(--text-secondary); margin-top: 0.25rem;">
                Chat bot kami di <a href="https://t.me/WCHBot" target="_blank"
                  style="color: var(--accent-primary);">@core_tesbot</a> — kirim /start untuk dapat Chat ID
              </p>
            </div>

            <div class="form-group">
              <label>Email <span style="color: var(--text-secondary); font-size: 0.8rem;">(opsional)</span></label>
              <input v-model="formData.email" type="email" class="form-control"
                placeholder="cth: owner@kopisenja.com" />
            </div>

            <div class="form-group" style="margin-bottom: 2rem;">
              <label>Password</label>
              <div style="position: relative;">
                <input v-model="formData.password" :type="showPassword ? 'text' : 'password'" class="form-control"
                  placeholder="Minimal 8 karakter" required style="padding-right: 2.5rem;" />
                <button type="button" @click="showPassword = !showPassword"
                  style="position: absolute; right: 0.75rem; top: 50%; transform: translateY(-50%); background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0; display: flex;">
                  <svg v-if="!showPassword" xmlns="http://www.w3.org/2000/svg" width="18" height="18"
                    viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"
                    stroke-linejoin="round">
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
              {{ loading ? 'Mengirim OTP...' : 'Daftar Sekarang' }}
            </button>
          </template>

          <!-- Step 2: OTP Verification -->
          <template v-if="step === 'verify'">
            <div style="text-align: center; margin-bottom: 1.5rem;">
              <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none"
                stroke="#10b981" stroke-width="2">
                <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                <polyline points="22 4 12 14.01 9 11.01" />
              </svg>
              <p style="color: var(--text-secondary); margin-top: 0.75rem;">
                Kode OTP telah dikirim ke <strong>{{ registerChannel === 'telegram' ? 'Telegram Anda' :
                  formData.phone_number }}</strong>
              </p>
            </div>

            <div class="form-group">
              <label>Masukkan Kode OTP</label>
              <input v-model="otpCode" type="text" class="form-control otp-input" placeholder="000000" maxlength="6"
                required />
            </div>

            <button type="submit" class="btn btn-primary" style="width: 100%; padding: 0.75rem;" :disabled="loading">
              {{ loading ? 'Memverifikasi...' : 'Verifikasi & Buat Akun' }}
            </button>

            <p style="text-align: center; margin-top: 1rem; font-size: 0.85rem;">
              <a href="#" @click.prevent="step = 'form'" style="color: var(--accent-primary);">← Kembali, ganti
                nomor</a>
            </p>
          </template>

          <div v-if="errorMsg" class="error-msg text-center"
            style="margin-top: 1rem; color: #ef4444; font-size: 0.875rem;">
            {{ errorMsg }}
          </div>
          <div v-if="successMsg" class="success-msg text-center"
            style="margin-top: 1rem; color: #10b981; font-size: 0.875rem;">
            {{ successMsg }}
          </div>
        </form>

        <p class="text-center" style="margin-top: 2rem; color: var(--text-secondary);">
          Sudah punya akun? <router-link to="/login" style="color: var(--accent-primary);">Masuk di sini</router-link>
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

const step = ref<'form' | 'verify'>('form')
const registerChannel = ref<'wa' | 'telegram'>('wa')
const loading = ref(false)
const showPassword = ref(false)
const errorMsg = ref('')
const successMsg = ref('')
const otpCode = ref('')

const formData = ref({
  name: '',
  username: '',
  email: '',
  phone_number: '',
  telegramChatId: '',
  password: '',
  plan: 'lite'
})

const handleRegister = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    let data
    if (registerChannel.value === 'telegram') {
      data = await api.telegramRegister({
        telegramChatId: formData.value.telegramChatId,
        phoneNumber: formData.value.phone_number,
        password: formData.value.password,
        username: formData.value.username,
        email: formData.value.email || formData.value.phone_number + '@wa.user',
        businessName: formData.value.name,
      })
    } else {
      data = await api.registerWA({
        phoneNumber: formData.value.phone_number,
        password: formData.value.password,
        username: formData.value.username,
        email: formData.value.email || formData.value.phone_number + '@wa.user',
        businessName: formData.value.name,
      })
    }
    if (data.success) {
      step.value = 'verify'
      successMsg.value = data.message || 'OTP telah dikirim.'
    } else {
      errorMsg.value = data.message || 'Gagal mengirim OTP.'
    }
  } catch (e) {
    errorMsg.value = 'Kesalahan jaringan. Pastikan backend menyala.'
  } finally {
    loading.value = false
  }
}

const handleVerifyOTP = async () => {
  loading.value = true
  errorMsg.value = ''
  successMsg.value = ''

  try {
    const data = await api.verifyOTP(formData.value.phone_number, otpCode.value)
    if (data.success) {
      router.push('/login')
    } else {
      errorMsg.value = data.message || 'OTP tidak valid.'
    }
  } catch (e) {
    errorMsg.value = 'Kesalahan jaringan.'
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
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary)), url('https://images.unsplash.com/photo-1556742049-0cfed4f6a45d?auto=format&fit=crop&q=80');
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
  .tab-btn svg {
    display: none;
  }
}

.otp-input {
  text-align: center;
  font-size: 1.5rem;
  letter-spacing: 0.5rem;
  font-weight: 700;
}

@media (max-width: 480px) {
  .auth-split-container {
    border-radius: 0;
  }
}
</style>
