<script setup lang="ts">
import { computed, ref, toRef, watch } from 'vue'
import { RouterLink } from 'vue-router'

import { useDoc } from '@/composables/useDoc'
import { useDocHistory, type Revision } from '@/composables/useDocHistory'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import { renderDoc } from '@/lib/markdown'
import Breadcrumbs from '@/components/Breadcrumbs.vue'
import MarkdownView from '@/components/MarkdownView.vue'

const props = defineProps<{ path: string }>()
const path = toRef(props, 'path')

const { revisions, loading, notFound, error } = useDocHistory(() => path.value)
const { doc: currentDoc } = useDoc(() => path.value)
const currentBody = computed(() => currentDoc.value?.body ?? '')

useDocumentTitle(() => `History — ${path.value || 'home'}`)

const docTo = computed(() => (path.value ? `/doc/${path.value}` : '/'))

// View mode for the right preview pane.
type ViewMode = 'rendered' | 'diff-edit' | 'diff-current' | 'raw'
const mode = ref<ViewMode>('diff-edit')

// Which revision is currently selected. Defaults to the newest one so the
// preview shows the most recent edit on load. Reactively snaps to the first
// revision whenever the list refreshes.
const selectedId = ref<string | null>(null)
watch(
  revisions,
  (list) => {
    if (list.length === 0) {
      selectedId.value = null
    } else if (!list.some((r) => r.id === selectedId.value)) {
      selectedId.value = list[0].id
    }
  },
  { immediate: true },
)
const selected = computed<Revision | null>(
  () => revisions.value.find((r) => r.id === selectedId.value) ?? null,
)

// Pull `body`/`title` out of a before/after snapshot. Snapshots mirror the
// documents row but we don't trust the shape — older revisions may have used
// different fields if the schema has evolved.
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

// `diff-edit` compares this revision against its own before — i.e. what
// changed at the moment of this edit. `diff-current` compares it against the
// live doc body — i.e. what's drifted since.
function diffEdit(r: Revision): DiffLine[] {
  return lineDiff(bodyOf(r.before), bodyOf(r.after))
}
function diffCurrent(r: Revision): DiffLine[] {
  return lineDiff(bodyOf(r.after), currentBody.value)
}
function statsOf(lines: DiffLine[]): { added: number; removed: number } {
  let added = 0, removed = 0
  for (const l of lines) {
    if (l.kind === 'add') added++
    else if (l.kind === 'del') removed++
  }
  return { added, removed }
}
// Stats shown in the left-rail row — always "what this edit changed", since
// that's the signal most useful for picking which revision to look at.
function editStats(r: Revision) {
  return statsOf(diffEdit(r))
}

const renderedHtml = computed(() =>
  selected.value ? renderDoc(bodyOf(selected.value.after)).html : '',
)
const previewDiff = computed<DiffLine[]>(() => {
  if (!selected.value) return []
  if (mode.value === 'diff-edit') return diffEdit(selected.value)
  if (mode.value === 'diff-current') return diffCurrent(selected.value)
  return []
})
const previewRaw = computed(() => (selected.value ? bodyOf(selected.value.after) : ''))

// On small screens, focusing a revision should scroll the preview into view —
// without it, tapping a row gives no visible feedback. Triggered on mode +
// selection changes since both can fire from row clicks.
const previewEl = ref<HTMLElement | null>(null)
watch([selectedId, mode], () => {
  if (typeof window === 'undefined') return
  if (window.matchMedia('(min-width: 1024px)').matches) return
  previewEl.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
})
</script>

<template>
  <div class="max-w-6xl mx-auto space-y-4">
    <Breadcrumbs :path="path" />

    <header class="flex items-baseline justify-between gap-4 flex-wrap">
      <div>
        <h1 class="text-3xl font-semibold">History</h1>
        <p class="text-xs text-zinc-500 mt-1">
          <code>{{ path || '/' }}</code>
        </p>
      </div>
      <RouterLink
        :to="docTo"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-sm"
      >
        ← Back to page
      </RouterLink>
    </header>

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

    <!-- Two-pane layout: revision list on the left, preview on the right.
         Stacks vertically below lg so mobile users see the list, tap a
         revision, and scroll naturally into the preview. -->
    <div
      v-else
      class="grid gap-4 lg:grid-cols-[18rem_minmax(0,1fr)] items-start"
    >
      <!-- Left rail: revision list. Sticky on lg+ so it doesn't scroll out of
           view while the preview pane scrolls. -->
      <aside class="lg:sticky lg:top-4">
        <ul
          class="divide-y divide-zinc-200 dark:divide-zinc-800 border border-zinc-200 dark:border-zinc-800 rounded max-h-[calc(100vh-8rem)] overflow-y-auto"
        >
          <li v-for="r in revisions" :key="r.id">
            <button
              type="button"
              class="w-full text-left px-3 py-2 flex flex-col gap-1 transition-colors"
              :class="
                selectedId === r.id
                  ? 'bg-brand-blue/10 dark:bg-brand-blue-dark/15 border-l-2 border-brand-blue dark:border-brand-blue-dark'
                  : 'border-l-2 border-transparent hover:bg-zinc-100 dark:hover:bg-zinc-800'
              "
              @click="selectedId = r.id"
            >
              <div class="flex items-center gap-2 flex-wrap">
                <span
                  class="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded shrink-0"
                  :class="
                    r.event_type === 'create'
                      ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
                      : 'bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300'
                  "
                >
                  {{ r.event_type }}
                </span>
                <time
                  class="text-sm text-zinc-700 dark:text-zinc-300 truncate"
                  :title="formatTime(r.timestamp)"
                >
                  {{ relativeTime(r.timestamp) }}
                </time>
              </div>
              <div class="flex items-center justify-between text-xs text-zinc-500 gap-2">
                <span class="truncate">{{ editorOf(r) }}</span>
                <span class="shrink-0 font-mono">
                  <span class="text-emerald-700 dark:text-emerald-400">+{{ editStats(r).added }}</span>
                  <span class="mx-0.5">/</span>
                  <span class="text-red-700 dark:text-red-400">−{{ editStats(r).removed }}</span>
                </span>
              </div>
              <span
                v-if="titleOf(r.after) && titleOf(r.before) && titleOf(r.before) !== titleOf(r.after)"
                class="text-[11px] text-amber-700 dark:text-amber-300 truncate"
              >
                title: {{ titleOf(r.before) }} → {{ titleOf(r.after) }}
              </span>
            </button>
          </li>
        </ul>
      </aside>

      <!-- Right pane: preview with mode toggle. -->
      <div ref="previewEl" class="space-y-3 min-w-0">
        <div
          class="inline-flex rounded border border-zinc-300 dark:border-zinc-700 overflow-hidden text-xs"
          role="tablist"
        >
          <button
            type="button"
            role="tab"
            :aria-selected="mode === 'rendered'"
            class="px-2.5 py-1.5 transition-colors"
            :class="
              mode === 'rendered'
                ? 'bg-zinc-200 dark:bg-zinc-700 font-medium'
                : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'
            "
            @click="mode = 'rendered'"
          >
            Rendered
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="mode === 'diff-edit'"
            class="px-2.5 py-1.5 border-l border-zinc-300 dark:border-zinc-700 transition-colors"
            :class="
              mode === 'diff-edit'
                ? 'bg-zinc-200 dark:bg-zinc-700 font-medium'
                : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'
            "
            @click="mode = 'diff-edit'"
          >
            Diff this edit
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="mode === 'diff-current'"
            class="px-2.5 py-1.5 border-l border-zinc-300 dark:border-zinc-700 transition-colors"
            :class="
              mode === 'diff-current'
                ? 'bg-zinc-200 dark:bg-zinc-700 font-medium'
                : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'
            "
            @click="mode = 'diff-current'"
          >
            Diff vs current
          </button>
          <button
            type="button"
            role="tab"
            :aria-selected="mode === 'raw'"
            class="px-2.5 py-1.5 border-l border-zinc-300 dark:border-zinc-700 transition-colors"
            :class="
              mode === 'raw'
                ? 'bg-zinc-200 dark:bg-zinc-700 font-medium'
                : 'hover:bg-zinc-100 dark:hover:bg-zinc-800'
            "
            @click="mode = 'raw'"
          >
            Raw
          </button>
        </div>

        <div v-if="selected" class="space-y-2">
          <div class="text-xs text-zinc-500">
            <time :title="formatTime(selected.timestamp)">{{ relativeTime(selected.timestamp) }}</time>
            · by {{ editorOf(selected) }}
            · <code>{{ selected.event_type }}</code>
          </div>

          <div
            v-if="mode === 'rendered'"
            class="rounded border border-zinc-200 dark:border-zinc-800 p-4 bg-white dark:bg-zinc-900"
          >
            <MarkdownView :html="renderedHtml" />
          </div>

          <pre
            v-else-if="mode === 'raw'"
            class="text-xs leading-snug font-mono whitespace-pre-wrap overflow-x-auto bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded p-3 text-zinc-700 dark:text-zinc-300"
          >{{ previewRaw }}</pre>

          <template v-else>
            <p
              v-if="previewDiff.every((l) => l.kind === 'eq')"
              class="text-xs italic text-zinc-500 px-1"
            >
              <template v-if="mode === 'diff-current'">
                This revision matches the current version — no drift since.
              </template>
              <template v-else>
                No body changes in this edit (title or other metadata may have changed).
              </template>
            </p>
            <pre
              v-else
              class="text-xs leading-snug font-mono whitespace-pre-wrap overflow-x-auto bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded p-3"
            ><template v-for="(line, idx) in previewDiff" :key="idx"><span
                  v-if="line.kind === 'add'"
                  class="block bg-emerald-50 dark:bg-emerald-950/50 text-emerald-900 dark:text-emerald-200"
                >+ {{ line.text }}</span><span
                  v-else-if="line.kind === 'del'"
                  class="block bg-red-50 dark:bg-red-950/50 text-red-900 dark:text-red-200"
                >− {{ line.text }}</span><span
                  v-else
                  class="block text-zinc-600 dark:text-zinc-400"
                >  {{ line.text }}</span></template></pre>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
