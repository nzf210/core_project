<template>
  <div v-if="error" class="error-banner" role="alert">
    <div class="error-content">
      <span class="error-icon">⚠️</span>
      <div class="error-message">
        <strong>{{ title }}</strong>
        <p>{{ error }}</p>
      </div>
      <button v-if="onRetry" @click="handleRetry" class="btn-secondary btn-sm">
        {{ retrying ? 'Retrying...' : 'Retry' }}
      </button>
      <button @click="$emit('dismiss')" class="btn-text btn-sm" aria-label="Dismiss">
        ✕
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  error: string | null
  title?: string
  onRetry?: () => void | Promise<void>
}>()

defineEmits<{
  dismiss: []
}>()

const retrying = ref(false)

async function handleRetry() {
  if (!props.onRetry) return
  retrying.value = true
  try {
    await props.onRetry()
  } finally {
    retrying.value = false
  }
}
</script>

<style scoped>
.error-banner {
  background: #fee;
  border: 1px solid #fcc;
  border-radius: 4px;
  padding: 12px;
  margin-bottom: 16px;
}

.error-content {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.error-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.error-message {
  flex: 1;
  min-width: 0;
}

.error-message strong {
  display: block;
  color: #c33;
  margin-bottom: 4px;
}

.error-message p {
  margin: 0;
  color: #666;
  font-size: 0.9rem;
}

.btn-sm {
  padding: 4px 12px;
  font-size: 0.85rem;
  white-space: nowrap;
}

.btn-text {
  background: none;
  border: none;
  color: #999;
  cursor: pointer;
  padding: 4px 8px;
  font-size: 1.2rem;
  line-height: 1;
}

.btn-text:hover {
  color: #666;
}
</style>
