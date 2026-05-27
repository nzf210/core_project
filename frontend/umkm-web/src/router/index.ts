import { createRouter, createWebHistory } from 'vue-router'

import Dashboard from '../components/Dashboard.vue'
import DynamicDashboard from '../components/DynamicDashboard.vue'
import Onboarding from '../components/Onboarding.vue'
import Journal from '../components/Journal.vue'
import ProductCatalog from '../components/ProductCatalog.vue'
import POS from '../components/POS.vue'
import SuperAdminDashboard from '../components/SuperAdminDashboard.vue'
import SuperAdminN8n from '../components/SuperAdminN8n.vue'
import Settings from '../components/Settings.vue'
import Login from '../components/Login.vue'
import Register from '../components/Register.vue'
import SuperAdminLogin from '../components/SuperAdminLogin.vue'
import ForgotPassword from '../components/ForgotPassword.vue'
import ResetPassword from '../components/ResetPassword.vue'
import Automations from '../components/Automations.vue'

const routes = [
  { path: '/', component: DynamicDashboard, name: 'Dashboard' },
  { path: '/dashboard', component: DynamicDashboard, name: 'DynamicDashboard' },
  { path: '/dashboard-classic', component: Dashboard, name: 'DashboardClassic' },
  { path: '/onboarding', component: Onboarding, name: 'Onboarding' },
  { path: '/journal', component: Journal, name: 'Journal' },
  { path: '/catalog', component: ProductCatalog, name: 'ProductCatalog' },
  { path: '/pos', component: POS, name: 'POS' },
  { path: '/superadmin', component: SuperAdminDashboard, name: 'SuperAdminDashboard' },
  { path: '/superadmin/n8n', component: SuperAdminN8n, name: 'SuperAdminN8n' },
  { path: '/settings', component: Settings, name: 'Settings' },
  { path: '/automations', component: Automations, name: 'Automations' },
  { path: '/login', component: Login, name: 'Login', meta: { requiresGuest: true } },
  { path: '/register', component: Register, name: 'Register', meta: { requiresGuest: true } },
  { path: '/superadmin-login', component: SuperAdminLogin, name: 'SuperAdminLogin', meta: { requiresGuest: true } },
  { path: '/forgot-password', component: ForgotPassword, name: 'ForgotPassword', meta: { requiresGuest: true } },
  { path: '/reset-password', component: ResetPassword, name: 'ResetPassword', meta: { requiresGuest: true }, props: (route: any) => ({ initialEmail: route.query.email }) }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('access_token')
  const tenantId = localStorage.getItem('tenant_id')
  const isLoggedIn = !!(token && tenantId)

  if (to.meta.requiresGuest) {
    if (isLoggedIn) {
      next({ path: '/' })
    } else {
      next()
    }
  } else {
    if (isLoggedIn) {
      if (to.path !== '/onboarding' && to.path !== '/login' && to.path !== '/register') {
        const onboardingDone = localStorage.getItem('onboarding_completed')
        if (!onboardingDone) {
          next({ path: '/onboarding' })
          return
        }
      }
      next()
    } else {
      next({ path: '/login' })
    }
  }
})

export default router
