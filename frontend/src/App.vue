<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import Sidebar from '@/components/Sidebar.vue'

const auth = useAuthStore()
const config = useConfigStore()
const route = useRoute()

// Hide the chrome on the login page so it can render full-screen.
const showChrome = computed(() => route.name !== 'login')

onMounted(() => {
  config.load()
})
</script>

<template>
  <RouterView v-if="!showChrome" />

  <div v-else class="min-h-dvh flex flex-col bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100">
    <header class="border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
      <div class="max-w-6xl mx-auto px-4 h-12 flex items-center justify-between">
        <RouterLink to="/" class="text-sm font-semibold">
          {{ config.config?.title || 'pb-wiki' }}
        </RouterLink>

        <nav class="flex items-center gap-3 text-sm">
          <template v-if="auth.isAuthenticated">
            <RouterLink v-if="auth.isAdmin" to="/admin" class="underline">Admin</RouterLink>
            <span class="text-zinc-500">
              {{ auth.record?.email }}
              <span class="text-xs text-zinc-400">({{ auth.role }})</span>
            </span>
            <button class="underline" @click="auth.logout">Sign out</button>
          </template>
          <template v-else>
            <RouterLink to="/login" class="underline">Sign in</RouterLink>
          </template>
        </nav>
      </div>
    </header>

    <div class="flex-1 flex">
      <aside class="hidden md:block w-64 border-r border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3 overflow-y-auto">
        <Sidebar />
      </aside>

      <main class="flex-1 p-6 overflow-x-auto">
        <div class="max-w-3xl mx-auto">
          <RouterView />
        </div>
      </main>
    </div>
  </div>
</template>
