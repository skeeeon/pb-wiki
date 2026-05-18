<script setup lang="ts">
import { computed } from 'vue'
import type { Heading } from '@/lib/markdown'

const props = defineProps<{
  headings: Heading[]
  activeSlug?: string | null
  pageTitle?: string | null
}>()

// h1-h4 land in the TOC. h5/h6 are excluded — they produce too dense a tree
// to be useful as nav.
const items = computed(() => props.headings.filter((h) => h.level >= 1 && h.level <= 4))

// Re-base the level so the smallest visible heading sits at indent 0.
const minLevel = computed(() => items.value.reduce((m, h) => Math.min(m, h.level), 6))

// Visible whenever we have either a title or at least one heading — the title
// alone is still useful as a sticky "you are here" indicator on long pages.
const visible = computed(() => items.value.length > 0 || !!props.pageTitle)

function scrollTo(slug: string, ev: MouseEvent) {
  const el = document.getElementById(slug)
  if (!el) return
  ev.preventDefault()
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  history.replaceState(null, '', `#${slug}`)
}

function scrollToTop(ev: MouseEvent) {
  ev.preventDefault()
  window.scrollTo({ top: 0, behavior: 'smooth' })
  // Strip the hash so refreshing doesn't re-jump to the last heading.
  history.replaceState(null, '', window.location.pathname + window.location.search)
}
</script>

<template>
  <nav v-if="visible" aria-label="On this page" class="text-sm">
    <a
      v-if="pageTitle"
      href="#"
      class="block mb-2 font-semibold text-zinc-900 dark:text-zinc-100 hover:text-brand-blue dark:hover:text-brand-blue-dark truncate"
      title="Scroll to top"
      @click="scrollToTop"
    >
      {{ pageTitle }}
    </a>
    <h2
      v-if="items.length > 0"
      class="text-xs uppercase tracking-wide text-zinc-500 mb-2"
    >
      Contents
    </h2>
    <ul
      v-if="items.length > 0"
      class="space-y-1 border-l border-zinc-200 dark:border-zinc-800"
    >
      <li
        v-for="h in items"
        :key="h.slug"
        :style="{ paddingLeft: `${(h.level - minLevel) * 0.75 + 0.75}rem` }"
      >
        <a
          :href="`#${h.slug}`"
          class="block py-0.5 truncate hover:text-brand-blue dark:hover:text-brand-blue-dark"
          :class="
            activeSlug === h.slug
              ? 'text-brand-blue dark:text-brand-blue-dark font-medium'
              : 'text-zinc-600 dark:text-zinc-400'
          "
          @click="scrollTo(h.slug, $event)"
        >
          {{ h.text }}
        </a>
      </li>
    </ul>
  </nav>
</template>
