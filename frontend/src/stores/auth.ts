import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

import { pb } from '@/lib/pb'
import type { Role, UserRecord } from '@/lib/types'

export const useAuthStore = defineStore('auth', () => {
  // Seed from PB's persisted auth store (localStorage) so a refresh keeps the
  // user logged in. We then subscribe to onChange so any later login/logout
  // call flows through reactively to every Vue component.
  const token = ref<string>(pb.authStore.token)
  const record = ref<UserRecord | null>(pb.authStore.record as UserRecord | null)

  pb.authStore.onChange((newToken, newRecord) => {
    token.value = newToken
    record.value = newRecord as UserRecord | null
  })

  const isAuthenticated = computed(() => !!token.value)
  const role = computed<Role | null>(() => record.value?.role ?? null)
  const isAdmin = computed(() => role.value === 'admin')
  // editor-or-better — convenient for "can write" gating.
  const isEditor = computed(() => role.value === 'admin' || role.value === 'editor')
  const groups = computed<string[]>(() => record.value?.groups ?? [])

  async function loginWithPassword(email: string, password: string) {
    return await pb.collection('users').authWithPassword<UserRecord>(email, password)
  }

  async function loginWithOAuth(provider: string) {
    return await pb.collection('users').authWithOAuth2<UserRecord>({ provider })
  }

  // Self-registration. The users collection's CreateRule is open and
  // domain-gated via PB's EmailField.OnlyDomains (configured in the admin
  // UI), so this call will be rejected server-side for disallowed domains.
  // The auth hook assigns role=viewer on creation. We immediately sign the
  // new user in so they land on the wiki without a second round-trip.
  async function register(email: string, password: string, passwordConfirm: string) {
    await pb.collection('users').create({ email, password, passwordConfirm })
    return await pb.collection('users').authWithPassword<UserRecord>(email, password)
  }

  function logout() {
    pb.authStore.clear()
  }

  return {
    token,
    record,
    isAuthenticated,
    role,
    isAdmin,
    isEditor,
    groups,
    loginWithPassword,
    loginWithOAuth,
    register,
    logout,
  }
})
