<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { useDocsStore } from '@/stores/docs'
import { useTheme } from '@/composables/useTheme'
import { useSearch, highlightMatch } from '@/composables/useSearch'
import {
  DropdownMenuRoot,
  DropdownMenuTrigger,
  DropdownMenuPortal,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from 'reka-ui'
import { buildTree, type TreeNode } from './sidebarTree'
import SidebarTreeItem from './SidebarTreeItem.vue'

const auth = useAuthStore()
const config = useConfigStore()
const docsStore = useDocsStore()
const { list: docs, loading, error } = storeToRefs(docsStore)
const route = useRoute()
const router = useRouter()
const { theme, toggle: toggleTheme } = useTheme()

async function signOut() {
  auth.logout()
  await router.push('/')
}

onMounted(() => docsStore.load())

const tree = computed<TreeNode[]>(() => buildTree(docs.value))
const hasHome = computed(() => docs.value.some((d) => d.path === ''))

// ----- Search -----
const q = ref('')
const { isSearching, results, bodyLoading } = useSearch(q, docs)

// ----- Tree expand/collapse -----
const expanded = ref<Set<string>>(new Set())

const currentPath = computed<string | null>(() => {
  if (route.name === 'home') return ''
  if (route.name === 'doc-view') {
    const p = route.params.path
    if (Array.isArray(p)) return p.join('/')
    if (typeof p === 'string') return p
    return ''
  }
  return null
})

watch(
  currentPath,
  (p) => {
    if (p === null) return
    const segments = p.split('/').filter(Boolean)
    let acc = ''
    for (const seg of segments) {
      acc = acc ? `${acc}/${seg}` : seg
      expanded.value.add(acc)
    }
    expanded.value = new Set(expanded.value)
  },
  { immediate: true },
)

function toggleExpand(path: string) {
  if (expanded.value.has(path)) expanded.value.delete(path)
  else expanded.value.add(path)
  expanded.value = new Set(expanded.value)
}
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Brand block — stays pinned at the top while the tree scrolls. -->
    <div class="shrink-0 p-4 space-y-3">
      <RouterLink to="/" class="block" :aria-label="config.config?.title || 'pb-wiki'">
        <img src="/logo.svg" :alt="config.config?.title || 'pb-wiki'" class="h-12 block dark:hidden" />
        <img src="/logo-dark.svg" :alt="config.config?.title || 'pb-wiki'" class="h-12 hidden dark:block" />
      </RouterLink>
      <h1 v-if="config.config?.title" class="text-base font-semibold leading-tight">
        {{ config.config.title }}
      </h1>

      <div class="relative">
        <svg
          class="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-400 pointer-events-none"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.3-4.3" />
        </svg>
        <input
          v-model="q"
          type="search"
          placeholder="Search docs…"
          class="w-full pl-8 pr-2 py-1.5 rounded border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 text-sm focus:outline-none focus:border-brand-blue"
        />
      </div>
    </div>

    <!-- Divider -->
    <div class="shrink-0 border-t border-zinc-200 dark:border-zinc-800" />

    <!-- Tree / search results — the ONLY scrollable region in the sidebar. -->
    <div class="flex-1 min-h-0 overflow-y-auto p-2">
      <!-- Search results — instant title/path matches + debounced body matches -->
      <ul v-if="isSearching" class="space-y-0.5">
        <li v-if="results.length === 0 && !bodyLoading" class="text-sm text-zinc-500 px-2 py-1">
          No matches.
        </li>
        <li v-for="r in results" :key="r.id">
          <RouterLink
            :to="r.path === '' ? '/' : `/doc/${r.path}`"
            class="block px-2 py-1.5 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800"
            :class="{
              'bg-brand-blue/10 text-brand-blue dark:text-brand-blue-dark font-medium':
                (r.path === '' && route.path === '/') || route.path === `/doc/${r.path}`,
            }"
          >
            <div class="text-sm truncate">{{ r.title || r.path || 'Home' }}</div>
            <div v-if="r.path" class="text-xs text-zinc-500 truncate font-mono">{{ r.path }}</div>
            <div
              v-if="r.snippet"
              class="text-xs text-zinc-600 dark:text-zinc-400 mt-0.5 line-clamp-2"
              v-html="highlightMatch(r.snippet, q)"
            />
          </RouterLink>
        </li>
        <li v-if="bodyLoading" class="text-xs text-zinc-500 px-2 py-1.5 italic">
          Searching content…
        </li>
      </ul>

      <!-- Tree -->
      <template v-else>
        <div class="flex items-center justify-between mb-1 px-2">
          <h2 class="text-xs uppercase tracking-wide text-zinc-500">Pages</h2>
          <RouterLink
            v-if="auth.isEditor"
            to="/new/"
            class="text-xs text-zinc-500 hover:underline"
            title="Create a new top-level page"
          >
            + New
          </RouterLink>
        </div>

        <RouterLink
          v-if="hasHome"
          to="/"
          class="flex items-center py-1 px-2 rounded text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
          :class="{
            'bg-brand-blue/10 text-brand-blue dark:text-brand-blue-dark font-medium': route.path === '/',
          }"
        >
          Home
        </RouterLink>

        <ul v-if="!loading && tree.length > 0" class="space-y-0.5 mt-1">
          <SidebarTreeItem
            v-for="node in tree"
            :key="node.fullPath"
            :node="node"
            :depth="0"
            :expanded="expanded"
            :on-toggle="toggleExpand"
          />
        </ul>
        <p v-else-if="loading" class="text-sm text-zinc-500 px-2 py-1">Loading…</p>
        <p v-else-if="error" class="text-sm text-red-600 dark:text-red-400 px-2 py-1">
          {{ error instanceof Error ? error.message : String(error) }}
        </p>
        <p v-else class="text-sm text-zinc-500 px-2 py-1">No pages yet.</p>
      </template>
    </div>

    <!-- User menu — pinned to bottom, includes the theme toggle. -->
    <div class="shrink-0 border-t border-zinc-200 dark:border-zinc-800 p-3">
      <template v-if="auth.isAuthenticated">
        <div class="flex items-center gap-2">
          <DropdownMenuRoot>
            <DropdownMenuTrigger
              as="button"
              type="button"
              class="flex items-center gap-2 flex-1 min-w-0 p-1 -m-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-blue"
              :aria-label="`Account menu for ${auth.record?.email}`"
            >
              <div class="w-8 h-8 rounded-full bg-brand-blue text-white text-sm font-medium flex items-center justify-center shrink-0">
                {{ (auth.record?.email ?? '?').charAt(0).toUpperCase() }}
              </div>
              <div class="min-w-0 flex-1 text-left">
                <div class="text-sm truncate">{{ auth.record?.email }}</div>
                <div class="text-xs text-zinc-500">{{ auth.role }}</div>
              </div>
              <svg class="w-3.5 h-3.5 text-zinc-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="m6 9 6 6 6-6" />
              </svg>
            </DropdownMenuTrigger>
            <DropdownMenuPortal>
              <DropdownMenuContent
                side="top"
                align="start"
                :side-offset="6"
                class="min-w-[12rem] rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-1 shadow-md text-sm focus:outline-none z-[60]"
              >
                <DropdownMenuItem
                  v-if="auth.isAdmin"
                  as-child
                >
                  <RouterLink
                    to="/admin"
                    class="flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer outline-none data-[highlighted]:bg-zinc-100 dark:data-[highlighted]:bg-zinc-800"
                  >
                    <svg class="w-4 h-4 text-zinc-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="3" />
                      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
                    </svg>
                    Admin
                  </RouterLink>
                </DropdownMenuItem>
                <DropdownMenuSeparator
                  v-if="auth.isAdmin"
                  class="h-px my-1 bg-zinc-200 dark:bg-zinc-800"
                />
                <DropdownMenuItem
                  class="flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer outline-none data-[highlighted]:bg-zinc-100 dark:data-[highlighted]:bg-zinc-800 text-red-600 dark:text-red-400"
                  @select="signOut"
                >
                  <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                    <polyline points="16 17 21 12 16 7" />
                    <line x1="21" y1="12" x2="9" y2="12" />
                  </svg>
                  Sign out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenuPortal>
          </DropdownMenuRoot>
          <button
            type="button"
            class="shrink-0 p-1.5 rounded border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            :title="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
            :aria-label="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
            @click="toggleTheme"
          >
            <svg v-if="theme === 'dark'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
            </svg>
            <svg v-else class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          </button>
        </div>
      </template>
      <template v-else>
        <div class="flex items-center gap-2">
          <RouterLink
            to="/login"
            class="flex-1 text-center rounded-md bg-brand-red hover:bg-brand-red-hover text-white text-sm font-medium px-3 py-1.5"
          >
            Sign in
          </RouterLink>
          <button
            type="button"
            class="shrink-0 p-1.5 rounded border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            :title="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
            :aria-label="theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'"
            @click="toggleTheme"
          >
            <svg v-if="theme === 'dark'" class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
            </svg>
            <svg v-else class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          </button>
        </div>
      </template>
    </div>
  </div>
</template>
