import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Dashboard from './views/Dashboard.vue'
import VoucherPrograms from './views/VoucherPrograms.vue'
import GenerateVouchers from './views/GenerateVouchers.vue'
import FrozenAccounts from './views/FrozenAccounts.vue'
import PlanFeatures from './views/PlanFeatures.vue'
import './style.css'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Dashboard, name: 'dashboard' },
    { path: '/vouchers/programs', component: VoucherPrograms, name: 'voucher-programs' },
    { path: '/vouchers/generate', component: GenerateVouchers, name: 'generate-vouchers' },
    { path: '/frozen-accounts', component: FrozenAccounts, name: 'frozen-accounts' },
    { path: '/plan-features', component: PlanFeatures, name: 'plan-features' },
  ],
})

createApp(App).use(router).mount('#app')
