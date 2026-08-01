import { createRouter, createWebHistory } from 'vue-router'

import { api, sanitizeJWT, sanitizeUUID, sanitizeRole, sanitizeText, sanitizeBoolean } from '../api'
import Dashboard from '../components/Dashboard.vue'
import DynamicDashboard from '../components/DynamicDashboard.vue'
import Onboarding from '../components/Onboarding.vue'
import Journal from '../components/Journal.vue'
import ProductCatalog from '../components/ProductCatalog.vue'
import POS from '../components/POS.vue'
import SuperAdminDashboard from '../components/SuperAdminDashboard.vue'
import Settings from '../components/Settings.vue'
import Login from '../components/Login.vue'
import Register from '../components/Register.vue'
import SuperAdminLogin from '../components/SuperAdminLogin.vue'
import ForgotPassword from '../components/ForgotPassword.vue'
import Automations from '../components/Automations.vue'
import WASetup from '../components/WASetup.vue'
import DataTransfer from '../components/DataTransfer.vue'
import Reports from '../components/Reports.vue'
import ClinicQueue from '../components/ClinicQueue.vue'
import ClinicFrontdesk from '../components/ClinicFrontdesk.vue'
import AffiliateDashboard from '../components/AffiliateDashboard.vue'
import AffiliateLeaderboard from '../components/AffiliateLeaderboard.vue'
import Wallet from '../components/Wallet.vue'
import Addons from '../components/Addons.vue'
import LandingPage from '../components/LandingPage.vue'

const routes = [
  // Landing page: public, dedicated URL
  { path: '/landing', component: LandingPage, name: 'Landing', meta: { public: true } },
  // Root redirects to login (authed → dashboard via beforeEach)
  { path: '/', redirect: '/login' },
  { path: '/dashboard', component: DynamicDashboard, name: 'DynamicDashboard' },
  { path: '/dashboard-classic', component: Dashboard, name: 'DashboardClassic' },
  { path: '/onboarding', component: Onboarding, name: 'Onboarding' },
  { path: '/journal', component: Journal, name: 'Journal' },
  { path: '/reports', component: Reports, name: 'Reports' },
  { path: '/catalog', component: ProductCatalog, name: 'ProductCatalog' },
  { path: '/pos', component: POS, name: 'POS' },
  { path: '/superadmin', component: SuperAdminDashboard, name: 'SuperAdminDashboard' },
  { path: '/settings', component: Settings, name: 'Settings' },
  { path: '/automations', component: Automations, name: 'Automations' },
  { path: '/wa-setup', component: WASetup, name: 'WASetup' },
  { path: '/data-transfer', component: DataTransfer, name: 'DataTransfer' },
  { path: '/login', component: Login, name: 'Login', meta: { requiresGuest: true } },
  { path: '/register', component: Register, name: 'Register', meta: { requiresGuest: true } },
  { path: '/clinic', component: ClinicQueue, name: 'ClinicQueue' },
  { path: '/clinic/frontdesk', component: ClinicFrontdesk, name: 'ClinicFrontdesk' },
  // F047: Rekam Medis + Jadwal Dokter + Notifikasi WA Klinik — all use the same component with tabs
  { path: '/clinic/medical-record', redirect: '/clinic/frontdesk?tab=records' },
  { path: '/clinic/schedule', redirect: '/clinic/frontdesk?tab=doctors' },
  { path: '/clinic/notifications', redirect: '/clinic/frontdesk?tab=notifications' },
  { path: '/superadmin-login', component: SuperAdminLogin, name: 'SuperAdminLogin', meta: { requiresGuest: true } },
  { path: '/forgot-password', component: ForgotPassword, name: 'ForgotPassword', meta: { requiresGuest: true } },
  // Legacy token-based reset — dead endpoint, redirect to the chat-based flow
  { path: '/reset-password', redirect: '/forgot-password' },
  // F036: Affiliate
  { path: '/affiliate', component: AffiliateDashboard, name: 'AffiliateDashboard' },
  { path: '/leaderboard', component: AffiliateLeaderboard, name: 'AffiliateLeaderboard', meta: { public: true } },
  { path: '/wallet', component: Wallet, name: 'Wallet' },
  // F053: Addon marketplace page
  { path: '/addons', component: Addons, name: 'Addons' },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// In-memory cache of the last /api/me result, used to avoid hammering the
// backend on every navigation. Keyed by tenant_id+user_id pair.
let _meCache: { key: string; ts: number; data: any } | null = null
const ME_CACHE_TTL_MS = 30_000 // 30s

function syncUserDataToStorage(data: any) {
  if (data.onboarding_completed !== undefined) {
    localStorage.setItem('onboarding_completed', sanitizeBoolean(data.onboarding_completed ? 'true' : 'false'))
  }
  if (data.plan !== undefined) {
    localStorage.setItem('plan', sanitizeText(data.plan || '', 50))
  }
  if (data.role !== undefined) {
    localStorage.setItem('role', sanitizeRole(data.role || ''))
  }
  if (data.is_frozen !== undefined) {
    const status = data.is_frozen ? 'frozen' : 'active'
    sessionStorage.setItem('subscription_status', sanitizeText(status, 20))
  }
  if (data.addons !== undefined && Array.isArray(data.addons)) {
    const sanitized = data.addons.map((a: any) => ({
      id: sanitizeUUID(a.id),
      name: sanitizeText(a.name || '', 100)
    }))
    localStorage.setItem('tenant_addons', JSON.stringify(sanitized))
  }
  if (data.must_change_password !== undefined) {
    localStorage.setItem('must_change_password', sanitizeBoolean(data.must_change_password ? 'true' : 'false'))
  }
}

async function fetchAndSyncMe(): Promise<any | null> {
  const token = localStorage.getItem('access_token')
  const tenantId = localStorage.getItem('tenant_id')
  if (!token || !tenantId) return null
  const key = `${tenantId}:${localStorage.getItem('user_id') || ''}`
  const now = Date.now()
  if (_meCache && _meCache.key === key && (now - _meCache.ts) < ME_CACHE_TTL_MS) {
    return _meCache.data
  }
  const res = await api.me()
  if (res && res.success && res.data) {
    _meCache = { key, ts: now, data: res.data }
    syncUserDataToStorage(res.data)
  }
  return res && res.success ? res.data : null
}

// Helper: handle impersonate auto-login via query param
async function handleImpersonateLogin(to: any, next: any): Promise<boolean> {
  const impersonateToken = to.query.impersonate_token as string | undefined
  if (!impersonateToken) return false

  try {
    const res = await fetch('/api/auth/validate', {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${impersonateToken}` }
    })
    const data = await res.json()
    if (res.ok && data.success) {
      localStorage.setItem('access_token', sanitizeJWT(impersonateToken))
      localStorage.setItem('tenant_id', sanitizeUUID(data.data.tenant_id))
      localStorage.setItem('user_id', sanitizeUUID(data.data.user_id))
      localStorage.setItem('role', sanitizeRole(data.data.role))
      if (data.data.impersonated_by) {
        sessionStorage.setItem('impersonated_by', data.data.impersonated_by)
      }
      await fetchAndSyncMe()
      next({ path: '/dashboard', replace: true })
      return true
    }
  } catch (err) {
    console.error('Impersonate auto-login failed:', err)
  }
  return false
}

// Helper: handle guest routes (login/register)
function handleGuestRoute(to: any, next: any, isLoggedIn: boolean, isSuperadmin: boolean): boolean {
  if (!to.meta.requiresGuest) return false

  if (isLoggedIn || isSuperadmin) {
    next({ path: '/dashboard' })
  } else {
    next()
  }
  return true
}

// Helper: handle authenticated routes with onboarding/frozen checks
async function handleAuthenticatedRoute(to: any, next: any, isSuperadmin: boolean): Promise<void> {
  // Re-sync if onboarding flag missing (fixes redirect loop on new device)
  const onboardingDone = localStorage.getItem('onboarding_completed')
  if (!onboardingDone && !isSuperadmin) {
    try {
      await fetchAndSyncMe()
    } catch (e) {
      // Fall through to existing behavior (redirect to onboarding)
    }
  }

  if (to.path !== '/onboarding' && to.path !== '/login' && to.path !== '/register') {
    const onboardingDone2 = localStorage.getItem('onboarding_completed')
    if (!onboardingDone2 && !isSuperadmin) {
      next({ path: '/onboarding' })
      return
    }

    const subscriptionStatus = sessionStorage.getItem('subscription_status')
    if (subscriptionStatus === 'frozen' && !isSuperadmin) {
      if (to.path !== '/onboarding' && to.path !== '/settings') {
        next({ path: '/onboarding' })
        return
      }
    }
  }
  next()
}

router.beforeEach(async (to, _from, next) => {
  // F058: Impersonate auto-login
  if (await handleImpersonateLogin(to, next)) return

  // Public routes bypass
  if (to.meta.public) {
    next()
    return
  }

  const token = localStorage.getItem('access_token')
  const tenantId = localStorage.getItem('tenant_id')
  const role = localStorage.getItem('role')
  const isSuperadmin = role === 'superadmin'
  const isLoggedIn = !!(token && tenantId)

  // Guest routes (login/register)
  if (handleGuestRoute(to, next, isLoggedIn, isSuperadmin)) return

  // Authenticated routes
  if (isLoggedIn || isSuperadmin) {
    await handleAuthenticatedRoute(to, next, isSuperadmin)
  } else {
    next({ path: '/login' })
  }
})

export default router
