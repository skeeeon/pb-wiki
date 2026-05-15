<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

import { pb } from '@/lib/pb'
import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const config = useConfigStore()

const email = ref('')
const password = ref('')
const submitting = ref(false)
const errorMsg = ref('')

interface OAuthProvider {
  name: string
  displayName: string
}
const providers = ref<OAuthProvider[]>([])

onMounted(async () => {
  await config.load()
  try {
    const methods = await pb.collection('users').listAuthMethods()
    providers.value =
      methods.oauth2?.providers?.map((p) => ({
        name: p.name,
        displayName: p.displayName ?? p.name,
      })) ?? []
  } catch {
    // listAuthMethods is best-effort; surface no error if it fails.
  }
})

async function submitPassword() {
  if (!email.value || !password.value) return
  submitting.value = true
  errorMsg.value = ''
  try {
    await auth.loginWithPassword(email.value, password.value)
    redirectAfterLogin()
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : 'Login failed.'
  } finally {
    submitting.value = false
  }
}

async function submitOAuth(provider: string) {
  submitting.value = true
  errorMsg.value = ''
  try {
    await auth.loginWithOAuth(provider)
    redirectAfterLogin()
  } catch (err) {
    errorMsg.value = err instanceof Error ? err.message : 'OAuth login failed.'
  } finally {
    submitting.value = false
  }
}

function redirectAfterLogin() {
  const redirect = (route.query.redirect as string) || '/'
  router.replace(redirect)
}
</script>

<template>
  <div class="min-h-dvh flex items-center justify-center p-6 bg-zinc-50 dark:bg-zinc-950">
    <div class="w-full max-w-sm space-y-6 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-xl p-6 shadow-sm">
      <header class="space-y-1 text-center">
        <h1 class="text-xl font-semibold">{{ config.config?.title || 'pb-wiki' }}</h1>
        <p class="text-sm text-zinc-500">Sign in</p>
      </header>

      <form class="space-y-3" @submit.prevent="submitPassword">
        <label class="block text-sm">
          <span class="text-zinc-700 dark:text-zinc-300">Email</span>
          <input
            v-model="email"
            type="email"
            autocomplete="username"
            required
            class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
          />
        </label>
        <label class="block text-sm">
          <span class="text-zinc-700 dark:text-zinc-300">Password</span>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            required
            class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
          />
        </label>

        <button
          type="submit"
          :disabled="submitting"
          class="w-full rounded-md bg-zinc-900 dark:bg-zinc-100 text-zinc-50 dark:text-zinc-900 px-3 py-2 text-sm font-medium disabled:opacity-60"
        >
          {{ submitting ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>

      <div v-if="providers.length > 0" class="space-y-2">
        <div class="text-center text-xs uppercase tracking-wide text-zinc-500">or</div>
        <button
          v-for="p in providers"
          :key="p.name"
          type="button"
          :disabled="submitting"
          class="w-full rounded-md border border-zinc-300 dark:border-zinc-700 px-3 py-2 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800 disabled:opacity-60"
          @click="submitOAuth(p.name)"
        >
          Continue with {{ p.displayName }}
        </button>
      </div>

      <p v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400 text-center">
        {{ errorMsg }}
      </p>
    </div>
  </div>
</template>
