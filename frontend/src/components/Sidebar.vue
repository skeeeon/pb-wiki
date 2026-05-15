<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { pb } from '@/lib/pb'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { useTheme } from '@/composables/useTheme'
import { useSearch, highlightMatch } from '@/composables/useSearch'
import type { DocumentRecord } from '@/lib/types'
import { buildTree, type TreeNode } from './sidebarTree'
import SidebarTreeItem from './SidebarTreeItem.vue'

const auth = useAuthStore()
const config = useConfigStore()
const route = useRoute()
const { theme, toggle: toggleTheme } = useTheme()

const docs = ref<DocumentRecord[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    docs.value = await pb.collection('documents').getFullList<DocumentRecord>({
      sort: '+path',
      fields: 'id,path,title',
    })
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}
onMounted(load)

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
        <p v-else-if="error" class="text-sm text-red-600 dark:text-red-400 px-2 py-1">{{ error }}</p>
        <p v-else class="text-sm text-zinc-500 px-2 py-1">No pages yet.</p>
      </template>
    </div>

    <!-- User menu — pinned to bottom, includes the theme toggle. -->
    <div class="shrink-0 border-t border-zinc-200 dark:border-zinc-800 p-3">
      <template v-if="auth.isAuthenticated">
        <div class="flex items-center gap-2 mb-2">
          <div class="w-8 h-8 rounded-full bg-brand-blue text-white text-sm font-medium flex items-center justify-center shrink-0">
            {{ (auth.record?.email ?? '?').charAt(0).toUpperCase() }}
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-sm truncate">{{ auth.record?.email }}</div>
            <div class="text-xs text-zinc-500">{{ auth.role }}</div>
          </div>
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
        <div class="flex items-center gap-3 text-xs">
          <RouterLink v-if="auth.isAdmin" to="/admin" class="underline">Admin</RouterLink>
          <button type="button" class="underline" @click="auth.logout">Sign out</button>
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
