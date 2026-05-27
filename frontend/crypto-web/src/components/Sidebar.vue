<template>
  <aside :class="[
    'sidebar', 
    { 'collapsed': isCollapsed },
    { 'mobile-open': isMobileOpen }
  ]">
    <div class="sidebar-header">
      <div class="logo-container" v-if="!isCollapsed">
        <span class="logo-icon">📈</span>
        <h2 class="logo-text">WCH Trade</h2>
      </div>
      <div class="logo-container-collapsed" v-else>
        <span class="logo-icon">📈</span>
      </div>
      <button class="collapse-btn desktop-only" @click="$emit('toggleCollapse')">
        <span v-if="!isCollapsed">◁</span>
        <span v-else>▷</span>
      </button>
      <button class="close-mobile-btn mobile-only" @click="$emit('closeMobile')">
        ✕
      </button>
    </div>

    <nav class="sidebar-nav">
      <button 
        v-for="item in navItems" 
        :key="item.id"
        :class="['nav-item', { active: currentView === item.id }]"
        @click="handleNavigate(item.id)"
        :title="isCollapsed ? item.label : ''"
      >
        <span class="nav-icon">{{ item.icon }}</span>
        <span class="nav-label" v-if="!isCollapsed">{{ item.label }}</span>
      </button>
    </nav>
    
    <div class="sidebar-footer" v-if="!isCollapsed">
      <div class="user-info">
        <div class="avatar">U</div>
        <div class="details">
          <div class="name">User Trader</div>
          <div class="status text-green">Pro Plan</div>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
const props = defineProps<{
  currentView: string;
  isCollapsed: boolean;
  isMobileOpen: boolean;
}>()

const emit = defineEmits(['navigate', 'toggleCollapse', 'closeMobile'])

const handleNavigate = (id: string) => {
  emit('navigate', id)
  emit('closeMobile') // Auto-close on mobile after navigating
}

const navItems = [
  { id: 'dashboard', label: 'Dashboard', icon: '📊' },
  { id: 'bots', label: 'My Bots', icon: '🤖' },
  { id: 'performance', label: 'Performance', icon: '📈' },
  { id: 'wallets', label: 'Wallets', icon: '💼' },
  { id: 'apikeys', label: 'API Keys', icon: '🔑' },
  { id: 'settings', label: 'Settings', icon: '⚙️' }
]
</script>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  height: 100vh;
  background-color: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  position: fixed;
  left: 0;
  top: 0;
  z-index: 100;
}

.sidebar.collapsed {
  width: var(--sidebar-collapsed-width);
}

.sidebar-header {
  height: var(--topbar-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1rem;
  border-bottom: 1px solid var(--border-color);
}

.logo-container, .logo-container-collapsed {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.logo-icon {
  font-size: 1.5rem;
}

.logo-text {
  font-size: 1.2rem;
  font-weight: 700;
  margin: 0;
  background: linear-gradient(90deg, var(--accent-primary), #a855f7);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.collapse-btn, .close-mobile-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.collapse-btn:hover, .close-mobile-btn:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.mobile-only {
  display: none;
}

.sidebar-nav {
  flex: 1;
  padding: 1rem 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  overflow-y: auto;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius-md);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
  width: 100%;
}

.sidebar.collapsed .nav-item {
  justify-content: center;
  padding: 0.75rem 0;
}

.nav-item:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.active {
  background-color: rgba(59, 130, 246, 0.1);
  color: var(--accent-primary);
  font-weight: 500;
}

.nav-icon {
  font-size: 1.25rem;
}

.sidebar-footer {
  padding: 1rem;
  border-top: 1px solid var(--border-color);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background-color: var(--accent-primary);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
}

.details .name {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
}

.details .status {
  font-size: 0.75rem;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
    width: 280px;
    box-shadow: 4px 0 24px rgba(0, 0, 0, 0.5);
  }
  
  .sidebar.mobile-open {
    transform: translateX(0);
  }
  
  /* Reset collapse state on mobile (always expanded view) */
  .sidebar.collapsed {
    width: 280px;
  }
  
  .desktop-only {
    display: none;
  }
  
  .mobile-only {
    display: flex;
  }
}
</style>
