import { ref, onMounted, watch } from 'vue'

type Theme = 'dark' | 'light' | 'system'

export function useTheme() {
  const theme = ref<Theme>('system')

  const applyTheme = (t: Theme) => {
    if (t === 'system') {
      const isDark = globalThis.matchMedia('(prefers-color-scheme: dark)').matches
      document.documentElement.className = isDark ? 'theme-dark' : 'theme-light'
    } else {
      document.documentElement.className = `theme-${t}`
    }
  }

  onMounted(() => {
    const saved = localStorage.getItem('theme-preference') as Theme
    if (saved) theme.value = saved
    applyTheme(theme.value)

    globalThis.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (theme.value === 'system') applyTheme('system')
    })
  })

  watch(theme, (newTheme) => {
    localStorage.setItem('theme-preference', newTheme)
    applyTheme(newTheme)
  })

  return { theme }
}
