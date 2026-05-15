<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import type { TreeNode } from './sidebarTree'

defineProps<{ node: TreeNode; depth: number }>()

const route = useRoute()
</script>

<template>
  <li>
    <RouterLink
      :to="`/doc/${node.fullPath}`"
      class="flex items-center py-1 px-2 rounded text-sm hover:bg-zinc-100 dark:hover:bg-zinc-800"
      :class="{
        'bg-zinc-100 dark:bg-zinc-800 font-medium':
          route.path === `/doc/${node.fullPath}` ||
          (route.path === '/' && node.fullPath === ''),
      }"
      :style="{ paddingLeft: 8 + depth * 12 + 'px' }"
    >
      <span class="truncate">{{ node.title || node.segment || 'Home' }}</span>
    </RouterLink>
    <ul v-if="node.children.length > 0">
      <SidebarTreeItem
        v-for="child in node.children"
        :key="child.fullPath"
        :node="child"
        :depth="depth + 1"
      />
    </ul>
  </li>
</template>
