<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { pb } from '@/lib/pb'
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

const oldPassword = ref('')
const newPassword = ref('')
const newPasswordConfirm = ref('')
const changingPassword = ref(false)
const pwErrorMsg = ref('')
const pwSuccessMsg = ref('')

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

const initial = computed(() => (auth.record?.email ?? '?').charAt(0).toUpperCase())
const displayName = computed(() => auth.record?.name || auth.record?.email || 'Unknown user')

const dirty = computed(() => {
  const r = auth.record
  if (!r) return false
  if ((name.value ?? '') !== (r.name ?? '')) return true
  if (avatarFile.value) return true
  if (clearAvatar.value && r.avatar) return true
  return false
})

// Clear the post-save confirmation as soon as the form is touched again.
watch(dirty, (isDirty) => {
  if (isDirty) successMsg.value = ''
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

async function save() {
  if (!auth.record || saving.value) return
  saving.value = true
  errorMsg.value = ''
  successMsg.value = ''
  try {
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
    successMsg.value = 'Profile updated.'
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    saving.value = false
  }
}

const passwordFormValid = computed(
  () =>
    oldPassword.value.length > 0 &&
    newPassword.value.length > 0 &&
    newPassword.value === newPasswordConfirm.value,
)

async function changePassword() {
  if (!auth.record || changingPassword.value || !passwordFormValid.value) return
  changingPassword.value = true
  pwErrorMsg.value = ''
  pwSuccessMsg.value = ''
  try {
    await pb.collection('users').update(auth.record.id, {
      oldPassword: oldPassword.value,
      password: newPassword.value,
      passwordConfirm: newPasswordConfirm.value,
    })
    // PocketBase rotates the user's tokenKey on password change, which
    // invalidates the current auth token. Re-authenticate with the new
    // password so the user stays signed in without bouncing to /login.
    const email = auth.record.email
    const fresh = newPassword.value
    oldPassword.value = ''
    newPassword.value = ''
    newPasswordConfirm.value = ''
    await pb.collection('users').authWithPassword<UserRecord>(email, fresh)
    pwSuccessMsg.value = 'Password updated.'
  } catch (err) {
    pwErrorMsg.value = err instanceof Error ? err.message : String(err)
  } finally {
    changingPassword.value = false
  }
}

// Clear the password-change confirmation as soon as the fields are touched again.
watch([oldPassword, newPassword, newPasswordConfirm], () => {
  if (pwSuccessMsg.value) pwSuccessMsg.value = ''
})
</script>

<template>
  <div class="max-w-2xl mx-auto space-y-6">
    <header>
      <h1 class="text-xl font-semibold">Account</h1>
      <p class="text-sm text-zinc-500">Manage your profile and how others see you.</p>
    </header>

    <form
      class="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm overflow-hidden"
      @submit.prevent="save"
    >
      <!-- Identity strip: large avatar, display name, email, role chip,
           plus the avatar-edit controls. Visually distinct so the user gets
           an at-a-glance preview of how they appear elsewhere in the app. -->
      <section class="flex items-start gap-5 p-6 bg-zinc-50 dark:bg-zinc-950/40 border-b border-zinc-200 dark:border-zinc-800">
        <div
          class="w-20 h-20 rounded-full bg-brand-blue text-white text-3xl font-medium
                 flex items-center justify-center shrink-0 overflow-hidden
                 ring-2 ring-white dark:ring-zinc-900"
        >
          <img
            v-if="avatarPreview"
            :src="avatarPreview"
            :alt="displayName"
            class="w-full h-full object-cover"
          />
          <img
            v-else-if="currentAvatarUrl"
            :src="currentAvatarUrl"
            :alt="displayName"
            class="w-full h-full object-cover"
          />
          <span v-else>{{ initial }}</span>
        </div>

        <div class="min-w-0 flex-1 space-y-1.5">
          <p class="text-base font-medium truncate">{{ displayName }}</p>
          <p class="text-xs text-zinc-500 font-mono truncate">{{ auth.record?.email }}</p>
          <span
            v-if="auth.role"
            class="inline-flex items-center rounded-full bg-brand-blue/10 text-brand-blue dark:text-brand-blue-dark text-xs font-medium px-2 py-0.5"
          >
            {{ auth.role }}
          </span>
        </div>

        <div class="flex flex-col items-end gap-1.5 shrink-0">
          <label
            for="avatar-input"
            class="rounded-md border border-zinc-300 dark:border-zinc-700 px-3 py-1.5 text-sm font-medium
                   cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 bg-white dark:bg-zinc-900"
          >
            Change…
          </label>
          <input
            id="avatar-input"
            type="file"
            accept="image/*"
            class="sr-only"
            @change="onAvatarChange"
          />
          <button
            v-if="currentAvatarUrl || avatarPreview"
            type="button"
            class="text-xs text-red-600 dark:text-red-400 hover:underline"
            @click="removeAvatar"
          >
            Remove avatar
          </button>
        </div>
      </section>

      <!-- Editable fields. -->
      <section class="p-6 space-y-5">
        <label class="block text-sm">
          <span class="text-zinc-700 dark:text-zinc-300 font-medium">Display name</span>
          <input
            v-model="name"
            type="text"
            autocomplete="name"
            placeholder="Your name"
            class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm
                   focus:outline-none focus:border-brand-blue"
          />
          <span class="mt-1.5 block text-xs text-zinc-500">
            Shown in the sidebar and beside pages you've edited.
          </span>
        </label>

        <div class="space-y-1">
          <span class="block text-sm font-medium text-zinc-700 dark:text-zinc-300">Email</span>
          <p class="text-sm font-mono text-zinc-900 dark:text-zinc-100 truncate">
            {{ auth.record?.email }}
          </p>
          <p class="text-xs text-zinc-500">
            Email changes require verification and aren't supported here yet.
          </p>
        </div>
      </section>

      <!-- Footer actions — sit on a tinted strip so the primary action is
           easy to find without scanning the form. -->
      <div
        class="flex items-center justify-between gap-3 px-6 py-4
               bg-zinc-50 dark:bg-zinc-950/40 border-t border-zinc-200 dark:border-zinc-800"
      >
        <p class="text-sm min-h-[1.25rem]">
          <span v-if="successMsg" class="text-green-600 dark:text-green-400">{{ successMsg }}</span>
          <span v-else-if="errorMsg" class="text-red-600 dark:text-red-400">{{ errorMsg }}</span>
          <span v-else-if="dirty" class="text-zinc-500">Unsaved changes</span>
        </p>
        <button
          type="submit"
          :disabled="saving || !dirty"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-4 py-1.5 text-sm font-medium
                 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ saving ? 'Saving…' : 'Save changes' }}
        </button>
      </div>
    </form>

    <form
      class="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm overflow-hidden"
      autocomplete="off"
      @submit.prevent="changePassword"
    >
      <header class="px-6 pt-5 pb-2">
        <h2 class="text-base font-semibold">Change password</h2>
        <p class="text-xs text-zinc-500 mt-0.5">
          You'll stay signed in on this device after the change.
        </p>
      </header>

      <section class="px-6 pb-5 pt-3 space-y-4">
        <!-- Hidden username field is a hint for password managers so they
             save the new password against the correct account. -->
        <input
          type="email"
          autocomplete="username"
          :value="auth.record?.email ?? ''"
          class="hidden"
          readonly
          tabindex="-1"
        />
        <label class="block text-sm">
          <span class="text-zinc-700 dark:text-zinc-300 font-medium">Current password</span>
          <input
            v-model="oldPassword"
            type="password"
            autocomplete="current-password"
            class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm
                   focus:outline-none focus:border-brand-blue"
          />
        </label>
        <label class="block text-sm">
          <span class="text-zinc-700 dark:text-zinc-300 font-medium">New password</span>
          <input
            v-model="newPassword"
            type="password"
            autocomplete="new-password"
            minlength="8"
            class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm
                   focus:outline-none focus:border-brand-blue"
          />
        </label>
        <label class="block text-sm">
          <span class="text-zinc-700 dark:text-zinc-300 font-medium">Confirm new password</span>
          <input
            v-model="newPasswordConfirm"
            type="password"
            autocomplete="new-password"
            minlength="8"
            class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm
                   focus:outline-none focus:border-brand-blue"
          />
          <span
            v-if="newPasswordConfirm && newPassword !== newPasswordConfirm"
            class="mt-1.5 block text-xs text-red-600 dark:text-red-400"
          >
            Passwords don't match.
          </span>
        </label>
      </section>

      <div
        class="flex items-center justify-between gap-3 px-6 py-4
               bg-zinc-50 dark:bg-zinc-950/40 border-t border-zinc-200 dark:border-zinc-800"
      >
        <p class="text-sm min-h-[1.25rem]">
          <span v-if="pwSuccessMsg" class="text-green-600 dark:text-green-400">{{ pwSuccessMsg }}</span>
          <span v-else-if="pwErrorMsg" class="text-red-600 dark:text-red-400">{{ pwErrorMsg }}</span>
        </p>
        <button
          type="submit"
          :disabled="changingPassword || !passwordFormValid"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-4 py-1.5 text-sm font-medium
                 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ changingPassword ? 'Updating…' : 'Update password' }}
        </button>
      </div>
    </form>
  </div>
</template>
