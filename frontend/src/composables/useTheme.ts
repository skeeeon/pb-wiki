import { ref, watch } from 'vue'

export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'pb-wiki:theme'

// Module-level state — a single source of truth across the app. The first
// component to call useTheme initializes it; later callers re-use.
const theme = ref<Theme>(detectInitial())

watch(theme, apply, { immediate: true })

function detectInitial(): Theme {
  if (typeof window === 'undefined') return 'light'
  const stored = localStorage.getItem(STORAGE_KEY) as Theme | null
  if (stored === 'light' || stored === 'dark') return stored
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function apply(next: Theme) {
  if (typeof document === 'undefined') return
  document.documentElement.classList.toggle('dark', next === 'dark')
  localStorage.setItem(STORAGE_KEY, next)
}

export function useTheme() {
  return {
    theme,
    toggle() {
      theme.value = theme.value === 'dark' ? 'light' : 'dark'
    },
    set(next: Theme) {
      theme.value = next
    },
  }
}
