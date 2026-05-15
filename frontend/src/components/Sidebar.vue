<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import { pb } from '@/lib/pb'
import { useAuthStore } from '@/stores/auth'
import type { DocumentRecord } from '@/lib/types'
import { buildTree, type TreeNode } from './sidebarTree'
import SidebarTreeItem from './SidebarTreeItem.vue'

const auth = useAuthStore()
const route = useRoute()

const docs = ref<DocumentRecord[]>([])
const loading = ref(true)
const error = ref<string>('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    // Hooks (internal/hooks/documents.go) filter this list to only paths the
    // current user has access to read; intermediate tree nodes are fabricated
    // client-side so the user can still navigate down.
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
</script>

<template>
  <div class="text-sm">
    <div class="flex items-center justify-between mb-2 px-2">
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
      class="block py-1 px-2 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800"
      :class="{ 'bg-zinc-100 dark:bg-zinc-800 font-medium': route.path === '/' }"
    >
      Home
    </RouterLink>

    <ul v-if="!loading && tree.length > 0" class="space-y-0.5 mt-1">
      <SidebarTreeItem v-for="node in tree" :key="node.fullPath" :node="node" :depth="0" />
    </ul>

    <p v-else-if="loading" class="text-zinc-500 px-2 py-1">Loading…</p>
    <p v-else-if="error" class="text-red-600 dark:text-red-400 px-2 py-1">{{ error }}</p>
    <p v-else class="text-zinc-500 px-2 py-1">No pages yet.</p>
  </div>
</template>
