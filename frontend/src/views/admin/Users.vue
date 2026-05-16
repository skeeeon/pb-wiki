<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { pb } from '@/lib/pb'
import type { Role, UserRecord } from '@/lib/types'
import AdminNav from '@/components/admin/AdminNav.vue'

// A UI-side mirror of UserRecord that holds the groups field as a CSV string
// for in-place editing; we parse it back to an array on save.
interface UserRow extends UserRecord {
  groupsText: string
  saving?: boolean
}

const users = ref<UserRow[]>([])
const loading = ref(true)
const errorMsg = ref('')

const newEmail = ref('')
const newPassword = ref('')
const newRole = ref<Role>('viewer')
const creating = ref(false)

async function load() {
  loading.value = true
  errorMsg.value = ''
  try {
    const rows = await pb.collection('users').getFullList<UserRecord>({ sort: '+email' })
    users.value = rows.map((u) => ({
      ...u,
      groupsText: (u.groups ?? []).join(', '),
    }))
  } catch (err) {
    errorMsg.value = formatError(err)
  } finally {
    loading.value = false
  }
}

async function saveUser(u: UserRow) {
  u.saving = true
  errorMsg.value = ''
  try {
    const groups = u.groupsText
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
    await pb.collection('users').update(u.id, { role: u.role, groups })
    u.groups = groups
  } catch (err) {
    errorMsg.value = formatError(err)
  } finally {
    u.saving = false
  }
}

async function deleteUser(u: UserRow) {
  if (!confirm(`Delete user ${u.email}?`)) return
  try {
    await pb.collection('users').delete(u.id)
    users.value = users.value.filter((x) => x.id !== u.id)
  } catch (err) {
    errorMsg.value = formatError(err)
  }
}

async function createUser() {
  if (creating.value) return
  if (!newEmail.value || !newPassword.value) return
  creating.value = true
  errorMsg.value = ''
  try {
    await pb.collection('users').create({
      email: newEmail.value,
      password: newPassword.value,
      passwordConfirm: newPassword.value,
      role: newRole.value,
      emailVisibility: true,
    })
    newEmail.value = ''
    newPassword.value = ''
    newRole.value = 'viewer'
    await load()
  } catch (err) {
    errorMsg.value = formatError(err)
  } finally {
    creating.value = false
  }
}

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

onMounted(load)
</script>

<template>
  <div class="max-w-5xl mx-auto space-y-6">
    <AdminNav />

    <header>
      <h1 class="text-xl font-semibold">Users</h1>
      <p class="text-sm text-zinc-500">Manage roles and group memberships used by access rules.</p>
    </header>

    <p v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400">{{ errorMsg }}</p>

    <section
      class="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm overflow-hidden"
    >
      <header
        class="flex items-baseline justify-between px-6 py-3 bg-zinc-50 dark:bg-zinc-950/40 border-b border-zinc-200 dark:border-zinc-800"
      >
        <h2 class="text-sm font-semibold">All users</h2>
        <span class="text-xs text-zinc-500">{{ users.length }} total</span>
      </header>
      <table v-if="!loading && users.length > 0" class="w-full text-sm">
        <thead class="text-zinc-500 dark:text-zinc-400 text-xs uppercase tracking-wide">
          <tr class="border-b border-zinc-200 dark:border-zinc-800">
            <th class="text-left px-6 py-2 font-medium">Email</th>
            <th class="text-left px-3 py-2 font-medium w-32">Role</th>
            <th class="text-left px-3 py-2 font-medium">Groups</th>
            <th class="text-right px-6 py-2 font-medium w-40">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-200 dark:divide-zinc-800">
          <tr v-for="u in users" :key="u.id">
            <td class="px-6 py-2 font-mono text-xs">{{ u.email }}</td>
            <td class="px-3 py-2">
              <select
                v-model="u.role"
                class="rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm focus:outline-none focus:border-brand-blue"
              >
                <option value="admin">admin</option>
                <option value="editor">editor</option>
                <option value="viewer">viewer</option>
              </select>
            </td>
            <td class="px-3 py-2">
              <input
                v-model="u.groupsText"
                placeholder="comma, separated, groups"
                class="w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm font-mono focus:outline-none focus:border-brand-blue"
              />
            </td>
            <td class="px-6 py-2 text-right">
              <div class="inline-flex items-center gap-2">
                <button
                  type="button"
                  :disabled="u.saving"
                  class="rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 hover:bg-zinc-100 dark:hover:bg-zinc-800 px-3 py-1 text-xs font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                  @click="saveUser(u)"
                >
                  {{ u.saving ? 'Saving…' : 'Save' }}
                </button>
                <button
                  type="button"
                  class="text-xs text-red-600 dark:text-red-400 hover:underline"
                  @click="deleteUser(u)"
                >
                  Delete
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else-if="loading" class="px-6 py-4 text-sm text-zinc-500">Loading…</p>
      <p v-else class="px-6 py-4 text-sm text-zinc-500">No users yet.</p>
    </section>

    <form
      class="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm overflow-hidden"
      @submit.prevent="createUser"
    >
      <header class="px-6 pt-5 pb-2">
        <h2 class="text-base font-semibold">Add a user</h2>
        <p class="text-xs text-zinc-500 mt-0.5">
          The email domain must be allow-listed in PocketBase's users collection.
        </p>
      </header>

      <section class="px-6 pb-5 pt-3">
        <div class="grid gap-3 sm:grid-cols-[2fr_2fr_1fr]">
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300 font-medium">Email</span>
            <input
              v-model="newEmail"
              type="email"
              required
              placeholder="user@example.com"
              class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm focus:outline-none focus:border-brand-blue"
            />
          </label>
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300 font-medium">Password</span>
            <input
              v-model="newPassword"
              type="password"
              required
              placeholder="≥ 8 characters"
              autocomplete="new-password"
              class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm focus:outline-none focus:border-brand-blue"
            />
          </label>
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300 font-medium">Role</span>
            <select
              v-model="newRole"
              class="mt-1 block w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm focus:outline-none focus:border-brand-blue"
            >
              <option value="admin">admin</option>
              <option value="editor">editor</option>
              <option value="viewer">viewer</option>
            </select>
          </label>
        </div>
      </section>

      <div
        class="flex items-center justify-end gap-3 px-6 py-4 bg-zinc-50 dark:bg-zinc-950/40 border-t border-zinc-200 dark:border-zinc-800"
      >
        <button
          type="submit"
          :disabled="creating"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-4 py-1.5 text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {{ creating ? 'Creating…' : 'Create user' }}
        </button>
      </div>
    </form>
  </div>
</template>
