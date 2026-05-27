<template>
  <div v-if="!isAuthenticated" class="auth-container">
    <Auth :onAuthSuccess="handleAuth" />
  </div>
  
  <AppLayout v-else v-model:currentView="currentView" @logout="logout">
    <transition name="fade" mode="out-in">
      <div :key="currentView">
        <CryptoDashboard v-if="currentView === 'dashboard'" />
        <BotManagement v-if="currentView === 'bots'" />
        <Performance v-if="currentView === 'performance'" />
        <ApiKeys v-if="currentView === 'apikeys'" />
        
        <!-- Placeholders for other views -->
        <div v-if="!['dashboard', 'bots', 'performance', 'apikeys'].includes(currentView)" class="placeholder-view">
          <div class="card">
            <h2>{{ currentView.charAt(0).toUpperCase() + currentView.slice(1) }} View</h2>
            <p class="text-muted">This view is under construction.</p>
          </div>
        </div>
      </div>
    </transition>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import AppLayout from './components/AppLayout.vue'
import CryptoDashboard from './components/CryptoDashboard.vue'
import BotManagement from './components/BotManagement.vue'
import Performance from './components/Performance.vue'
import Auth from './components/Auth.vue'
import ApiKeys from './components/ApiKeys.vue'

// Check if token exists in localStorage
const isAuthenticated = ref(!!localStorage.getItem('token')) 
const currentView = ref('bots') // Default to bots view

const handleAuth = (token: string) => {
  localStorage.setItem('token', token)
  isAuthenticated.value = true
}

// Global logout function
const logout = () => {
  localStorage.removeItem('token')
  isAuthenticated.value = false
}
</script>

<style scoped>
.auth-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--bg-primary);
}

.placeholder-view {
  display: flex;
  justify-content: center;
  padding: 2rem;
}

.placeholder-view .card {
  text-align: center;
  max-width: 400px;
  width: 100%;
}
</style>
