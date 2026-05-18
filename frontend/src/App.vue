<script setup lang="ts">
import { onMounted, onBeforeUnmount, computed, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import { useConfigStore } from '@/stores/config'
import { useDocsStore } from '@/stores/docs'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import { useTheme } from '@/composables/useTheme'
import Sidebar from '@/components/Sidebar.vue'

const config = useConfigStore()
const docsStore = useDocsStore()
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
    // After navigation a scroll event won't necessarily fire (e.g. landing
    // at y=0 on the new page), so re-evaluate the scrolled flag explicitly.
    scrolled.value = window.scrollY > 80
  },
)

// Escape closes the drawer; matches the usual modal/menu convention.
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && mobileOpen.value) mobileOpen.value = false
}

// Body scroll lock while the drawer is open — otherwise touch-scrolling
// over the backdrop drags the page behind it. When the drawer closes,
// return focus to the hamburger that opened it.
const hamburgerBtn = ref<HTMLButtonElement | null>(null)
watch(mobileOpen, (open, wasOpen) => {
  document.body.style.overflow = open ? 'hidden' : ''
  if (wasOpen && !open) hamburgerBtn.value?.focus()
})

// Current page title for the mobile top bar — derived from the docs store
// rather than a separate fetch. Empty for non-doc routes.
const currentDocTitle = computed(() => {
  let path: string | null = null
  if (route.name === 'home') path = ''
  else if (route.name === 'doc-view') {
    const p = route.params.path
    path = Array.isArray(p) ? p.join('/') : typeof p === 'string' ? p : ''
  }
  if (path === null) return ''
  const doc = docsStore.list.find((d) => d.path === path)
  return doc?.title || (path === '' ? 'Home' : path.split('/').pop() || '')
})

// Show the doc title in the mobile top bar only once the user has scrolled
// past the heading — keeps the bar uncluttered at the top of the page.
const scrolled = ref(false)
function onScroll() {
  scrolled.value = window.scrollY > 80
}

onMounted(() => {
  config.load()
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('scroll', onScroll)
  document.body.style.overflow = ''
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
        ref="hamburgerBtn"
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
      <!-- Title swap: site title at rest, current page title once scrolled
           past the heading. The two are stacked and cross-faded so the bar
           height stays put. -->
      <div class="relative flex-1 min-w-0 h-6">
        <span
          class="absolute inset-0 text-base font-semibold truncate transition-opacity duration-150"
          :class="scrolled && currentDocTitle ? 'opacity-0' : 'opacity-100'"
          :aria-hidden="scrolled && !!currentDocTitle"
        >
          {{ config.config?.title || 'pb-wiki' }}
        </span>
        <span
          class="absolute inset-0 text-base font-semibold truncate transition-opacity duration-150"
          :class="scrolled && currentDocTitle ? 'opacity-100' : 'opacity-0'"
          :aria-hidden="!(scrolled && currentDocTitle)"
        >
          {{ currentDocTitle }}
        </span>
      </div>
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
      <Sidebar :open="mobileOpen" @close="mobileOpen = false" />
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
