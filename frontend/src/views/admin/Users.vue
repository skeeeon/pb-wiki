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
  <div class="max-w-5xl mx-auto">
    <AdminNav />

    <header class="flex items-baseline justify-between mb-4">
      <h1 class="text-xl font-semibold">Users</h1>
    </header>

    <p v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400 mb-3">{{ errorMsg }}</p>

    <section class="border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden mb-6">
      <table v-if="!loading && users.length > 0" class="w-full text-sm">
        <thead class="bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
          <tr>
            <th class="text-left px-3 py-2 font-medium">Email</th>
            <th class="text-left px-3 py-2 font-medium w-32">Role</th>
            <th class="text-left px-3 py-2 font-medium">Groups</th>
            <th class="text-right px-3 py-2 font-medium w-40">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-200 dark:divide-zinc-800">
          <tr v-for="u in users" :key="u.id">
            <td class="px-3 py-2 font-mono text-xs">{{ u.email }}</td>
            <td class="px-3 py-2">
              <select
                v-model="u.role"
                class="rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm"
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
                class="w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm font-mono"
              />
            </td>
            <td class="px-3 py-2 text-right space-x-3">
              <button
                type="button"
                :disabled="u.saving"
                class="text-sm underline disabled:opacity-60"
                @click="saveUser(u)"
              >
                {{ u.saving ? 'Saving…' : 'Save' }}
              </button>
              <button
                type="button"
                class="text-sm text-red-600 dark:text-red-400 underline"
                @click="deleteUser(u)"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else-if="loading" class="px-3 py-3 text-zinc-500">Loading…</p>
      <p v-else class="px-3 py-3 text-zinc-500">No users yet.</p>
    </section>

    <section class="border border-zinc-200 dark:border-zinc-800 rounded-lg p-4">
      <h2 class="text-sm font-semibold mb-3">Add a user</h2>
      <form class="grid gap-3 sm:grid-cols-[2fr_2fr_1fr_auto]" @submit.prevent="createUser">
        <input
          v-model="newEmail"
          type="email"
          required
          placeholder="user@example.com"
          class="rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
        />
        <input
          v-model="newPassword"
          type="password"
          required
          placeholder="Password (≥ 8 chars)"
          autocomplete="new-password"
          class="rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
        />
        <select
          v-model="newRole"
          class="rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm"
        >
          <option value="admin">admin</option>
          <option value="editor">editor</option>
          <option value="viewer">viewer</option>
        </select>
        <button
          type="submit"
          :disabled="creating"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-3 py-2 text-sm font-medium disabled:opacity-60"
        >
          {{ creating ? 'Creating…' : 'Create' }}
        </button>
      </form>
    </section>
  </div>
</template>
