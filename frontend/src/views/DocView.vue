<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, toRef, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  PopoverRoot,
  PopoverTrigger,
  PopoverPortal,
  PopoverContent,
} from 'reka-ui'

import { useDoc } from '@/composables/useDoc'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import { useActiveHeading } from '@/composables/useActiveHeading'
import { useAuthStore } from '@/stores/auth'
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

useDocumentTitle(() => doc.value?.title || null)

const editTo = computed(() => `/edit/${path.value}`)
const newChildTo = computed(() => `/new/${path.value ? path.value + '/' : ''}`)

const rendered = computed(() => renderDoc(doc.value?.body ?? ''))
const tocHeadings = computed(() => rendered.value.headings.filter((h) => h.level >= 1 && h.level <= 4))
const showToc = computed(() => rendered.value.showToc && tocHeadings.value.length > 0)

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

// Floating TOC button — visible only when the inline mobile TOC has scrolled
// out of viewport, so users keep an affordance to jump headings on long pages.
// Desktop (lg+) uses the sticky aside and doesn't need this.
const inlineTocEl = ref<HTMLElement | null>(null)
const inlineTocVisible = ref(true)
const fabOpen = ref(false)
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

function onPopoverClick(ev: MouseEvent) {
  const target = ev.target as HTMLElement | null
  if (target?.closest('a')) fabOpen.value = false
}

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
  <div class="max-w-5xl mx-auto space-y-4">
    <Breadcrumbs :path="path" />

    <div v-if="loading" class="text-zinc-500 text-sm">Loading…</div>

    <section v-else-if="notFound" class="space-y-3">
      <h1 class="text-2xl font-semibold">Not found</h1>
      <p class="text-zinc-600 dark:text-zinc-400 text-sm">
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
          <nav v-if="auth.isEditor" class="flex items-center gap-2 text-sm">
            <RouterLink
              :to="editTo"
              class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
              </svg>
              Edit
            </RouterLink>
            <RouterLink
              :to="newChildTo"
              class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-zinc-300 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19" />
                <line x1="5" y1="12" x2="19" y2="12" />
              </svg>
              Add subpage
            </RouterLink>
          </nav>
        </div>
        <p v-if="editedRelative" class="text-xs text-zinc-500">
          Edited <time :title="editedAbsolute ?? ''">{{ editedRelative }}</time>
          <template v-if="editorName"> by {{ editorName }}</template>
        </p>
      </header>

      <!-- Mobile TOC — collapsed by default, hidden once the sidebar TOC takes
           over on lg+. Reuses TocSidebar so styling stays in one place. -->
      <details
        v-if="showToc"
        ref="inlineTocEl"
        class="lg:hidden rounded-md border border-zinc-200 dark:border-zinc-800 px-3 py-2"
      >
        <summary class="text-sm font-medium cursor-pointer select-none text-zinc-700 dark:text-zinc-300">
          On this page
        </summary>
        <div class="pt-2">
          <TocSidebar :headings="rendered.headings" :active-slug="activeSlug" />
        </div>
      </details>

      <!-- Sidebar TOC on lg+. Grid keeps the article width stable whether TOC
           is shown or not — when hidden, the article gets the full column. -->
      <div
        class="grid gap-8"
        :class="showToc ? 'lg:grid-cols-[minmax(0,1fr)_14rem]' : ''"
      >
        <MarkdownView :html="rendered.html" />
        <aside v-if="showToc" class="hidden lg:block">
          <div class="sticky top-6">
            <TocSidebar :headings="rendered.headings" :active-slug="activeSlug" />
          </div>
        </aside>
      </div>

      <!-- Floating TOC button — appears below lg once the inline TOC has
           scrolled out of viewport, keeping the heading list reachable on
           long pages without bringing the full sidebar back. -->
      <PopoverRoot v-if="showToc && !inlineTocVisible" v-model:open="fabOpen">
        <PopoverTrigger
          as="button"
          type="button"
          aria-label="On this page"
          class="lg:hidden fixed bottom-[calc(1rem+env(safe-area-inset-bottom))] right-4 z-40 w-12 h-12 rounded-full bg-brand-blue text-white shadow-lg flex items-center justify-center hover:bg-brand-blue-hover focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-blue"
        >
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="8" y1="6" x2="21" y2="6" />
            <line x1="8" y1="12" x2="21" y2="12" />
            <line x1="8" y1="18" x2="21" y2="18" />
            <line x1="3" y1="6" x2="3.01" y2="6" />
            <line x1="3" y1="12" x2="3.01" y2="12" />
            <line x1="3" y1="18" x2="3.01" y2="18" />
          </svg>
        </PopoverTrigger>
        <PopoverPortal>
          <PopoverContent
            side="top"
            align="end"
            :side-offset="8"
            :collision-padding="16"
            class="lg:hidden z-50 w-[min(20rem,calc(100vw-2rem))] max-h-[60vh] overflow-y-auto rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3 shadow-lg focus:outline-none"
            @click="onPopoverClick"
          >
            <TocSidebar :headings="rendered.headings" :active-slug="activeSlug" />
          </PopoverContent>
        </PopoverPortal>
      </PopoverRoot>
    </article>
  </div>
</template>
