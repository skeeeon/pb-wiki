<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <!-- Top tabs share the same border treatment as Account cards: each link is
       a pill that gains a subtle background when active, mirroring the
       brand-blue accent used on the role chip in Account. -->
  <nav
    class="mb-6 flex items-center gap-1 border-b border-zinc-200 dark:border-zinc-800"
    aria-label="Admin sections"
  >
    <RouterLink
      to="/admin/users"
      class="px-3 py-2 -mb-px text-sm text-zinc-500 border-b-2 border-transparent hover:text-zinc-900 dark:hover:text-zinc-100"
      active-class="font-medium text-brand-blue dark:text-brand-blue-dark border-brand-blue dark:border-brand-blue-dark"
    >
      Users
    </RouterLink>
    <RouterLink
      to="/admin/access-rules"
      class="px-3 py-2 -mb-px text-sm text-zinc-500 border-b-2 border-transparent hover:text-zinc-900 dark:hover:text-zinc-100"
      active-class="font-medium text-brand-blue dark:text-brand-blue-dark border-brand-blue dark:border-brand-blue-dark"
    >
      Access rules
    </RouterLink>
    <RouterLink
      to="/admin/move"
      class="px-3 py-2 -mb-px text-sm text-zinc-500 border-b-2 border-transparent hover:text-zinc-900 dark:hover:text-zinc-100"
      active-class="font-medium text-brand-blue dark:text-brand-blue-dark border-brand-blue dark:border-brand-blue-dark"
    >
      Move
    </RouterLink>
    <RouterLink
      to="/admin/settings"
      class="px-3 py-2 -mb-px text-sm text-zinc-500 border-b-2 border-transparent hover:text-zinc-900 dark:hover:text-zinc-100"
      active-class="font-medium text-brand-blue dark:text-brand-blue-dark border-brand-blue dark:border-brand-blue-dark"
    >
      Settings
    </RouterLink>
  </nav>
</template>
