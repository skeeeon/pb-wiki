<script setup lang="ts">
import { onMounted, computed, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import { useConfigStore } from '@/stores/config'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
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
      class="md:hidden fixed inset-x-0 top-0 z-30 h-14 flex items-center px-3 gap-3 border-b-2 border-brand-red bg-white dark:bg-zinc-900"
    >
      <button
        type="button"
        class="p-2.5 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800"
        aria-label="Open menu"
        @click="mobileOpen = true"
      >
        <svg class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="3" y1="6" x2="21" y2="6" />
          <line x1="3" y1="12" x2="21" y2="12" />
          <line x1="3" y1="18" x2="21" y2="18" />
        </svg>
      </button>
      <span class="text-base font-semibold truncate">
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

    <!-- No overflow-x on <main>: setting it would make main a scroll
         container, which scopes descendant `position: sticky` (e.g. the
         TOC rail in DocView) to main instead of the viewport. Wide
         elements (pre, table) handle their own horizontal overflow. -->
    <main class="md:ml-80 p-6 pt-20 md:pt-6">
      <RouterView />
    </main>
  </div>
</template>
