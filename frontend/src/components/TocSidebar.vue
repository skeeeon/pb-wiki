<script setup lang="ts">
import { computed } from 'vue'
import type { Heading } from '@/lib/markdown'

const props = defineProps<{ headings: Heading[] }>()

// h1-h4 land in the TOC. h5/h6 are excluded — they produce too dense a tree
// to be useful as nav.
const items = computed(() => props.headings.filter((h) => h.level >= 1 && h.level <= 4))

// Re-base the level so the smallest visible heading sits at indent 0.
const minLevel = computed(() => items.value.reduce((m, h) => Math.min(m, h.level), 6))

function scrollTo(slug: string, ev: MouseEvent) {
  const el = document.getElementById(slug)
  if (!el) return
  ev.preventDefault()
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  history.replaceState(null, '', `#${slug}`)
}
</script>

<template>
  <nav v-if="items.length > 0" aria-label="On this page" class="text-sm">
    <h2 class="text-xs uppercase tracking-wide text-zinc-500 mb-2">On this page</h2>
    <ul class="space-y-1 border-l border-zinc-200 dark:border-zinc-800">
      <li
        v-for="h in items"
        :key="h.slug"
        :style="{ paddingLeft: `${(h.level - minLevel) * 0.75 + 0.75}rem` }"
      >
        <a
          :href="`#${h.slug}`"
          class="block py-0.5 text-zinc-600 dark:text-zinc-400 hover:text-brand-blue dark:hover:text-brand-blue-dark truncate"
          @click="scrollTo(h.slug, $event)"
        >
          {{ h.text }}
        </a>
      </li>
    </ul>
  </nav>
</template>
