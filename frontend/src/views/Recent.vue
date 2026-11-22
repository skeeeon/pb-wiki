<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

import { pb } from '@/lib/pb'
import { relativeTime } from '@/lib/time'
import type { DocumentRecord } from '@/lib/types'
import { useDocumentTitle } from '@/composables/useDocumentTitle'

useDocumentTitle(() => 'Recent changes')

// The latest save of each page, newest first. The documents ListRule decides
// what the reader may see, so this needs no endpoint of its own. Deleted
// pages and older edits to the same page don't show; the page's history
// has those.
const docs = ref<DocumentRecord[]>([])
const loading = ref(true)
const error = ref<unknown>(null)

onMounted(async () => {
  try {
    const res = await pb.collection('documents').getList<DocumentRecord>(1, 50, {
      sort: '-updated',
      fields: 'id,path,title,created,updated,expand.updated_by.name,expand.updated_by.email',
      expand: 'updated_by',
    })
    docs.value = res.items
  } catch (err) {
    error.value = err
  } finally {
    loading.value = false
  }
})

function editorOf(d: DocumentRecord): string | null {
  const u = d.expand?.updated_by
  return u ? u.name || u.email : null
}
</script>

<template>
  <div class="max-w-3xl mx-auto space-y-4">
    <h1 class="text-4xl font-semibold">Recent changes</h1>

    <p v-if="loading" class="text-slate-500 text-sm">Loading…</p>
    <p v-else-if="error" class="text-red-600 dark:text-red-400 text-sm">
      {{ error instanceof Error ? error.message : String(error) }}
    </p>
    <p v-else-if="docs.length === 0" class="text-slate-500 text-sm">No pages yet.</p>

    <ul v-else class="divide-y divide-slate-200 dark:divide-slate-800">
      <li v-for="d in docs" :key="d.id">
        <RouterLink
          :to="d.path === '' ? '/' : `/doc/${d.path}`"
          class="flex items-baseline justify-between gap-4 py-2.5 px-2 -mx-2 rounded hover:bg-slate-100 dark:hover:bg-slate-800"
        >
          <div class="min-w-0">
            <div class="truncate">
              {{ d.title || d.path || 'Home' }}
              <span
                v-if="d.created === d.updated"
                class="ml-1.5 text-xs px-1.5 py-0.5 rounded bg-primary/10 text-primary align-middle"
              >new</span>
            </div>
            <div v-if="d.path" class="text-xs text-slate-500 truncate font-mono">{{ d.path }}</div>
          </div>
          <div class="shrink-0 text-xs text-slate-500 text-right">
            <time :title="new Date(d.updated).toLocaleString()">{{ relativeTime(d.updated) }}</time>
            <div v-if="editorOf(d)" class="truncate max-w-40">{{ editorOf(d) }}</div>
          </div>
        </RouterLink>
      </li>
    </ul>
  </div>
</template>
