<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { RouterLink, RouterView, useRouter, useRoute } from 'vue-router'
import { isAuthed, getRole, logout } from './api/client'

const router = useRouter()
const route = useRoute()
const authed = ref(isAuthed())
const role = ref(getRole())

watch(
  () => route.path,
  () => {
    authed.value = isAuthed()
    role.value = getRole()
  }
)

onMounted(() => {
  if (!authed.value) {
    router.push('/login')
  }
})

function doLogout() {
  logout()
  authed.value = false
  router.push('/login')
}
</script>

<template>
  <div v-if="authed">
    <header class="topbar">
      <div class="brand">⚡ WCH Superadmin</div>
      <nav>
        <RouterLink to="/">Dashboard</RouterLink>
        <RouterLink to="/tenants">Tenant</RouterLink>
        <RouterLink to="/vouchers/programs">Voucher Programs</RouterLink>
        <RouterLink to="/vouchers/generate">Generate Links</RouterLink>
        <RouterLink to="/vouchers/analytics">Analytics</RouterLink>
        <RouterLink to="/frozen-accounts">Frozen Accounts</RouterLink>
        <RouterLink to="/plan-features">Plan Limits</RouterLink>
        <RouterLink to="/feature-matrix">Feature Matrix</RouterLink>
        <RouterLink to="/addon-pricing">Addon Pricing</RouterLink>
        <RouterLink to="/referral-config">Referral</RouterLink>
        <RouterLink to="/campaign-licenses">Licenses</RouterLink>
        <RouterLink to="/">📱 WA Center</RouterLink>
        <RouterLink to="/landing-editor">🌐 Landing Editor</RouterLink>
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
  <RouterView v-else />
</template>

<style scoped>
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 24px;
  background: var(--card);
  border-bottom: 1px solid var(--border);
  position: sticky;
  top: 0;
  z-index: 10;
  gap: 20px;
}
.brand { font-weight: 700; font-size: 16px; white-space: nowrap; flex-shrink: 0; }
nav {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
  flex: 1;
  justify-content: center;
}
nav a {
  color: var(--muted);
  padding: 5px 9px;
  border-radius: 6px;
  font-size: 13px;
  white-space: nowrap;
}
nav a:hover { color: var(--text); background: var(--bg); text-decoration: none; }
nav a.router-link-exact-active { color: var(--accent); background: rgba(59, 130, 246, 0.1); }
.user-info { display: flex; align-items: center; gap: 10px; flex-shrink: 0; }
.role {
  font-size: 11px;
  padding: 3px 8px;
  background: var(--accent);
  border-radius: 4px;
  text-transform: uppercase;
  font-weight: 700;
  color: white;
}
button {
  background: var(--bg);
  color: var(--text);
  border: 1px solid var(--border);
  padding: 5px 12px;
  border-radius: 6px;
  font-size: 13px;
}
button:hover { background: var(--border); }
.container { max-width: 1400px; margin: 0 auto; padding: 28px 24px; }
</style>
