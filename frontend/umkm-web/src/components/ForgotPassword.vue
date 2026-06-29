<template>
  <div class="auth-split-container">
    <div class="auth-image-side">
      <div class="glass-overlay">
        <h1 class="text-accent-primary">WCH UMKM</h1>
        <p>Dashboard terpadu dengan asisten AI akuntansi pertama di Indonesia.</p>
      </div>
    </div>

    <div class="auth-form-side">
      <div class="auth-form-wrapper glass-card animate-fade-in">

        <div style="margin-bottom: 1.5rem;">
          <h2>Lupa Password?</h2>
          <p class="text-muted">
            Reset password dilakukan langsung via chat — tidak perlu email.
            Pilih saluran yang Anda gunakan:
          </p>
        </div>

        <!-- Tab navigation -->
        <div class="reset-tabs">
          <button
            class="reset-tab"
            :class="{ active: activeTab === 'wa' }"
            @click="activeTab = 'wa'"
          >
            💬 WhatsApp
          </button>
          <button
            class="reset-tab"
            :class="{ active: activeTab === 'telegram' }"
            @click="activeTab = 'telegram'"
          >
            ✈️ Telegram
          </button>
        </div>

        <!-- WA instructions -->
        <div v-if="activeTab === 'wa'" class="reset-content">
          <div class="info-box">
            <strong>Buka chat WA Anda</strong> ke nomor bisnis WCH, lalu ketik salah satu command berikut:
          </div>

          <div class="command-box">
            <code>RESET</code>
            <span class="cmd-desc">atau</span>
            <code>LUPA SANDI</code>
            <span class="cmd-desc">atau</span>
            <code>LUPA PASSWORD</code>
          </div>

          <div class="flow-card">
            <div class="flow-step">
              <span class="flow-num">1</span>
              <div class="flow-body">
                <strong>Kirim command di atas</strong>
                <p>Buka WA → chat ke nomor bisnis WCH → ketik <code>RESET</code></p>
              </div>
            </div>
            <div class="flow-step">
              <span class="flow-num">2</span>
              <div class="flow-body">
                <strong>Sistem akan balas</strong>
                <p>Meminta nomor HP terdaftar Anda</p>
              </div>
            </div>
            <div class="flow-step">
              <span class="flow-num">3</span>
              <div class="flow-body">
                <strong>Kirim nomor HP</strong>
                <p>Contoh: <code>0812xxxxxxxx</code></p>
              </div>
            </div>
            <div class="flow-step">
              <span class="flow-num">4</span>
              <div class="flow-body">
                <strong>Kode OTP dikirim via WA</strong>
                <p>Balas pesan WA dengan 6 digit kode OTP tersebut</p>
              </div>
            </div>
            <div class="flow-step">
              <span class="flow-num">5</span>
              <div class="flow-body">
                <strong>Kirim password baru</strong>
                <p>Minimal 8 karakter. Contoh: <code>Bisnis123</code></p>
              </div>
            </div>
            <div class="flow-step done">
              <span class="flow-num">✓</span>
              <div class="flow-body">
                <strong>Password berhasil direset!</strong>
                <p>Login dengan password baru Anda</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Telegram instructions -->
        <div v-else class="reset-content">
          <div class="info-box">
            <strong>Buka Telegram</strong> → cari bot <strong>WCH Platform</strong>, lalu ketik command berikut:
          </div>

          <div class="command-box">
            <code>/reset_password</code>
          </div>

          <div class="flow-card">
            <div class="flow-step">
              <span class="flow-num">1</span>
              <div class="flow-body">
                <strong>Kirim <code>/reset_password</code></strong>
                <p>Buka Telegram → cari bot WCH → ketik <code>/reset_password</code></p>
              </div>
            </div>
            <div class="flow-step">
              <span class="flow-num">2</span>
              <div class="flow-body">
                <strong>Bot meminta nomor HP</strong>
                <p>Kirim nomor HP terdaftar Anda (format: <code>0812xxxxxxxx</code>)</p>
              </div>
            </div>
            <div class="flow-step">
              <span class="flow-num">3</span>
              <div class="flow-body">
                <strong>Kode OTP dikirim via Telegram</strong>
                <p>Balas dengan 6 digit kode OTP dari bot</p>
              </div>
            </div>
            <div class="flow-step">
              <span class="flow-num">4</span>
              <div class="flow-body">
                <strong>Kirim password baru</strong>
                <p>Minimal 8 karakter. Contoh: <code>Bisnis123</code></p>
              </div>
            </div>
            <div class="flow-step done">
              <span class="flow-num">✓</span>
              <div class="flow-body">
                <strong>Password berhasil direset!</strong>
                <p>Login dengan password baru Anda</p>
              </div>
            </div>
          </div>
        </div>

        <p class="text-center" style="margin-top: 2rem; color: var(--text-secondary);">
          Ingat kata sandi Anda?
          <router-link to="/login" style="color: var(--accent-primary);">Masuk di sini</router-link>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const activeTab = ref<'wa' | 'telegram'>('wa')
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
  .auth-image-side { display: block; }
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
  max-width: 500px;
  padding: 2.5rem;
}

.reset-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
  border-bottom: 2px solid var(--surface-2, #1e293b);
  padding-bottom: 0;
}

.reset-tab {
  background: none;
  border: none;
  padding: 0.5rem 1rem;
  cursor: pointer;
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-secondary);
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  transition: color 0.2s, border-color 0.2s;
  border-radius: 6px 6px 0 0;
}

.reset-tab.active {
  color: var(--accent-primary);
  border-bottom-color: var(--accent-primary);
}

.info-box {
  background: var(--surface-2, #1e293b);
  border-left: 3px solid var(--accent-primary);
  padding: 0.75rem 1rem;
  border-radius: 6px;
  font-size: 0.88rem;
  color: var(--text-secondary);
  margin-bottom: 1rem;
}

.info-box strong {
  color: var(--text-primary);
}

.command-box {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  background: rgba(59, 130, 246, 0.1);
  border: 1px dashed rgba(59, 130, 246, 0.4);
  border-radius: 8px;
  padding: 0.75rem 1rem;
  margin-bottom: 1.25rem;
}

.command-box code {
  background: var(--accent-primary);
  color: white;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 600;
  letter-spacing: 0.03em;
}

.cmd-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.flow-card {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.flow-step {
  display: flex;
  gap: 0.75rem;
  align-items: flex-start;
  padding: 0.6rem 0.75rem;
  border-radius: 8px;
  background: var(--surface-1, #1e293b);
}

.flow-step.done {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.flow-num {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--accent-primary);
  color: white;
  font-size: 0.72rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 1px;
}

.flow-step.done .flow-num {
  background: #10b981;
}

.flow-body strong {
  font-size: 0.85rem;
  display: block;
  margin-bottom: 1px;
}

.flow-body p {
  font-size: 0.78rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.4;
}

code {
  background: var(--surface-2, #1e293b);
  padding: 1px 5px;
  border-radius: 4px;
  font-size: 0.82em;
  color: var(--accent-primary);
}

.flow-body code {
  background: rgba(59, 130, 246, 0.15);
}
</style>
