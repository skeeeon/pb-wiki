<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'

const props = defineProps<{ path: string }>()

const crumbs = computed(() => {
  if (!props.path) return []
  const segments = props.path.split('/')
  let acc = ''
  return segments.map((seg) => {
    acc = acc ? `${acc}/${seg}` : seg
    return { label: seg, to: `/doc/${acc}` }
  })
})
</script>

<template>
  <nav v-if="crumbs.length > 0" class="text-sm text-zinc-500 mb-3 flex items-center flex-wrap gap-x-1">
    <RouterLink to="/" class="hover:underline">Home</RouterLink>
    <template v-for="(c, i) in crumbs" :key="c.to">
      <span aria-hidden="true">/</span>
      <RouterLink
        :to="c.to"
        class="hover:underline"
        :class="{ 'text-zinc-900 dark:text-zinc-100 font-medium': i === crumbs.length - 1 }"
      >
        {{ c.label }}
      </RouterLink>
    </template>
  </nav>
</template>
