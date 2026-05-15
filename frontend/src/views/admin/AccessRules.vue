<script setup lang="ts">
import { onMounted, ref } from 'vue'

import { pb } from '@/lib/pb'
import type { AccessLevel, AccessRuleRecord } from '@/lib/types'
import AdminNav from '@/components/admin/AdminNav.vue'

interface RuleRow extends AccessRuleRecord {
  groupsText: string
  saving?: boolean
}

const rules = ref<RuleRow[]>([])
const loading = ref(true)
const errorMsg = ref('')

const newRule = ref<{
  pattern: string
  access: AccessLevel
  groupsText: string
  priority: number
  description: string
}>({
  pattern: '',
  access: 'public',
  groupsText: '',
  priority: 100,
  description: '',
})
const creating = ref(false)

async function load() {
  loading.value = true
  errorMsg.value = ''
  try {
    const rows = await pb
      .collection('access_rules')
      .getFullList<AccessRuleRecord>({ sort: '+priority' })
    rules.value = rows.map((r) => ({
      ...r,
      groupsText: (r.groups ?? []).join(', '),
    }))
  } catch (err) {
    errorMsg.value = formatError(err)
  } finally {
    loading.value = false
  }
}

async function saveRule(r: RuleRow) {
  r.saving = true
  errorMsg.value = ''
  try {
    await pb.collection('access_rules').update(r.id, {
      pattern: r.pattern,
      access: r.access,
      groups: parseCSV(r.groupsText),
      priority: r.priority,
      description: r.description,
    })
  } catch (err) {
    errorMsg.value = formatError(err)
  } finally {
    r.saving = false
  }
}

async function deleteRule(r: RuleRow) {
  if (!confirm(`Delete rule "${r.pattern}"?`)) return
  try {
    await pb.collection('access_rules').delete(r.id)
    rules.value = rules.value.filter((x) => x.id !== r.id)
  } catch (err) {
    errorMsg.value = formatError(err)
  }
}

async function createRule() {
  if (creating.value) return
  if (!newRule.value.pattern) return
  creating.value = true
  errorMsg.value = ''
  try {
    await pb.collection('access_rules').create({
      pattern: newRule.value.pattern,
      access: newRule.value.access,
      groups: parseCSV(newRule.value.groupsText),
      priority: newRule.value.priority,
      description: newRule.value.description,
    })
    newRule.value = {
      pattern: '',
      access: 'public',
      groupsText: '',
      priority: 100,
      description: '',
    }
    await load()
  } catch (err) {
    errorMsg.value = formatError(err)
  } finally {
    creating.value = false
  }
}

function parseCSV(s: string): string[] {
  return s
    .split(',')
    .map((x) => x.trim())
    .filter(Boolean)
}

function formatError(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

onMounted(load)
</script>

<template>
  <div>
    <AdminNav />

    <header class="flex items-baseline justify-between mb-4">
      <h1 class="text-xl font-semibold">Access rules</h1>
    </header>

    <p class="text-sm text-zinc-500 mb-4">
      Rules are evaluated in ascending <code>priority</code> order and the first matching pattern wins.
      Use lower numbers for rules that must take precedence.
    </p>

    <p v-if="errorMsg" class="text-sm text-red-600 dark:text-red-400 mb-3">{{ errorMsg }}</p>

    <section class="border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden mb-6">
      <table v-if="!loading && rules.length > 0" class="w-full text-sm">
        <thead class="bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
          <tr>
            <th class="text-left px-3 py-2 font-medium w-20">Priority</th>
            <th class="text-left px-3 py-2 font-medium">Pattern</th>
            <th class="text-left px-3 py-2 font-medium w-32">Access</th>
            <th class="text-left px-3 py-2 font-medium">Groups</th>
            <th class="text-left px-3 py-2 font-medium">Description</th>
            <th class="text-right px-3 py-2 font-medium w-32">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-zinc-200 dark:divide-zinc-800">
          <tr v-for="r in rules" :key="r.id">
            <td class="px-3 py-2">
              <input
                v-model.number="r.priority"
                type="number"
                class="w-16 rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm"
              />
            </td>
            <td class="px-3 py-2">
              <input
                v-model="r.pattern"
                class="w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm font-mono"
              />
            </td>
            <td class="px-3 py-2">
              <select
                v-model="r.access"
                class="rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm"
              >
                <option value="public">public</option>
                <option value="private">private</option>
                <option value="restricted">restricted</option>
              </select>
            </td>
            <td class="px-3 py-2">
              <input
                v-model="r.groupsText"
                :disabled="r.access !== 'restricted'"
                placeholder="(restricted only)"
                class="w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm font-mono disabled:opacity-50"
              />
            </td>
            <td class="px-3 py-2">
              <input
                v-model="r.description"
                class="w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-sm"
              />
            </td>
            <td class="px-3 py-2 text-right space-x-3">
              <button
                type="button"
                :disabled="r.saving"
                class="text-sm underline disabled:opacity-60"
                @click="saveRule(r)"
              >
                {{ r.saving ? 'Saving…' : 'Save' }}
              </button>
              <button
                type="button"
                class="text-sm text-red-600 dark:text-red-400 underline"
                @click="deleteRule(r)"
              >
                Delete
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else-if="loading" class="px-3 py-3 text-zinc-500">Loading…</p>
      <p v-else class="px-3 py-3 text-zinc-500">No rules yet. The wiki falls back to <code>wiki_config.private_default</code>.</p>
    </section>

    <section class="border border-zinc-200 dark:border-zinc-800 rounded-lg p-4">
      <h2 class="text-sm font-semibold mb-3">Add a rule</h2>
      <form class="space-y-3" @submit.prevent="createRule">
        <div class="grid gap-3 sm:grid-cols-[80px_1fr_140px]">
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300">Priority</span>
            <input
              v-model.number="newRule.priority"
              type="number"
              class="mt-1 block w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-2 text-sm"
            />
          </label>
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300">Pattern</span>
            <input
              v-model="newRule.pattern"
              required
              placeholder="finance/**"
              class="mt-1 block w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-2 text-sm font-mono"
            />
          </label>
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300">Access</span>
            <select
              v-model="newRule.access"
              class="mt-1 block w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-2 text-sm"
            >
              <option value="public">public</option>
              <option value="private">private</option>
              <option value="restricted">restricted</option>
            </select>
          </label>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300">Groups (for restricted)</span>
            <input
              v-model="newRule.groupsText"
              :disabled="newRule.access !== 'restricted'"
              placeholder="finance, execs"
              class="mt-1 block w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-2 text-sm font-mono disabled:opacity-50"
            />
          </label>
          <label class="block text-sm">
            <span class="text-zinc-700 dark:text-zinc-300">Description</span>
            <input
              v-model="newRule.description"
              class="mt-1 block w-full rounded border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-2 text-sm"
            />
          </label>
        </div>
        <button
          type="submit"
          :disabled="creating"
          class="rounded-md bg-brand-red hover:bg-brand-red-hover text-white px-3 py-2 text-sm font-medium disabled:opacity-60"
        >
          {{ creating ? 'Creating…' : 'Create rule' }}
        </button>
      </form>
    </section>
  </div>
</template>
