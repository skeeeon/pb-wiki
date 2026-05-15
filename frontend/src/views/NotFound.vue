<script setup lang="ts">
import { useRoute, RouterLink } from 'vue-router'

const route = useRoute()
</script>

<template>
  <div class="max-w-md mx-auto text-center py-16 space-y-3">
    <h1 class="text-4xl font-semibold">404</h1>
    <p class="text-zinc-600 dark:text-zinc-400">
      Nothing matches <code>{{ route.fullPath }}</code>.
    </p>
    <RouterLink class="inline-block text-sm underline" to="/">Back to home</RouterLink>
  </div>
</template>
