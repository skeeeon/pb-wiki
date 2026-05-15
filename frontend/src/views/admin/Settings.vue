<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'

import { useConfigStore } from '@/stores/config'
import AdminNav from '@/components/admin/AdminNav.vue'

const config = useConfigStore()

const title = ref('')
const privateDefault = ref(false)
const requireLogin = ref(false)
const defaultLandingPath = ref('')

const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

// Hydrate the form whenever the underlying config loads or reloads.
watch(
  () => config.config,
  (c) => {
    if (!c) return
    title.value = c.title
    privateDefault.value = c.private_default
    requireLogin.value = c.require_login
    defaultLandingPath.value = c.default_landing_path
  },
  { immediate: true },
)

onMounted(() => {
  // Force a reload so admins always see the current persisted values, not a
  // stale cached copy from the initial app-mount load.
  void config.load(true)
})

async function save() {
  if (saving.value) return
  saving.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    await config.save({
      title: title.value,
      private_default: privateDefault.value,
      require_login: requireLogin.value,
      default_landing_path: defaultLandingPath.value,
    })
    successMsg.value = 'Saved.'
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="max-w-3xl mx-auto">
    <AdminNav />

    <header class="flex items-baseline justify-between mb-4">
      <h1 class="text-xl font-semibold">Settings</h1>
    </header>

    <p v-if="config.loading" class="text-sm text-zinc-500">Loading…</p>

    <form v-else class="space-y-5 max-w-xl" @submit.prevent="save">
      <label class="block text-sm">
        <span class="text-zinc-700 dark:text-zinc-300">Wiki title</span>
        <input
          v-model="title"
          class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
        />
      </label>

      <label class="flex items-start gap-2 text-sm">
        <input
          v-model="privateDefault"
          type="checkbox"
          class="mt-1"
        />
        <span>
          <span class="text-zinc-700 dark:text-zinc-300 font-medium">Private by default</span>
          <span class="block text-zinc-500">
            When checked, paths not matched by any access rule require login. When unchecked,
            unmatched paths are world-readable.
          </span>
        </span>
      </label>

      <label class="flex items-start gap-2 text-sm">
        <input
          v-model="requireLogin"
          type="checkbox"
          class="mt-1"
        />
        <span>
          <span class="text-zinc-700 dark:text-zinc-300 font-medium">Require login for the entire wiki</span>
          <span class="block text-zinc-500">
            When checked, anonymous visitors are redirected to the login page from every route —
            including the homepage. Overrides explicit <code>public</code> access rules.
          </span>
        </span>
      </label>

      <label class="block text-sm">
        <span class="text-zinc-700 dark:text-zinc-300">Default landing path</span>
        <input
          v-model="defaultLandingPath"
          placeholder="(homepage)"
          class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm font-mono"
        />
        <span class="block mt-1 text-xs text-zinc-500">
          Path the root URL <code>/</code> resolves to when there's no homepage document.
          Leave empty for the path-empty home convention.
        </span>
      </label>

      <div class="flex items-center gap-3">
        <button
          type="submit"
          :disabled="saving"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-3 py-2 text-sm font-medium disabled:opacity-60"
        >
          {{ saving ? 'Saving…' : 'Save settings' }}
        </button>
        <span v-if="successMsg" class="text-sm text-green-600 dark:text-green-400">{{ successMsg }}</span>
        <span v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400">{{ errorMsg }}</span>
      </div>
    </form>
  </div>
</template>
