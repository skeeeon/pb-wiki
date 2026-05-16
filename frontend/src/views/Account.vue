<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { pb } from '@/lib/pb'
import { applyPwaState, clearCache as clearPwaCache, pwaEnabled } from '@/lib/pwa'
import { useAuthStore } from '@/stores/auth'
import type { UserRecord } from '@/lib/types'

const auth = useAuthStore()

const name = ref('')
const avatarFile = ref<File | null>(null)
const avatarPreview = ref<string | null>(null)
const clearAvatar = ref(false)
const saving = ref(false)
const errorMsg = ref('')
const successMsg = ref('')

// Re-seed the form when the underlying record arrives (e.g. on first mount
// before pb.authStore.onChange has fired) or when the user switches accounts.
watch(
  () => auth.record,
  (r) => {
    name.value = r?.name ?? ''
    avatarFile.value = null
    clearAvatar.value = false
    if (avatarPreview.value) URL.revokeObjectURL(avatarPreview.value)
    avatarPreview.value = null
  },
  { immediate: true },
)

const currentAvatarUrl = computed(() => {
  const r = auth.record
  if (!r || !r.avatar || clearAvatar.value) return null
  return pb.files.getURL(r, r.avatar)
})

function onAvatarChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  avatarFile.value = file
  clearAvatar.value = false
  if (avatarPreview.value) URL.revokeObjectURL(avatarPreview.value)
  avatarPreview.value = file ? URL.createObjectURL(file) : null
}

function removeAvatar() {
  avatarFile.value = null
  clearAvatar.value = true
  if (avatarPreview.value) URL.revokeObjectURL(avatarPreview.value)
  avatarPreview.value = null
}

const cacheBusy = ref(false)
const cacheMsg = ref('')

async function onTogglePwa(e: Event) {
  const checked = (e.target as HTMLInputElement).checked
  cacheBusy.value = true
  cacheMsg.value = ''
  try {
    await applyPwaState(checked)
    cacheMsg.value = checked ? 'Offline cache enabled.' : 'Offline cache disabled and cleared.'
  } catch (err) {
    cacheMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    cacheBusy.value = false
  }
}

async function clearCacheNow() {
  cacheBusy.value = true
  cacheMsg.value = ''
  try {
    await clearPwaCache()
    cacheMsg.value = 'Cached pages cleared.'
  } catch (err) {
    cacheMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    cacheBusy.value = false
  }
}

async function save() {
  if (!auth.record || saving.value) return
  saving.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
    // FormData is needed whenever a file is involved; we use it for the
    // clear-avatar case too so the request shape is consistent.
    const fd = new FormData()
    fd.append('name', name.value)
    if (avatarFile.value) {
      fd.append('avatar', avatarFile.value)
    } else if (clearAvatar.value) {
      fd.append('avatar', '')
    }
    await pb.collection('users').update<UserRecord>(auth.record.id, fd)
    // Refresh the cached auth record so the sidebar/header pick up the new
    // name and avatar without a page reload.
    await pb.collection('users').authRefresh<UserRecord>()
    successMsg.value = 'Saved.'
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="max-w-xl mx-auto space-y-6">
    <header>
      <h1 class="text-lg font-semibold">Account</h1>
      <p class="text-sm text-zinc-500">Update your profile.</p>
    </header>

    <form class="space-y-4" @submit.prevent="save">
      <div class="space-y-1">
        <span class="block text-sm text-zinc-700 dark:text-zinc-300">Email</span>
        <p class="text-sm font-mono text-zinc-900 dark:text-zinc-100">
          {{ auth.record?.email }}
        </p>
        <p class="text-xs text-zinc-500">
          Email changes require verification and aren't supported here yet.
        </p>
      </div>

      <label class="block text-sm">
        <span class="text-zinc-700 dark:text-zinc-300">Name</span>
        <input
          v-model="name"
          type="text"
          autocomplete="name"
          class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
        />
      </label>

      <div class="space-y-2">
        <span class="block text-sm text-zinc-700 dark:text-zinc-300">Avatar</span>
        <div class="flex items-center gap-4">
          <div
            class="w-16 h-16 rounded-full bg-brand-blue text-white text-2xl font-medium flex items-center justify-center shrink-0 overflow-hidden"
          >
            <img
              v-if="avatarPreview"
              :src="avatarPreview"
              :alt="auth.record?.email ?? ''"
              class="w-full h-full object-cover"
            />
            <img
              v-else-if="currentAvatarUrl"
              :src="currentAvatarUrl"
              :alt="auth.record?.email ?? ''"
              class="w-full h-full object-cover"
            />
            <span v-else>{{ (auth.record?.email ?? '?').charAt(0).toUpperCase() }}</span>
          </div>
          <div class="flex flex-col gap-2">
            <input
              type="file"
              accept="image/*"
              class="text-sm"
              @change="onAvatarChange"
            />
            <button
              v-if="currentAvatarUrl || avatarPreview"
              type="button"
              class="text-xs text-red-600 dark:text-red-400 hover:underline self-start"
              @click="removeAvatar"
            >
              Remove avatar
            </button>
          </div>
        </div>
      </div>

      <div class="flex items-center gap-3 pt-2">
        <button
          type="submit"
          :disabled="saving"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-3 py-1.5 text-sm font-medium disabled:opacity-60"
        >
          {{ saving ? 'Saving…' : 'Save changes' }}
        </button>
        <p v-if="successMsg" class="text-sm text-green-600 dark:text-green-400">{{ successMsg }}</p>
        <p v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400">{{ errorMsg }}</p>
      </div>
    </form>

    <section class="space-y-3 border-t border-zinc-200 dark:border-zinc-800 pt-6">
      <header>
        <h2 class="text-base font-semibold">Offline cache</h2>
        <p class="text-sm text-zinc-500">
          Caches pages you visit on this device so they remain readable offline.
          Only content you have access to is cached. The cache is cleared on sign-out.
        </p>
      </header>

      <label class="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          :checked="pwaEnabled"
          :disabled="cacheBusy"
          class="h-4 w-4"
          @change="onTogglePwa"
        />
        <span>Enable offline cache</span>
      </label>

      <div class="flex items-center gap-3">
        <button
          type="button"
          :disabled="cacheBusy"
          class="rounded-md border border-zinc-300 dark:border-zinc-700 px-3 py-1.5 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800 disabled:opacity-60"
          @click="clearCacheNow"
        >
          Clear cached pages
        </button>
        <p v-if="cacheMsg" class="text-sm text-zinc-600 dark:text-zinc-400">{{ cacheMsg }}</p>
      </div>
    </section>
  </div>
</template>
