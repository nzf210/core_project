<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink, RouterView, useRouter } from 'vue-router'
import { isAuthed, getRole, logout } from './api/client'

const router = useRouter()
const role = ref(getRole())

onMounted(() => {
  if (!isAuthed()) {
    router.push('/login')
  }
})

function doLogout() {
  logout()
  router.push('/login')
}
</script>

<template>
  <div v-if="isAuthed()">
    <header class="topbar">
      <div class="brand">⚡ WCH Superadmin</div>
      <nav>
        <RouterLink to="/">Dashboard</RouterLink>
        <RouterLink to="/vouchers/programs">Voucher Programs</RouterLink>
        <RouterLink to="/vouchers/generate">Generate Links</RouterLink>
        <RouterLink to="/vouchers/analytics">Analytics</RouterLink>
        <RouterLink to="/frozen-accounts">Frozen Accounts</RouterLink>
        <RouterLink to="/plan-features">Plan Matrix</RouterLink>
      </nav>
      <div class="user-info">
        <span class="role">{{ role }}</span>
        <button @click="doLogout">Logout</button>
      </div>
    </header>
    <main class="container">
      <RouterView />
    </main>
  </div>
  <div v-else class="login-required">
    <h2>Login required</h2>
    <p>Please <a href="/login">login as superadmin</a> first.</p>
  </div>
</template>

<style scoped>
.topbar { display: flex; align-items: center; justify-content: space-between; padding: 14px 24px; background: var(--card); border-bottom: 1px solid var(--border); position: sticky; top: 0; z-index: 10; }
.brand { font-weight: 700; font-size: 18px; }
nav { display: flex; gap: 20px; }
nav a { color: var(--muted); padding: 6px 10px; border-radius: 6px; }
nav a:hover { color: var(--text); background: var(--bg); text-decoration: none; }
nav a.router-link-exact-active { color: var(--accent); background: rgba(59, 130, 246, 0.1); }
.user-info { display: flex; align-items: center; gap: 12px; }
.role { font-size: 12px; padding: 3px 8px; background: var(--accent); border-radius: 4px; text-transform: uppercase; }
button { background: var(--bg); color: var(--text); border: 1px solid var(--border); padding: 6px 12px; border-radius: 6px; }
button:hover { background: var(--border); }
.container { max-width: 1280px; margin: 0 auto; padding: 24px; }
.login-required { text-align: center; padding: 100px 20px; color: var(--muted); }
</style>
