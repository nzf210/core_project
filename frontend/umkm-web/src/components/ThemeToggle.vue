<template>
  <button @click="cycleTheme" class="theme-toggle" :title="`Theme: ${theme}`">
    <span v-if="theme === 'dark'">🌙</span>
    <span v-else-if="theme === 'light'">☀️</span>
    <span v-else>💻</span>
  </button>
</template>

<script setup lang="ts">
import { useTheme } from '../composables/useTheme'

const { theme } = useTheme()

const cycleTheme = () => {
  if (theme.value === 'system') theme.value = 'dark'
  else if (theme.value === 'dark') theme.value = 'light'
  else theme.value = 'system'
}
</script>

<style scoped>
.theme-toggle {
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 1.25rem;
  padding: 0.5rem;
  border-radius: var(--radius-md);
  transition: background 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.theme-toggle:hover {
  background: var(--surface-2);
}
</style>
