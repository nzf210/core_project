import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Login from './views/Login.vue'
import Dashboard from './views/Dashboard.vue'
import TenantManagement from './views/TenantManagement.vue'
import LandingEditor from './views/LandingEditor.vue'
import VoucherPrograms from './views/VoucherPrograms.vue'
import VoucherAnalytics from './views/VoucherAnalytics.vue'
import GenerateVouchers from './views/GenerateVouchers.vue'
import FrozenAccounts from './views/FrozenAccounts.vue'
import PlanFeatures from './views/PlanFeatures.vue'
import FeatureMatrix from './views/FeatureMatrix.vue'
import AddonPricing from './views/AddonPricing.vue'
import ReferralConfig from './views/ReferralConfig.vue'
import CampaignLicenses from './views/CampaignLicenses.vue'
import CoordinatorAssignment from './components/CoordinatorAssignment.vue'
import './style.css'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: Login, name: 'login', meta: { public: true } },
    { path: '/', component: Dashboard, name: 'dashboard' },
    { path: '/tenants', component: TenantManagement, name: 'tenant-management' },
    { path: '/landing-editor', component: LandingEditor, name: 'landing-editor' },
    { path: '/vouchers/programs', component: VoucherPrograms, name: 'voucher-programs' },
    { path: '/vouchers/generate', component: GenerateVouchers, name: 'generate-vouchers' },
    { path: '/vouchers/analytics', component: VoucherAnalytics, name: 'voucher-analytics' },
    { path: '/frozen-accounts', component: FrozenAccounts, name: 'frozen-accounts' },
    { path: '/plan-features', component: PlanFeatures, name: 'plan-features' },
    { path: '/feature-matrix', component: FeatureMatrix, name: 'feature-matrix' },
    { path: '/addon-pricing', component: AddonPricing, name: 'addon-pricing' },
    { path: '/referral-config', component: ReferralConfig, name: 'referral-config' },
    { path: '/campaign-licenses', component: CampaignLicenses, name: 'campaign-licenses' },
    { path: '/coordinator-assignment', component: CoordinatorAssignment, name: 'coordinator-assignment' },
  ],
})

// Navigation guard: redirect to login if not authenticated
router.beforeEach((to, _from, next) => {
  if (to.meta.public) return next()
  const token = localStorage.getItem('access_token')
  if (!token) return next('/login')
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    if (payload.role !== 'superadmin') {
      localStorage.removeItem('access_token')
      return next('/login')
    }
  } catch {
    localStorage.removeItem('access_token')
    return next('/login')
  }
  next()
})

createApp(App).use(router).mount('#app')
