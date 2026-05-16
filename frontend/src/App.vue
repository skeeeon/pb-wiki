<script setup lang="ts">
import { onMounted, computed, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import { useConfigStore } from '@/stores/config'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import { online } from '@/composables/useOnline'
import Sidebar from '@/components/Sidebar.vue'

const config = useConfigStore()
const route = useRoute()

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
    <!-- Mobile top bar — hamburger + title. Hidden on md+. -->
    <header
      class="md:hidden fixed inset-x-0 top-0 z-30 h-12 flex items-center px-3 gap-3 border-b-2 border-brand-red bg-white dark:bg-zinc-900"
    >
      <button
        type="button"
        class="p-1.5 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800"
        aria-label="Open menu"
        @click="mobileOpen = true"
      >
        <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="3" y1="6" x2="21" y2="6" />
          <line x1="3" y1="12" x2="21" y2="12" />
          <line x1="3" y1="18" x2="21" y2="18" />
        </svg>
      </button>
      <span class="text-sm font-semibold truncate">
        {{ config.config?.title || 'pb-wiki' }}
      </span>
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

    <!-- Offline indicator. Fixed at the top of the main pane (right of the
         sidebar on md+) so it doesn't shift layout when toggling. -->
    <div
      v-if="!online"
      class="fixed top-12 md:top-0 left-0 md:left-80 right-0 z-30
             bg-brand-yellow/90 text-zinc-900 text-xs font-medium
             px-3 py-1.5 flex items-center gap-2 border-b border-amber-600/30"
      role="status"
    >
      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M1 1l22 22" />
        <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55" />
        <path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39" />
        <path d="M10.71 5.05A16 16 0 0 1 22.58 9" />
        <path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88" />
        <path d="M8.53 16.11a6 6 0 0 1 6.95 0" />
        <line x1="12" y1="20" x2="12.01" y2="20" />
      </svg>
      Offline — showing cached pages.
    </div>

    <!-- No overflow-x on <main>: setting it would make main a scroll
         container, which scopes descendant `position: sticky` (e.g. the
         TOC rail in DocView) to main instead of the viewport. Wide
         elements (pre, table) handle their own horizontal overflow. -->
    <main
      class="md:ml-80 p-6 pt-16 md:pt-6"
      :class="{ 'pt-24 md:pt-12': !online }"
    >
      <RouterView />
    </main>
  </div>
</template>
