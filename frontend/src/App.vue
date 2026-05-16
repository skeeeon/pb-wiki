<script setup lang="ts">
import { onMounted, computed, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import { useConfigStore } from '@/stores/config'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import { useTheme } from '@/composables/useTheme'
import Sidebar from '@/components/Sidebar.vue'

const config = useConfigStore()
const route = useRoute()
const { theme, toggle: toggleTheme } = useTheme()

useDocumentTitle()

// Hide the chrome on the login page so it can render full-screen.
const showChrome = computed(() => route.name !== 'login')

// Mobile drawer state. Closes automatically on navigation so tapping a doc
// link inside the drawer doesn't leave it hanging open.
const mobileOpen = ref(false)
watch(
  () => route.fullPath,
  () => {
    mobileOpen.value = false
  },
)

onMounted(() => {
  config.load()
})
</script>

<template>
  <RouterView v-if="!showChrome" />

  <div v-else class="min-h-dvh">
    <!-- Mobile top bar — hamburger + title + theme toggle. Hidden on md+.
         Height extends behind the iOS notch via safe-area-inset-top so the
         contents (sized to h-14) stay below the system status area. -->
    <header
      class="md:hidden fixed inset-x-0 top-0 z-30 h-[calc(3.5rem+env(safe-area-inset-top))] pt-[env(safe-area-inset-top)] flex items-center px-3 gap-3 border-b-2 border-brand-red bg-white dark:bg-zinc-900"
    >
      <button
        type="button"
        class="p-2.5 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800"
        aria-label="Open menu"
        @click="mobileOpen = true"
      >
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="3" y1="6" x2="21" y2="6" />
          <line x1="3" y1="12" x2="21" y2="12" />
          <line x1="3" y1="18" x2="21" y2="18" />
        </svg>
      </button>
      <span class="flex-1 min-w-0 text-base font-semibold truncate">
        {{ config.config?.title || 'pb-wiki' }}
      </span>
      <button
        type="button"
        class="shrink-0 p-2 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800"
        :title="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
        :aria-label="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
        @click="toggleTheme"
      >
        <svg v-if="theme === 'dark'" class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
        </svg>
        <svg v-else class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      </button>
    </header>

    <!-- Mobile backdrop -->
    <div
      v-show="mobileOpen"
      class="md:hidden fixed inset-0 z-40 bg-black/50"
      @click="mobileOpen = false"
    />

    <!-- Sidebar — always fixed. Off-screen on mobile, drawer when opened;
         always visible on md+. Fixed avoids the sticky-in-flex gotcha where
         a stretched flex parent prevents sticky from engaging. -->
    <aside
      class="fixed top-0 left-0 z-50 w-80 h-dvh flex flex-col
             border-r-2 border-brand-red bg-white dark:bg-zinc-900
             transition-transform duration-200 ease-out
             -translate-x-full md:translate-x-0"
      :class="{ 'translate-x-0': mobileOpen }"
    >
      <Sidebar />
    </aside>

    <!-- No overflow-x on <main>: setting it would make main a scroll
         container, which scopes descendant `position: sticky` (e.g. the
         TOC rail in DocView) to main instead of the viewport. Wide
         elements (pre, table) handle their own horizontal overflow. -->
    <main class="md:ml-80 p-6 pt-[calc(5rem+env(safe-area-inset-top))] md:pt-6">
      <RouterView />
    </main>
  </div>
</template>
