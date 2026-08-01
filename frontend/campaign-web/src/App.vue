<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import Login from './components/Login.vue'
import GuestDashboard from './components/GuestDashboard.vue'
import CampaignDashboard from './components/CampaignDashboard.vue'
import Volunteer from './components/Volunteer.vue'
import Voter from './components/Voter.vue'
import TaskBoard from './components/TaskBoard.vue'
import AccessManager from './components/AccessManager.vue'
import ReportGenerator from './components/ReportGenerator.vue'
import Users from './components/Users.vue'
import Candidates from './components/Candidates.vue'
import NotificationBell from './components/NotificationBell.vue'
import DataVerification from './components/DataVerification.vue'
import RealCount from './components/RealCount.vue'

const isAuthenticated = ref(!!localStorage.getItem('accessToken'))
const userName = ref(localStorage.getItem('userName') || 'Admin')
const userRole = ref(localStorage.getItem('userRole') || 'admin')
const isDataVerified = ref(localStorage.getItem('isDataVerified') || 'false')

const currentTab = ref('dashboard')
const isSidebarOpen = ref(false)
const unauthMode = ref('guest') // 'guest' or 'login'
const registerRole = ref('')

const handleAuthRequired = () => {
  isAuthenticated.value = false
  unauthMode.value = 'guest'
}

onMounted(() => {
  window.addEventListener('auth-required', handleAuthRequired)
})

onUnmounted(() => {
  window.removeEventListener('auth-required', handleAuthRequired)
})

const goToLogin = () => {
  unauthMode.value = 'login'
  registerRole.value = ''
}

const goToRegister = (role: string) => {
  unauthMode.value = 'login'
  registerRole.value = role
}

const checkLoginState = () => {
  isAuthenticated.value = !!localStorage.getItem('accessToken')
  userName.value = localStorage.getItem('userName') || 'Admin'
  userRole.value = localStorage.getItem('userRole') || 'admin'
  isDataVerified.value = localStorage.getItem('isDataVerified') || 'false'
}

const handleLoginSuccess = () => {
  checkLoginState()
  currentTab.value = 'dashboard'
}

const handleVerificationSuccess = () => {
  isDataVerified.value = 'true'
}

const handleLogout = () => {
  localStorage.removeItem('accessToken')
  localStorage.removeItem('refreshToken')
  localStorage.removeItem('tenantId')
  localStorage.removeItem('userName')
  localStorage.removeItem('userRole')
  localStorage.removeItem('isDataVerified')
  isAuthenticated.value = false
  unauthMode.value = 'guest'
}

const setTab = (tab: string) => {
  currentTab.value = tab
  isSidebarOpen.value = false
}

const getTitle = () => {
  switch(currentTab.value) {
    case 'dashboard': return 'Dashboard Kampanye'
    case 'real-count': return 'Saksi & Real Count'
    case 'volunteers': return 'Manajemen Relawan'
    case 'voters': return 'Data Pemilih'
    case 'users': return 'Manajemen Pengguna'
    case 'candidates': return 'Verifikasi Calon'
    case 'tasks': return 'Tugas & Operasional'
    case 'access': return 'Akses & Log'
    case 'reports': return 'Laporan & Ekspor'
    default: return 'Campaign Manager'
  }
}
</script>

<template>
  <div v-if="!isAuthenticated">
    <GuestDashboard v-if="unauthMode === 'guest'" @login="goToLogin" @register="goToRegister" />
    <Login v-else-if="unauthMode === 'login'" :initialRole="registerRole" @login-success="handleLoginSuccess" @back="unauthMode = 'guest'" />
  </div>
  
  <div v-else class="app-container">
    <div v-if="isSidebarOpen" class="sidebar-overlay" @click="isSidebarOpen = false"></div>

    <nav class="sidebar" :class="{ 'sidebar-open': isSidebarOpen }">
      <div class="logo-area flex items-center justify-between">
        <h1>Campaign<span class="text-gradient">Manager</span></h1>
        <button class="mobile-close-btn" @click="isSidebarOpen = false" type="button">✕</button>
      </div>
      <ul class="nav-menu">
        <li @click="setTab('dashboard')" :class="{ active: currentTab === 'dashboard' }">Dashboard Utama</li>
        <li @click="setTab('real-count')" :class="{ active: currentTab === 'real-count' }">Saksi & Real Count</li>
        <li @click="setTab('users')" :class="{ active: currentTab === 'users' }">Pengguna & Jenjang</li>
        <li @click="setTab('candidates')" :class="{ active: currentTab === 'candidates' }">Verifikasi Calon</li>
        <li @click="setTab('volunteers')" :class="{ active: currentTab === 'volunteers' }">Manajemen Relawan</li>
        <li @click="setTab('voters')" :class="{ active: currentTab === 'voters' }">Voter CRM</li>
        <li @click="setTab('tasks')" :class="{ active: currentTab === 'tasks' }">Tugas & Operasional</li>
        <li @click="setTab('reports')" :class="{ active: currentTab === 'reports' }">Laporan</li>
        <li @click="setTab('access')" :class="{ active: currentTab === 'access' }">Akses & Log Audit</li>
      </ul>
      
      <div style="margin-top: auto; padding: 1rem;">
        <button @click="handleLogout" class="btn-logout" type="button" >Logout</button>
      </div>
    </nav>
    
    <main class="content">
      <div class="top-nav">
        <div class="flex items-center gap-4">
          <button class="mobile-menu-btn" @click="isSidebarOpen = true" type="button">☰</button>
          <h2>{{ getTitle() }}</h2>
        </div>
        <div class="flex items-center gap-4">
          <NotificationBell />
          <div class="user-profile">
            {{ userName }} <span class="role-badge">{{ userRole }}</span>
          </div>
        </div>
      </div>
      
      <!-- DATA VERIFICATION WALL -->
      <div class="main-body" v-if="userRole === 'kandidat' && isDataVerified === 'false'">
        <DataVerification @verified="handleVerificationSuccess" />
      </div>

      <div class="main-body" v-else>
        <CampaignDashboard v-if="currentTab === 'dashboard'" />
        <div v-else-if="currentTab === 'real-count'" class="card"><RealCount /></div>
        <div v-else-if="currentTab === 'users'" class="card"><Users /></div>
        <div v-else-if="currentTab === 'candidates'" class="card"><Candidates /></div>
        <div v-else-if="currentTab === 'volunteers'" class="card"><Volunteer /></div>
        <div v-else-if="currentTab === 'voters'" class="card"><Voter /></div>
        <div v-else-if="currentTab === 'tasks'" class="card"><TaskBoard /></div>
        <div v-else-if="currentTab === 'reports'" class="card"><ReportGenerator /></div>
        <div v-else-if="currentTab === 'access'" class="card"><AccessManager /></div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.app-container { display: flex; height: 100vh; width: 100vw; background-color: var(--bg-tertiary); position: relative; overflow: hidden; }
.sidebar { width: 280px; background-color: var(--bg-secondary); border-right: 1px solid var(--border-color); display: flex; flex-direction: column; z-index: 50; transition: transform 0.3s ease; }
.sidebar-overlay { display: none; position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0, 0, 0, 0.5); z-index: 40; }
.logo-area { padding: 1.5rem; border-bottom: 1px solid var(--border-color); }
.logo-area h1 { font-size: 1.5rem; font-weight: 800; margin: 0; color: var(--text-primary); }
.mobile-close-btn { display: none; background: none; border: none; font-size: 1.5rem; cursor: pointer; color: var(--text-primary); }
.nav-menu { list-style: none; padding: 1rem 0; margin: 0; overflow-y: auto;}
.nav-menu li { padding: 1rem 1.5rem; cursor: pointer; color: rgba(255, 255, 255, 0.85); font-weight: 500; transition: all 0.2s ease; border-left: 4px solid transparent; }
.nav-menu li:hover { background-color: var(--bg-primary); color: var(--text-primary); }
.nav-menu li.active { background-color: rgba(220, 38, 38, 0.05); color: var(--accent-primary); border-left-color: var(--accent-primary); }
.content { flex: 1; display: flex; flex-direction: column; overflow: hidden; width: 100%; }
.top-nav { height: 70px; background-color: var(--bg-secondary); border-bottom: 1px solid var(--border-color); display: flex; align-items: center; justify-content: space-between; padding: 0 2rem; flex-shrink: 0; }
.mobile-menu-btn { display: none; background: none; border: none; font-size: 1.5rem; cursor: pointer; color: var(--text-primary); }
.top-nav h2 { font-size: 1.25rem; font-weight: 600; margin: 0; }
.user-profile { background-color: var(--bg-tertiary); padding: 0.5rem 1rem; border-radius: var(--radius-sm); font-weight: 600; color: rgba(255, 255, 255, 0.95); display: flex; align-items: center; gap: 0.5rem; }
.role-badge { background: var(--accent-primary); color: white; padding: 2px 6px; border-radius: 4px; font-size: 0.7rem; text-transform: uppercase; }
.main-body { flex: 1; padding: 2rem; overflow-y: auto; -webkit-overflow-scrolling: touch; }
.btn-logout { width: 100%; padding: 0.75rem; background: #fef2f2; color: #991b1b; border: 1px solid #fecaca; border-radius: var(--radius-sm); font-weight: 600; cursor: pointer; transition: all 0.2s; }
.btn-logout:hover { background: #dc2626; color: white; }

@media (max-width: 768px) {
  .sidebar { position: fixed; top: 0; bottom: 0; left: 0; transform: translateX(-100%); }
  .sidebar-open { transform: translateX(0); }
  .sidebar-overlay { display: block; }
  .mobile-menu-btn, .mobile-close-btn { display: block; }
  .top-nav { padding: 0 1rem; }
  .top-nav h2 { font-size: 1.1rem; display: none; }
  .main-body { padding: 1rem; }
}
</style>
