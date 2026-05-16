<script setup lang="ts">
import { computed, ref } from 'vue'

import { pb } from '@/lib/pb'
import { useDocsStore } from '@/stores/docs'
import type { DocumentRecord } from '@/lib/types'
import AdminNav from '@/components/admin/AdminNav.vue'

const docsStore = useDocsStore()

const from = ref('')
const to = ref('')

interface PreviewRow {
  id: string
  from: string
  to: string
}

const preview = ref<PreviewRow[]>([])
const previewLoading = ref(false)
const previewed = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

const applying = ref(false)

// Normalize the same way the backend does so the preview matches what the
// commit will actually act on. Lifted out of the handler so the input
// can show the canonical form to the admin before they hit "Preview".
function normalize(p: string): string {
  return p.trim().replace(/^\/+|\/+$/g, '')
}

function rewrite(oldPath: string, fromN: string, toN: string): string {
  if (oldPath === fromN) return toN
  const suffix = oldPath.slice(fromN.length + 1)
  return toN ? `${toN}/${suffix}` : suffix
}

const fromNormalized = computed(() => normalize(from.value))
const toNormalized = computed(() => normalize(to.value))

const canPreview = computed(
  () => fromNormalized.value !== '' && fromNormalized.value !== toNormalized.value,
)

async function runPreview() {
  if (!canPreview.value || previewLoading.value) return
  errorMsg.value = ''
  successMsg.value = ''
  previewLoading.value = true
  preview.value = []
  previewed.value = false
  try {
    const f = fromNormalized.value
    const t = toNormalized.value
    // Match the doc at `from` itself plus anything under `from/`. PB's `~`
    // operator auto-wraps with %% unless an explicit % is present, so we
    // pass the trailing % to opt into "starts with".
    const matches = await pb.collection('documents').getFullList<DocumentRecord>({
      filter: pb.filter('path = {:from} || path ~ {:prefix}', {
        from: f,
        prefix: `${f}/%`,
      }),
      sort: '+path',
      fields: 'id,path,title',
    })
    preview.value = matches.map((d) => ({
      id: d.id,
      from: d.path,
      to: rewrite(d.path, f, t),
    }))
    previewed.value = true
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    previewLoading.value = false
  }
}

async function apply() {
  if (preview.value.length === 0 || applying.value) return
  // The server is authoritative on collisions and validation; we just POST
  // the prefixes and trust the response. A success refreshes the sidebar
  // doc list so the new tree shows up everywhere immediately.
  if (
    !confirm(
      `Move ${preview.value.length} document(s) from "${fromNormalized.value}" to "${
        toNormalized.value || '(homepage)'
      }"?`,
    )
  )
    return
  applying.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    const res = await pb.send<{ moved: number }>('/api/wiki/bulk-move', {
      method: 'POST',
      body: { from: fromNormalized.value, to: toNormalized.value },
    })
    successMsg.value = `Moved ${res.moved} document${res.moved === 1 ? '' : 's'}.`
    preview.value = []
    previewed.value = false
    from.value = ''
    to.value = ''
    await docsStore.reload()
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    applying.value = false
  }
}
</script>

<template>
  <div class="max-w-3xl mx-auto space-y-6">
    <AdminNav />

    <header>
      <h1 class="text-xl font-semibold">Move pages</h1>
      <p class="text-sm text-zinc-500">
        Rewrite the path prefix of an entire subtree in one transaction. Preview the affected
        documents before committing — the change is applied atomically and can't be undone.
      </p>
    </header>

    <form
      class="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm overflow-hidden"
      @submit.prevent="runPreview"
    >
      <section class="px-6 pt-5 pb-5 space-y-4">
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300 font-medium">From prefix</span>
            <input
              v-model="from"
              required
              placeholder="engineering/docs"
              class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm font-mono focus:outline-none focus:border-brand-blue"
            />
            <span class="mt-1.5 block text-xs text-zinc-500">
              The doc at this path and every descendant will be moved.
            </span>
          </label>
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300 font-medium">To prefix</span>
            <input
              v-model="to"
              placeholder="eng"
              class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm font-mono focus:outline-none focus:border-brand-blue"
            />
            <span class="mt-1.5 block text-xs text-zinc-500">
              Leave empty to land the subtree at the wiki root (homepage).
            </span>
          </label>
        </div>
      </section>

      <div
        class="flex items-center justify-between gap-3 px-6 py-4 bg-zinc-50 dark:bg-zinc-950/40 border-t border-zinc-200 dark:border-zinc-800"
      >
        <p class="text-sm min-h-[1.25rem]">
          <span v-if="successMsg" class="text-green-600 dark:text-green-400">{{ successMsg }}</span>
          <span v-else-if="errorMsg" class="text-red-600 dark:text-red-400">{{ errorMsg }}</span>
          <span
            v-else-if="previewed && preview.length === 0"
            class="text-zinc-500"
          >No matches.</span>
        </p>
        <button
          type="submit"
          :disabled="!canPreview || previewLoading"
          class="rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 px-4 py-1.5 text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ previewLoading ? 'Loading…' : 'Preview' }}
        </button>
      </div>
    </form>

    <section
      v-if="preview.length > 0"
      class="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm overflow-hidden"
    >
      <header
        class="flex items-baseline justify-between px-6 py-3 bg-zinc-50 dark:bg-zinc-950/40 border-b border-zinc-200 dark:border-zinc-800"
      >
        <h2 class="text-sm font-semibold">Preview</h2>
        <span class="text-xs text-zinc-500">{{ preview.length }} affected</span>
      </header>
      <table class="w-full text-sm">
        <thead class="text-zinc-500 dark:text-zinc-400 text-xs uppercase tracking-wide">
          <tr class="border-b border-zinc-200 dark:border-zinc-800">
            <th class="text-left px-6 py-2 font-medium">From</th>
            <th class="text-left px-6 py-2 font-medium">To</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-200 dark:divide-zinc-800">
          <tr v-for="row in preview" :key="row.id">
            <td class="px-6 py-2 font-mono text-xs text-zinc-700 dark:text-zinc-300">
              {{ row.from || '(homepage)' }}
            </td>
            <td class="px-6 py-2 font-mono text-xs">
              {{ row.to || '(homepage)' }}
            </td>
          </tr>
        </tbody>
      </table>

      <div
        class="flex items-center justify-end gap-3 px-6 py-4 bg-zinc-50 dark:bg-zinc-950/40 border-t border-zinc-200 dark:border-zinc-800"
      >
        <button
          type="button"
          :disabled="applying"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-4 py-1.5 text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          @click="apply"
        >
          {{ applying ? 'Applying…' : `Apply move (${preview.length})` }}
        </button>
      </div>
    </section>
  </div>
</template>
