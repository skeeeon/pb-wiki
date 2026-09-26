<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <!-- Top tabs share the same border treatment as Account cards: each link is
       a pill that gains a subtle background when active, mirroring the
       primary accent used on the role chip in Account. -->
  <nav
    class="mb-6 flex items-center gap-1 border-b border-slate-200 dark:border-slate-800"
    aria-label="Admin sections"
  >
    <RouterLink
      to="/admin/users"
      class="px-3 py-2 -mb-px text-sm text-slate-500 border-b-2 border-transparent hover:text-slate-900 dark:hover:text-slate-100"
      active-class="font-medium text-primary border-primary"
    >
      Users
    </RouterLink>
    <RouterLink
      to="/admin/move"
      class="px-3 py-2 -mb-px text-sm text-slate-500 border-b-2 border-transparent hover:text-slate-900 dark:hover:text-slate-100"
      active-class="font-medium text-primary border-primary"
    >
      Move
    </RouterLink>
    <RouterLink
      to="/admin/settings"
      class="px-3 py-2 -mb-px text-sm text-slate-500 border-b-2 border-transparent hover:text-slate-900 dark:hover:text-slate-100"
      active-class="font-medium text-primary border-primary"
    >
      Settings
    </RouterLink>
  </nav>
</template>
