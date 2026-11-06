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

  // The session in localStorage is only the client's copy. The server can
  // stop honouring it while the client still holds it: the token expires, the
  // password changes, the user is deleted, or a fresh database is deployed.
  // PocketBase then treats each request as a guest instead of answering 401,
  // so the UI still shows editor controls while every write fails the
  // collection rule and comes back as 404. The cached role and groups can
  // also be out of date after an admin changes them.
  //
  // verifySession asks the server. A rejected token clears the session, so
  // the UI shows the user as signed out. A good token refreshes the token and
  // the record. Network errors keep the session, so being offline does not
  // sign the user out.
  let lastVerified = 0
  async function verifySession() {
    if (!pb.authStore.token) return
    if (!pb.authStore.isValid) {
      pb.authStore.clear()
      return
    }
    try {
      await pb.collection('users').authRefresh<UserRecord>({ requestKey: 'auth-verify' })
      lastVerified = Date.now()
    } catch (err: unknown) {
      const status = (err as { status?: number })?.status ?? 0
      if (status === 401 || status === 403 || status === 404) {
        pb.authStore.clear()
      }
    }
  }

  // Check again when the tab comes back after a while: a tab left open for
  // days is the usual way to be holding a session the server has dropped.
  const REVERIFY_AFTER_MS = 5 * 60 * 1000
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible' && Date.now() - lastVerified > REVERIFY_AFTER_MS) {
      void verifySession()
    }
  })

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
    verifySession,
  }
})
