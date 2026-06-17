<template>
  <aside :class="['app-sidebar', { 'is-open': isOpen, 'is-collapsed': isCollapsed, 'frozen-offset': isFrozen }]">
    <!-- Overlay for mobile -->
    <div class="sidebar-overlay" @click="$emit('close')"></div>

    <!-- Sidebar Content -->
    <div class="sidebar-content">
      <!-- Header -->
      <div class="sidebar-header">
        <div class="logo-area">
          <h1 class="logo text-gradient">WCH UMKM</h1>
        </div>
        <button class="collapse-btn desktop-only" @click="toggleCollapse" :title="isCollapsed ? 'Expand' : 'Collapse'">
          <span>{{ isCollapsed ? '→' : '←' }}</span>
        </button>
        <button class="close-btn mobile-only" @click="$emit('close')">✕</button>
      </div>

      <!-- Navigation -->
      <nav class="sidebar-nav">
        <div v-for="group in filteredGroups" :key="group.group" class="nav-group">
          <button
            class="group-header"
            @click="toggleGroup(group.group)"
            :title="isCollapsed ? group.group : ''"
          >
            <span class="group-icon">{{ getGroupIcon(group.group) }}</span>
            <span v-if="!isCollapsed" class="group-label">{{ group.group }}</span>
            <span v-if="!isCollapsed" class="group-chevron" :class="{ rotated: !collapsedGroups.has(group.group) }">
              ▼
            </span>
          </button>

          <div
            v-show="!collapsedGroups.has(group.group)"
            class="group-items"
          >
            <router-link
              v-for="item in group.items"
              :key="item.to"
              :to="item.to"
              class="nav-item"
              active-class="active"
              :exact="item.to === '/'"
              @click="$emit('close')"
 :title="isCollapsed ? item.label : ''"
            >
              <span class="item-icon">{{ item.icon }}</span>
              <span v-if="!isCollapsed" class="item-label">{{ item.label }}</span>
            </router-link>
          </div>
        </div>
      </nav>

      <!-- User Profile -->
      <div class="sidebar-footer">
        <div class="user-profile">
          <div class="avatar">{{ (businessName || 'U')[0].toUpperCase() }}</div>
          <div v-if="!isCollapsed" class="user-info">
            <span class="business-name">{{ businessName || 'My UMKM' }}</span>
            <span v-if="plan !== 'lite' && plan !== 'inactive'" :class="['plan-chip', `plan-${plan}`]">
              {{ plan.toUpperCase() }}
            </span>
            <span v-else-if="plan === 'inactive'" class="plan-chip plan-inactive">INACTIVE</span>
            <span v-else class="plan-chip plan-lite">LITE</span>
          </div>
        </div>
        <button @click="logout" class="logout-btn" :title="isCollapsed ? 'Keluar' : ''">
          <span>🚪</span>
          <span v-if="!isCollapsed">Keluar</span>
        </button>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { menuConfig } from '../config/menu'

const props = defineProps<{
  isOpen: boolean
  userRole: string
  businessName: string
  plan: string
  businessType?: string
  isFrozen?: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const router = useRouter()
const isCollapsed = ref(false)
const collapsedGroups = ref<Set<string>>(new Set())

const filteredGroups = computed(() => {
  return menuConfig
    .map(group => ({
      ...group,
      items: group.items.filter(item => {
        // Role check
        if (item.roles && !item.roles.includes(props.userRole)) {
          return false
        }
        // Business type check (F047): kalau menu punya businessTypes filter,
        // tampil hanya jika tenant businessType termasuk di list
        if (item.businessTypes && item.businessTypes.length > 0) {
          if (!props.businessType || !item.businessTypes.includes(props.businessType)) {
            return false
          }
        }
        return true
      }),
    }))
    .filter(group => group.items.length > 0)
})

const getGroupIcon = (group: string) => {
  const icons: Record<string, string> = {
    'Operasi': '🏢',
    'Keuangan': '💵',
    'Sistem': '⚙️',
    'Admin': '🔐',
  }
  return icons[group] || '📁'
}

const toggleGroup = (group: string) => {
  if (collapsedGroups.value.has(group)) {
    collapsedGroups.value.delete(group)
  } else {
    collapsedGroups.value.add(group)
  }
}

const toggleCollapse = () => {
  isCollapsed.value = !isCollapsed.value
}

const logout = () => {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('tenant_id')
  localStorage.removeItem('role')
  emit('close')
  router.push('/login')
}
</script>

<style scoped>
.app-sidebar {
  --sidebar-width: 260px;
  --sidebar-collapsed-width: 72px;
  --sidebar-bg: var(--surface-0, #ffffff);
  --sidebar-border: var(--border-color, #e5e7eb);
  --text-primary: var(--text-primary, #1e293b);
  --text-secondary: var(--text-secondary, #64748b);
  --accent-primary: var(--accent-primary, #3b82f6);

  position: fixed;
  top: 0;
  left: 0;
  height: 100vh;
  width: var(--sidebar-width);
  z-index: 100;
  transition: width0.3s ease;
}

.app-sidebar.is-collapsed {
  width: var(--sidebar-collapsed-width);
}

.app-sidebar.frozen-offset {
  top: 48px;
  height: calc(100vh - 48px);
}

.sidebar-overlay {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: -1;
}

.sidebar-content {
  height: 100%;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--sidebar-border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem;
  border-bottom: 1px solid var(--sidebar-border);
  min-height: 64px;
}

.logo-area {
  flex: 1;
  overflow: hidden;
}

.logo {
  font-size: 1.25rem;
  margin: 0;
  font-weight: 700;
  white-space: nowrap;
}

.collapse-btn {
  background: transparent;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: var(--text-secondary);
  border-radius: 4px;
  transition: all 0.2s;
}

.collapse-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: var(--text-primary);
}

.close-btn {
  display: none;
  background: transparent;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  font-size: 1.25rem;
  color: var(--text-secondary);
}

.sidebar-nav {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0.75rem 0;
}

.nav-group {
  margin-bottom: 0.5rem;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.5rem 1rem;
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  transition: all 0.2s;
}

.group-header:hover {
  color: var(--text-primary);
}

.group-icon {
  font-size: 1rem;
  width: 24px;
  text-align: center;
}

.group-label {
  flex: 1;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-chevron {
  font-size: 0.6rem;
  transition: transform 0.2s;
}

.group-chevron.rotated {
  transform: rotate(-90deg);
}

.group-items {
  padding: 0.25rem 0;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 1rem 0.625rem 2.5rem;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.9rem;
  font-weight: 500;
  transition: all 0.2s;
  border-left: 3px solid transparent;
  white-space: nowrap;
  overflow: hidden;
}

.nav-item:hover {
  color: var(--text-primary);
  background: rgba(0, 0, 0, 0.03);
}

.nav-item.active {
  color: var(--accent-primary);
  background: rgba(59, 130, 246, 0.08);
  border-left-color: var(--accent-primary);
}

.item-icon {
  font-size: 1.1rem;
  width: 24px;
  text-align: center;
  flex-shrink: 0;
}

.item-label {
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar-footer {
  padding: 1rem;
  border-top: 1px solid var(--sidebar-border);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(to bottom right, var(--accent-primary), #1d4ed8);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  color: white;
  font-size: 0.9rem;
  flex-shrink: 0;
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  overflow: hidden;
}

.business-name {
  font-weight: 500;
  font-size: 0.9rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.plan-chip {
  display: inline-block;
  font-size: 0.6rem;
  padding: 0.1rem 0.4rem;
  border-radius: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  width: fit-content;
}

.plan-lite { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
.plan-pro { background: rgba(59, 130, 246, 0.15); color: #60a5fa; }
.plan-ultimate { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
.plan-inactive { background: rgba(239, 68, 68, 0.15); color: #ef4444; }

.logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid #ef4444;
  color: #ef4444;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.85rem;
  font-weight: 500;
  transition: all 0.2s;
  white-space: nowrap;
  overflow: hidden;
}

.logout-btn:hover {
  background: rgba(239, 68, 68, 0.1);
}

/* Mobile */
@media (max-width: 768px) {
  .app-sidebar {
    width: 100%;
    transform: translateX(-100%);
    transition: transform 0.3s ease;
  }

  .app-sidebar.is-open {
    transform: translateX(0);
  }

  .app-sidebar.is-open .sidebar-overlay {
    display: block;
  }

  .close-btn {
    display: block;
  }

  .collapse-btn {
    display: none;
  }
}

/* Desktop only */
.desktop-only {
  display: block;
}

.mobile-only {
  display: none;
}

@media (max-width: 768px) {
  .desktop-only {
    display: none !important;
  }

  .mobile-only {
    display: block;
  }
}
</style>
