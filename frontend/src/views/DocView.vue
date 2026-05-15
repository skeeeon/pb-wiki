<script setup lang="ts">
import { computed, toRef } from 'vue'
import { RouterLink } from 'vue-router'

import { useDoc } from '@/composables/useDoc'
import { useDocumentTitle } from '@/composables/useDocumentTitle'
import { useAuthStore } from '@/stores/auth'
import { renderDoc } from '@/lib/markdown'
import Breadcrumbs from '@/components/Breadcrumbs.vue'
import MarkdownView from '@/components/MarkdownView.vue'
import TocSidebar from '@/components/TocSidebar.vue'

const props = defineProps<{ path: string }>()
const path = toRef(props, 'path')
const auth = useAuthStore()

const { doc, loading, notFound, error } = useDoc(() => path.value)

useDocumentTitle(() => doc.value?.title || null)

const editTo = computed(() => `/edit/${path.value}`)
const newChildTo = computed(() => `/new/${path.value ? path.value + '/' : ''}`)

const rendered = computed(() => renderDoc(doc.value?.body ?? ''))
const showToc = computed(() => rendered.value.showToc && rendered.value.headings.some((h) => h.level >= 1 && h.level <= 4))
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
      <header class="flex items-baseline justify-between gap-4 flex-wrap">
        <h1 class="text-2xl font-semibold">{{ doc.title || 'Untitled' }}</h1>
        <nav v-if="auth.isEditor" class="flex items-center gap-3 text-sm">
          <RouterLink :to="editTo" class="underline">Edit</RouterLink>
          <RouterLink :to="newChildTo" class="underline">New child</RouterLink>
        </nav>
      </header>

      <!-- TOC rail shows on lg+ when the doc has eligible headings and hasn't
           opted out. Grid keeps the article width stable whether TOC is shown
           or not — when hidden, the article gets a single full-width column. -->
      <div
        class="grid gap-8"
        :class="showToc ? 'lg:grid-cols-[minmax(0,1fr)_14rem]' : ''"
      >
        <MarkdownView :html="rendered.html" />
        <aside v-if="showToc" class="hidden lg:block">
          <div class="sticky top-6">
            <TocSidebar :headings="rendered.headings" />
          </div>
        </aside>
      </div>
    </article>
  </div>
</template>
