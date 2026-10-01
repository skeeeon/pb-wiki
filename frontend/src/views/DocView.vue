<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, toRef, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import { useDoc } from '@/composables/useDoc'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import { useActiveHeading } from '@/composables/useActiveHeading'
import { publishPageToc } from '@/composables/usePageToc'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import { renderDoc } from '@/lib/markdown'
import type { UserRecord } from '@/lib/types'
import Breadcrumbs from '@/components/Breadcrumbs.vue'
import MarkdownView from '@/components/MarkdownView.vue'
import TocSidebar from '@/components/TocSidebar.vue'

const props = defineProps<{ path: string }>()
const path = toRef(props, 'path')
const route = useRoute()
const auth = useAuthStore()

const { doc, loading, notFound, error } = useDoc(() => path.value)

// wiki_config.default_landing_path: when the root URL has no homepage
// document, send the visitor to the landing page instead of "Not found".
const router = useRouter()
const config = useConfigStore()
watch(notFound, (missing) => {
  const landing = config.config?.default_landing_path
  if (missing && path.value === '' && landing) void router.replace(`/doc/${landing}`)
})

useDocumentTitle(() => doc.value?.title || null)

const editTo = computed(() => `/edit/${path.value}`)
const newChildTo = computed(() => `/new/${path.value ? path.value + '/' : ''}`)
const historyTo = computed(() => `/history/${path.value}`)

const rendered = computed(() => renderDoc(doc.value?.body ?? ''))
const tocHeadings = computed(() => rendered.value.headings.filter((h) => h.level >= 1 && h.level <= 4))
// A one-entry contents list is just noise, so a page needs two headings.
const showToc = computed(() => rendered.value.showToc && tocHeadings.value.length > 1)

const activeSlug = useActiveHeading(() => tocHeadings.value)

// Editor + last-modified — relies on `expand: 'updated_by'` on the fetch.
const editor = computed<UserRecord | null>(() => {
  const r = doc.value?.expand?.updated_by as UserRecord | undefined
  return r ?? null
})
const editorName = computed(() => editor.value?.name || editor.value?.email || null)
const editedRelative = computed(() => (doc.value?.updated ? relativeTime(doc.value.updated) : null))
const editedAbsolute = computed(() => (doc.value?.updated ? new Date(doc.value.updated).toLocaleString() : null))

// `Intl.RelativeTimeFormat` for "3 days ago" — keeps us off date libraries.
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

// Once the inline mobile TOC scrolls out of view, the phone top bar (App.vue)
// offers the contents instead, so long pages keep a way to jump headings.
// Desktop (lg+) uses the sticky aside.
const inlineTocEl = ref<HTMLElement | null>(null)
const inlineTocVisible = ref(true)
let observer: IntersectionObserver | null = null

watch(inlineTocEl, (el) => {
  observer?.disconnect()
  if (!el) {
    inlineTocVisible.value = true
    return
  }
  observer = new IntersectionObserver(
    ([entry]) => { inlineTocVisible.value = entry.isIntersecting },
    { threshold: 0 },
  )
  observer.observe(el)
})

onBeforeUnmount(() => observer?.disconnect())

publishPageToc(() =>
  showToc.value && doc.value
    ? {
        headings: rendered.value.headings,
        activeSlug: activeSlug.value,
        pageTitle: doc.value.title || 'Untitled',
        inlineVisible: inlineTocVisible.value,
      }
    : null,
)

// Browser hash-scroll fires before the async doc fetch lands, so the native
// scroll happens against an empty article. Re-trigger it once the rendered
// HTML is in the DOM. Watch doc + hash so hash-only changes (back/forward,
// in-page TOC clicks) still scroll, and only on real anchor hashes.
watch(
  [doc, () => route.hash],
  async ([d, hash]) => {
    if (!d || !hash || hash === '#') return
    await nextTick()
    const id = decodeURIComponent(hash.slice(1))
    document.getElementById(id)?.scrollIntoView({ block: 'start' })
  },
  { immediate: true },
)
</script>

<template>
  <div class="max-w-6xl mx-auto space-y-4">
    <Breadcrumbs :path="path" />

    <!-- Only the first load shows this. Moving between pages keeps the old
         page up until the new one arrives, so the page does not flash, and
         back/forward can restore a scroll position deeper than "Loading…". -->
    <div v-if="loading && !doc" class="text-slate-500 text-sm">Loading…</div>

    <section v-else-if="notFound" class="space-y-3">
      <h1 class="text-2xl font-semibold">Not found</h1>
      <p class="text-slate-600 dark:text-slate-400 text-sm">
        No document at <code>{{ path || '/' }}</code>.
      </p>
      <RouterLink
        v-if="auth.isEditor"
        :to="`/new/${path}`"
        class="inline-block text-sm underline"
      >
        Create this page →
      </RouterLink>
    </section>

    <section v-else-if="error" class="text-red-600 dark:text-red-400 text-sm">
      {{ String(error) }}
    </section>

    <article v-else-if="doc" class="space-y-4">
      <header class="space-y-1">
        <div class="flex items-baseline justify-between gap-4 flex-wrap">
          <h1 class="text-4xl font-semibold">{{ doc.title || 'Untitled' }}</h1>
          <nav class="flex items-center gap-2 text-sm">
            <RouterLink
              :to="historyTo"
              class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
              History
            </RouterLink>
            <template v-if="auth.isEditor">
              <RouterLink
                :to="editTo"
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                </svg>
                Edit
              </RouterLink>
              <RouterLink
                :to="newChildTo"
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-slate-300 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
                Add subpage
              </RouterLink>
            </template>
          </nav>
        </div>
        <p v-if="editedRelative" class="text-xs text-slate-500">
          Edited <time :title="editedAbsolute ?? ''">{{ editedRelative }}</time>
          <template v-if="editorName"> by {{ editorName }}</template>
        </p>
      </header>

      <!-- Mobile TOC — collapsed by default, hidden once the sidebar TOC takes
           over on lg+. Reuses TocSidebar so styling stays in one place. -->
      <details
        v-if="showToc"
        ref="inlineTocEl"
        class="lg:hidden rounded-lg border border-slate-200 dark:border-slate-800 px-3 py-2"
      >
        <summary class="text-sm font-medium cursor-pointer select-none text-slate-700 dark:text-slate-300">
          On this page
        </summary>
        <div class="pt-2">
          <TocSidebar
            :headings="rendered.headings"
            :active-slug="activeSlug"
            :page-title="doc.title"
          />
        </div>
      </details>

      <!-- Sidebar TOC on lg+. Grid keeps the article width stable whether TOC
           is shown or not — when hidden, the article gets the full column.
           The base minmax(0,1fr) column matters: without it the implicit
           column grows to the widest content (a wide table or code block),
           so on a phone the whole article becomes thousands of pixels wide
           instead of the table scrolling inside it. -->
      <div
        class="grid grid-cols-[minmax(0,1fr)] gap-8"
        :class="showToc ? 'lg:grid-cols-[minmax(0,1fr)_14rem]' : ''"
      >
        <MarkdownView :html="rendered.html" />
        <aside v-if="showToc" class="hidden lg:block">
          <div class="sticky top-6">
            <TocSidebar
              :headings="rendered.headings"
              :active-slug="activeSlug"
              :page-title="doc.title"
            />
          </div>
        </aside>
      </div>
    </article>
  </div>
</template>
