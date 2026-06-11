<template>
  <div class="app-container">
    <!-- Read-only banner: tampil saat akun freeze. User bisa login& lihat data, tapi tidak bisa input baru. -->
    <div v-if="isLoggedIn && isFrozen" class="frozen-banner">
      <span class="frozen-icon">❄️</span>
      <div class="frozen-text">
        <strong>Akun Anda dalam masa freeze.</strong>
        Anda masih bisa melihat data historis, tetapi tidak bisa input transaksi baru.
        <a href="/superadmin-login" class="redeem-link">Redeem voucher →</a>
      </div>
    </div>

    <header class="app-header" v-if="isLoggedIn">
      <div class="container flex items-center justify-between">
        <h1 class="logo text-gradient">WCH UMKM</h1>
        
        <button class="mobile-menu-btn" @click="isMobileMenuOpen = !isMobileMenuOpen">
          <span v-if="!isMobileMenuOpen">☰</span>
          <span v-else>✕</span>
        </button>

        <nav :class="['nav-links', { 'is-open': isMobileMenuOpen }]">
          <router-link to="/" class="nav-btn" active-class="active" @click="closeMenu">Dashboard</router-link>
          <router-link to="/pos" class="nav-btn" active-class="active" @click="closeMenu">Kasir</router-link>
          <router-link to="/journal" class="nav-btn" active-class="active" @click="closeMenu">Jurnal Keuangan</router-link>
          <router-link to="/catalog" class="nav-btn" active-class="active" @click="closeMenu">Katalog Produk</router-link>
          <router-link v-if="userRole === 'admin' || userRole === 'superadmin'" to="/superadmin" class="nav-btn" active-class="active" @click="closeMenu">Super Admin</router-link>
          <router-link to="/automations" class="nav-btn" active-class="active" @click="closeMenu">Automasi</router-link>
          <router-link to="/settings" class="nav-btn" active-class="active" @click="closeMenu">Pengaturan</router-link>
          
          <div class="user-profile-mobile">
            <div class="flex items-center gap-2">
              <div class="avatar">{{ (businessName || 'U')[0].toUpperCase() }}</div>
              <div>
              <span class="business-name-display">{{ businessName || 'My UMKM' }}</span>
              <span v-if="plan !== 'free' && plan !== 'inactive'" :class="['plan-chip', `plan-${plan}`]">{{ plan.toUpperCase() }}</span>
              <span v-else-if="plan === 'inactive'" class="plan-chip plan-inactive">INACTIVE</span>
              <span v-else class="plan-chip plan-free">FREE</span>
            </div>
            </div>
            <button @click="logout" class="nav-btn text-danger mobile-logout-btn">Keluar</button>
          </div>
        </nav>

        <div class="user-profile flex items-center gap-4 desktop-only">
          <div class="flex items-center gap-2">
            <div class="avatar">{{ (businessName || 'U')[0].toUpperCase() }}</div>
            <span>{{ businessName || 'My UMKM' }}</span>
          </div>
          <button @click="logout" class="nav-btn text-danger" style="color: #ef4444; border: 1px solid #ef4444; padding: 0.25rem 0.75rem; border-radius: 4px;">Keluar</button>
        </div>
      </div>
    </header>

    <main class="app-main container animate-fade-in">
      <router-view />
      <Chatbot v-if="isLoggedIn" />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api } from './api'
import Chatbot from './components/Chatbot.vue'

const router = useRouter()
const route = useRoute()

const isLoggedIn = ref(false)
const userRole = ref('user')
const isMobileMenuOpen = ref(false)
const businessName = ref('')
const plan = ref('free')
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
    plan.value = localStorage.getItem('plan') || 'free'

    // Read X-Subscription-Status header yang di-set oleh RequireActiveSubscription middleware
    // Backend set header ini di setiap response. Frontend cache di sessionStorage supaya
    // tidak perlu parse header di setiap navigasi.
    const cachedStatus = sessionStorage.getItem('subscription_status')
    isFrozen.value = cachedStatus === 'frozen'

    if (hasTenantId) {
      try {
        const data = await api.get('/api/profile')
        if (data.success && data.data && data.data.business_name) {
          businessName.value = data.data.business_name
          localStorage.setItem('business_name', data.data.business_name)
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

const closeMenu = () => {
  isMobileMenuOpen.value = false
}

const logout = () => {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('tenant_id')
  localStorage.removeItem('role')
  isLoggedIn.value = false
  closeMenu()
  router.push('/login')
}

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

.app-header {
  background: var(--glass-bg, rgba(255, 255, 255, 0.95));
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--glass-border, #e5e7eb);
  padding: 1rem 0;
  position: sticky;
  top: 0;
  z-index: 10;
}

.logo {
  font-size: 1.5rem;
  margin: 0;
  font-weight: 700;
}

.nav-links {
  display: flex;
  gap: 1rem;
}

.nav-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 1rem;
  font-family: inherit;
  font-weight: 500;
  cursor: pointer;
  padding: 0.5rem 1rem;
  border-radius: var(--radius-sm, 0.375rem);
  transition: all 0.2s ease;
  text-decoration: none;
}

.nav-btn:hover {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.05);
}

.nav-btn.active {
  color: var(--accent-primary);
  background: rgba(59, 130, 246, 0.1);
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 500;
}

.business-name-display {
  display: block;
  font-size: 0.9rem;
}

.plan-chip {
  display: inline-block;
  font-size: 0.65rem;
  padding: 0.1rem 0.4rem;
  border-radius: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.plan-free { background: rgba(100, 116, 139, 0.15); color: #94a3b8; }
.plan-lite { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
.plan-pro { background: rgba(59, 130, 246, 0.15); color: #60a5fa; }
.plan-enterprise { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
.plan-inactive { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(to bottom right, var(--accent-primary), var(--accent-secondary, #1d4ed8));
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  color: white;
}

.app-main {
  flex: 1;
  padding: 2rem 1.5rem;
}

.frozen-banner {
  background: linear-gradient(90deg, #f59e0b, #ef4444);
  color: white;
  padding: 10px 20px;
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 14px;
  position: sticky;
  top: 0;
  z-index: 20;
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
  background: transparent;
  border: none;
  font-size: 1.5rem;
  color: var(--text-primary);
  cursor: pointer;
  padding: 0.5rem;
}

.user-profile-mobile {
  display: none;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .mobile-menu-btn {
    display: block;
  }
  
  .desktop-only {
    display: none !important;
  }

  .nav-links {
    display: none;
    flex-direction: column;
    position: absolute;
    top: 100%;
    left: 0;
    width: 100%;
    background: var(--surface-0, #ffffff);
    padding: 1rem;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
    gap: 0.5rem;
    border-bottom: 1px solid var(--border-color, #e5e7eb);
  }

  .nav-links.is-open {
    display: flex;
  }

  .nav-btn {
    width: 100%;
    text-align: left;
    padding: 0.75rem 1rem;
  }

  .user-profile-mobile {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--border-color, #e5e7eb);
  }

  .user-profile-mobile .flex {
    padding: 0 1rem;
  }

  .mobile-logout-btn {
    color: #ef4444 !important;
    border: 1px solid #ef4444 !important;
    border-radius: 4px;
    text-align: center;
  }
  
  .app-main {
    padding: 1rem;
  }
}
</style>
