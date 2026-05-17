<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { RouterLink } from 'vue-router'

import { useDocHistory, type Revision } from '@/composables/useDocHistory'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import Breadcrumbs from '@/components/Breadcrumbs.vue'

const props = defineProps<{ path: string }>()
const path = toRef(props, 'path')

const { revisions, loading, notFound, error } = useDocHistory(() => path.value)

useDocumentTitle(() => `History — ${path.value || 'home'}`)

const docTo = computed(() => (path.value ? `/doc/${path.value}` : '/'))

const expanded = ref<string | null>(null)
function toggle(id: string) {
  expanded.value = expanded.value === id ? null : id
}

// Pull the body string out of a before/after snapshot. The snapshot mirrors
// the documents row, but we treat any field absence as empty so missing
// `before` (on a `create` event) doesn't break the diff.
function bodyOf(snapshot: Revision['before' | 'after']): string {
  if (!snapshot) return ''
  const v = (snapshot as Record<string, unknown>).body
  return typeof v === 'string' ? v : ''
}

function titleOf(snapshot: Revision['before' | 'after']): string {
  if (!snapshot) return ''
  const v = (snapshot as Record<string, unknown>).title
  return typeof v === 'string' ? v : ''
}

// Local-date format mirroring the convention in DocView.
function formatTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

function relativeTime(iso: string): string {
  const diffSec = Math.round((new Date(iso).getTime() - Date.now()) / 1000)
  const abs = Math.abs(diffSec)
  const fmt = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })
  if (abs < 60)       return fmt.format(diffSec, 'second')
  if (abs < 3600)     return fmt.format(Math.round(diffSec / 60), 'minute')
  if (abs < 86400)    return fmt.format(Math.round(diffSec / 3600), 'hour')
  if (abs < 2592000)  return fmt.format(Math.round(diffSec / 86400), 'day')
  if (abs < 31536000) return fmt.format(Math.round(diffSec / 2592000), 'month')
  return fmt.format(Math.round(diffSec / 31536000), 'year')
}

function editorOf(r: Revision): string {
  if (r.user) return r.user.name || r.user.email
  // pb-audit stores null user for admin/superuser edits since admins aren't
  // in the users collection. Make that visible rather than blank.
  return 'admin'
}

// Hand-rolled LCS line diff. Wiki bodies are short enough that O(n*m) is
// fine; this avoids pulling in the `diff` package for ~30 lines of code.
type DiffLine = { kind: 'eq' | 'add' | 'del'; text: string }
function lineDiff(before: string, after: string): DiffLine[] {
  const a = before.split('\n')
  const b = after.split('\n')
  const n = a.length
  const m = b.length

  // LCS table
  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      if (a[i] === b[j]) lcs[i][j] = lcs[i + 1][j + 1] + 1
      else lcs[i][j] = Math.max(lcs[i + 1][j], lcs[i][j + 1])
    }
  }

  const out: DiffLine[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ kind: 'eq', text: a[i] })
      i++; j++
    } else if (lcs[i + 1][j] >= lcs[i][j + 1]) {
      out.push({ kind: 'del', text: a[i] })
      i++
    } else {
      out.push({ kind: 'add', text: b[j] })
      j++
    }
  }
  while (i < n) { out.push({ kind: 'del', text: a[i++] }); }
  while (j < m) { out.push({ kind: 'add', text: b[j++] }); }
  return out
}

function diffFor(r: Revision): DiffLine[] {
  return lineDiff(bodyOf(r.before), bodyOf(r.after))
}

function diffStats(r: Revision): { added: number; removed: number } {
  const d = diffFor(r)
  let added = 0
  let removed = 0
  for (const l of d) {
    if (l.kind === 'add') added++
    else if (l.kind === 'del') removed++
  }
  return { added, removed }
}
</script>

<template>
  <div class="max-w-5xl mx-auto space-y-4">
    <Breadcrumbs :path="path" />

    <header class="flex items-baseline justify-between gap-4 flex-wrap">
      <h1 class="text-3xl font-semibold">History</h1>
      <RouterLink
        :to="docTo"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm"
      >
        ← Back to page
      </RouterLink>
    </header>

    <p class="text-xs text-zinc-500">
      <code>{{ path || '/' }}</code>
    </p>

    <div v-if="loading" class="text-zinc-500 text-sm">Loading…</div>

    <section v-else-if="notFound" class="space-y-2">
      <h2 class="text-xl font-medium">No history available</h2>
      <p class="text-sm text-zinc-600 dark:text-zinc-400">
        Either this page doesn't exist or you don't have access to it.
      </p>
    </section>

    <section v-else-if="error" class="text-red-600 dark:text-red-400 text-sm">
      {{ String(error) }}
    </section>

    <section v-else-if="revisions.length === 0" class="text-sm text-zinc-600 dark:text-zinc-400">
      No revisions captured yet — pb-audit only records changes made after it
      was enabled, so older pages may not have history until they're next
      edited.
    </section>

    <ul v-else class="divide-y divide-zinc-200 dark:divide-zinc-800 border border-zinc-200 dark:border-zinc-800 rounded">
      <li v-for="r in revisions" :key="r.id" class="px-3 py-2">
        <button
          type="button"
          class="flex items-center justify-between w-full text-left gap-3"
          @click="toggle(r.id)"
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2 flex-wrap">
              <span
                class="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded"
                :class="
                  r.event_type === 'create'
                    ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
                    : 'bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300'
                "
              >
                {{ r.event_type }}
              </span>
              <time class="text-sm text-zinc-700 dark:text-zinc-300" :title="formatTime(r.timestamp)">
                {{ relativeTime(r.timestamp) }}
              </time>
              <span class="text-xs text-zinc-500">by {{ editorOf(r) }}</span>
              <span
                v-if="titleOf(r.after) && titleOf(r.before) && titleOf(r.before) !== titleOf(r.after)"
                class="text-xs text-amber-700 dark:text-amber-300"
              >
                title: {{ titleOf(r.before) }} → {{ titleOf(r.after) }}
              </span>
            </div>
            <div class="text-xs text-zinc-500 mt-0.5">
              +{{ diffStats(r).added }} / −{{ diffStats(r).removed }}
            </div>
          </div>
          <svg
            class="w-4 h-4 text-zinc-400 shrink-0 transition-transform"
            :class="expanded === r.id ? 'rotate-90' : ''"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="9 18 15 12 9 6" />
          </svg>
        </button>

        <pre
          v-if="expanded === r.id"
          class="mt-2 text-xs leading-snug font-mono whitespace-pre-wrap overflow-x-auto bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded p-2"
        ><template v-for="(line, idx) in diffFor(r)" :key="idx"><span
              v-if="line.kind === 'add'"
              class="block bg-emerald-50 dark:bg-emerald-950/50 text-emerald-900 dark:text-emerald-200"
            >+ {{ line.text }}</span><span
              v-else-if="line.kind === 'del'"
              class="block bg-red-50 dark:bg-red-950/50 text-red-900 dark:text-red-200"
            >− {{ line.text }}</span><span
              v-else
              class="block text-zinc-600 dark:text-zinc-400"
            >  {{ line.text }}</span></template></pre>
      </li>
    </ul>
  </div>
</template>
