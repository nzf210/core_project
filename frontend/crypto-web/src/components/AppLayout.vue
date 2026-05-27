<template>
  <div class="app-wrapper">
    <!-- Mobile Sidebar Backdrop Overlay -->
    <div 
      v-if="isMobileSidebarOpen" 
      class="sidebar-backdrop" 
      @click="isMobileSidebarOpen = false"
    ></div>

    <Sidebar 
      :currentView="currentView"
      :isCollapsed="isSidebarCollapsed"
      :isMobileOpen="isMobileSidebarOpen"
      @navigate="handleNavigate"
      @toggleCollapse="isSidebarCollapsed = !isSidebarCollapsed"
      @closeMobile="isMobileSidebarOpen = false"
    />
    
    <div :class="['main-content-wrapper', { 'sidebar-collapsed': isSidebarCollapsed }]">
      <Topbar 
        @quickTrade="handleQuickTrade" 
        @toggleMobileMenu="isMobileSidebarOpen = !isMobileSidebarOpen" 
        @logout="emit('logout')"
      />
      
      <main class="content-area">
        <slot></slot>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { toast } from 'vue3-toastify'
import Sidebar from './Sidebar.vue'
import Topbar from './Topbar.vue'

defineProps<{
  currentView: string;
}>()

const emit = defineEmits(['update:currentView', 'logout'])

const isSidebarCollapsed = ref(false)
const isMobileSidebarOpen = ref(false)

const handleNavigate = (view: string) => {
  emit('update:currentView', view)
}

const handleQuickTrade = () => {
  toast.info('Fitur Quick Trade segera hadir!')
}
</script>

<style scoped>
.app-wrapper {
  display: flex;
  min-height: 100vh;
  width: 100%;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  position: relative;
}

.sidebar-backdrop {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(2px);
  z-index: 95;
}

.main-content-wrapper {
  flex: 1;
  margin-left: var(--sidebar-width);
  transition: margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  width: calc(100% - var(--sidebar-width));
}

.main-content-wrapper.sidebar-collapsed {
  margin-left: var(--sidebar-collapsed-width);
  width: calc(100% - var(--sidebar-collapsed-width));
}

.content-area {
  flex: 1;
  padding: 1.5rem;
  overflow-y: auto;
  overflow-x: hidden;
}

/* Mobile Responsiveness */
@media (max-width: 768px) {
  .main-content-wrapper,
  .main-content-wrapper.sidebar-collapsed {
    margin-left: 0;
    width: 100%;
  }
  
  .sidebar-backdrop {
    display: block;
  }
  
  .content-area {
    padding: 1rem;
  }
}
</style>
