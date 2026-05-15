<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <nav class="flex items-center gap-4 mb-6 text-sm border-b border-zinc-200 dark:border-zinc-800 pb-2">
    <RouterLink
      to="/admin/users"
      class="text-zinc-500 hover:underline"
      active-class="font-medium text-zinc-900 dark:text-zinc-100 no-underline"
    >
      Users
    </RouterLink>
    <RouterLink
      to="/admin/access-rules"
      class="text-zinc-500 hover:underline"
      active-class="font-medium text-zinc-900 dark:text-zinc-100 no-underline"
    >
      Access rules
    </RouterLink>
    <RouterLink
      to="/admin/settings"
      class="text-zinc-500 hover:underline"
      active-class="font-medium text-zinc-900 dark:text-zinc-100 no-underline"
    >
      Settings
    </RouterLink>
  </nav>
</template>
