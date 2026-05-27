<template>
  <div class="n8n-container">
    <div class="n8n-header">
      <h2>N8n Workflow Automations</h2>
      <button class="btn btn-secondary" @click="$router.push('/superadmin')">Kembali ke Dashboard</button>
    </div>
    <div class="iframe-wrapper">
      <iframe :src="n8nUrl" class="n8n-iframe" title="n8n Workflow Builder"></iframe>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { API_BASE } from '../api'

onMounted(() => {
  const token = localStorage.getItem('access_token') || ''
  // Set cookie for iframe asset requests, with domain scope if possible
  document.cookie = `access_token=${token}; path=/; max-age=3600; SameSite=Lax`
})

// Compute the full URL for the n8n proxy endpoint in api-gateway
const n8nUrl = computed(() => {
  const token = localStorage.getItem('access_token') || ''
  return `${API_BASE}/api/superadmin/n8n/?token=${token}`
})
</script>

<style scoped>
.n8n-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 120px);
  width: 100%;
}

.n8n-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.n8n-header h2 {
  margin: 0;
}

.iframe-wrapper {
  flex: 1;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid var(--border-color, #e5e7eb);
  background: white;
}

.n8n-iframe {
  width: 100%;
  height: 100%;
  border: none;
}
</style>
