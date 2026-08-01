import { ref, onMounted, watch } from 'vue'
import { sanitizeTheme } from '../api'

type Theme = 'dark' | 'light' | 'system'

const ALLOWED_THEMES: readonly Theme[] = ['dark', 'light', 'system'] as const

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
    const saved = localStorage.getItem('theme-preference')
    const sanitized = sanitizeTheme(saved) as Theme
    if (ALLOWED_THEMES.includes(sanitized)) theme.value = sanitized
    applyTheme(theme.value)

    globalThis.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (theme.value === 'system') applyTheme('system')
    })
  })

  watch(theme, (newTheme) => {
    localStorage.setItem('theme-preference', sanitizeTheme(newTheme))
    applyTheme(newTheme)
  })

  return { theme }
}
