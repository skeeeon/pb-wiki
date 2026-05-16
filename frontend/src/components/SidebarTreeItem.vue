<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import type { TreeNode } from './sidebarTree'

const props = defineProps<{
  node: TreeNode
  depth: number
  expanded: Set<string>
  onToggle: (path: string) => void
}>()

const route = useRoute()

const hasChildren = computed(() => props.node.children.length > 0)
const isOpen = computed(() => props.expanded.has(props.node.fullPath))
const isActive = computed(
  () =>
    route.path === `/doc/${props.node.fullPath}` ||
    (route.path === '/' && props.node.fullPath === ''),
)
</script>

<template>
  <li>
    <div
      class="flex items-center rounded text-[15px] hover:bg-zinc-100 dark:hover:bg-zinc-800"
      :class="{
        'bg-brand-blue/10 text-brand-blue dark:text-brand-blue-dark font-medium': isActive,
      }"
      :style="{ paddingLeft: 4 + depth * 12 + 'px' }"
    >
      <!-- Chevron toggle — only rendered when the node has children; otherwise
           a 20px spacer keeps the title aligned with siblings that do. -->
      <button
        v-if="hasChildren"
        type="button"
        class="shrink-0 w-5 h-6 flex items-center justify-center text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        :aria-label="isOpen ? 'Collapse' : 'Expand'"
        @click="onToggle(node.fullPath)"
      >
        <svg
          class="w-3 h-3 transition-transform"
          :class="{ 'rotate-90': isOpen }"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m9 18 6-6-6-6" />
        </svg>
      </button>
      <span v-else class="w-5 shrink-0" aria-hidden="true" />

      <RouterLink
        :to="`/doc/${node.fullPath}`"
        class="flex-1 min-w-0 py-1 pr-2 truncate"
      >
        {{ node.title || node.segment || 'Home' }}
      </RouterLink>
    </div>

    <ul v-if="hasChildren && isOpen">
      <SidebarTreeItem
        v-for="child in node.children"
        :key="child.fullPath"
        :node="child"
        :depth="depth + 1"
        :expanded="expanded"
        :on-toggle="onToggle"
      />
    </ul>
  </li>
</template>
