<template>
  <div class="app-container">
    <!-- Read-only banner: tampil saat akun freeze. User bisa login& lihat data, tapi tidak bisa input baru. -->
    <div v-if="isLoggedIn && isFrozen" class="frozen-banner">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
      <div class="frozen-text">
        <strong>Akun Anda dalam masa freeze.</strong>
        Anda masih bisa melihat data historis, tetapi tidak bisa input transaksi baru.
        <a href="/superadmin-login" class="redeem-link">Redeem voucher →</a>
      </div>
    </div>

    <!-- Mobile Menu Button -->
    <button
      v-if="isLoggedIn"
      class="mobile-menu-btn"
      @click="isMobileMenuOpen = true"
    >
      ☰
    </button>

    <!-- Sidebar -->
    <AppSidebar
      v-if="isLoggedIn"
      :is-open="isMobileMenuOpen"
      :user-role="userRole"
      :business-name="businessName"
      :plan="plan"
      :business-type="businessType"
      :is-frozen="isFrozen"
      @close="isMobileMenuOpen = false"
    />

    <!-- Main Content -->
    <main class="app-main" :class="{ 'with-sidebar': isLoggedIn, 'frozen-active': isFrozen }">
      <div class="container animate-fade-in">
        <router-view />
      </div>
    </main>

    <Chatbot v-if="isLoggedIn" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { api } from './api'
import Chatbot from './components/Chatbot.vue'
import AppSidebar from './components/AppSidebar.vue'
import { useTheme } from './composables/useTheme'

const route = useRoute()

// Initialize theme globally
useTheme()

const isLoggedIn = ref(false)
const userRole = ref('user')
const isMobileMenuOpen = ref(false)
const businessName = ref('')
const businessType = ref('umum')
const plan = ref('lite')
const isFrozen = ref(false)

const checkAuth = async () => {
  const role = localStorage.getItem('role')
  const isSuperadmin = role === 'superadmin'
  const hasToken = !!localStorage.getItem('access_token')
  const hasTenantId = !!localStorage.getItem('tenant_id')

  if (hasToken && (hasTenantId || isSuperadmin)) {
    isLoggedIn.value = true
    userRole.value = role || 'user'
    businessName.value = localStorage.getItem('business_name') || ''
    plan.value = localStorage.getItem('plan') || 'lite'

    const cachedStatus = sessionStorage.getItem('subscription_status')
    isFrozen.value = cachedStatus === 'frozen'

    if (hasTenantId) {
      try {
        const data = await api.get('/api/profile')
        if (data.success && data.data) {
          if (data.data.business_name) {
            businessName.value = data.data.business_name
            localStorage.setItem('business_name', data.data.business_name)
          }
          if (data.data.plan) {
            plan.value = data.data.plan
            localStorage.setItem('plan', data.data.plan)
          }
          if (data.data.business_type) {
            businessType.value = data.data.business_type
            localStorage.setItem('business_type', data.data.business_type)
          }
        }
        if (data.data && typeof data.data.is_frozen === 'boolean') {
          isFrozen.value = data.data.is_frozen
          sessionStorage.setItem('subscription_status', isFrozen.value ? 'frozen' : 'active')
        }
      } catch (e) {
        console.error('Failed to sync profile', e)
      }
    }
  } else {
    isLoggedIn.value = false
  }
}

watch(() => route.path, () => {
  checkAuth()
})

onMounted(() => {
  checkAuth()
})
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.frozen-banner {
  background: linear-gradient(90deg, #f59e0b, #ef4444);
  color: white;
  padding: 10px 20px;
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 110;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
}

.frozen-icon { font-size: 20px; }
.frozen-text { flex: 1; }
.redeem-link {
  color: white;
  text-decoration: underline;
  font-weight: 600;
  margin-left: 8px;
  padding: 4px 10px;
  background: rgba(255,255,255,0.2);
  border-radius: 4px;
}
.redeem-link:hover { background: rgba(255,255,255,0.3); }

.mobile-menu-btn {
  display: none;
  position: fixed;
  top: 1rem;
  left: 1rem;
  z-index: 60;
  background: var(--surface-0, #ffffff);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  padding: 0.75rem;
  font-size: 1.25rem;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.app-main {
  flex: 1;
  padding: 2rem;
  transition: margin-left 0.3s ease;
}

.app-main.with-sidebar {
  margin-left: 260px;
}

.app-main.frozen-active {
  padding-top: 4rem;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .mobile-menu-btn {
    display: block;
  }

  .app-main.with-sidebar {
    margin-left: 0;
    padding-top: 4rem;
    padding-left: 1rem;
    padding-right: 1rem;
  }
}
</style>