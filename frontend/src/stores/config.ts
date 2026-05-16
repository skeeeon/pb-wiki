import { defineStore } from 'pinia'
import { ref } from 'vue'

import { pb } from '@/lib/pb'
import type { WikiConfig } from '@/lib/types'

const STORAGE_KEY = 'pb-wiki:config-cache'

// useConfigStore loads the wiki_config singleton row — viewable by anyone so
// the login page / unauth landing can render the wiki title.
export const useConfigStore = defineStore('config', () => {
  const config = ref<WikiConfig | null>(null)
  const loading = ref(false)
  const error = ref<unknown>(null)

  // Synchronously seed `config` from localStorage. Used at app startup so a
  // cold load doesn't block mount on a network round-trip — critical for
  // offline PWA opens. Returns true if a seed was applied.
  function loadFromStorage(): boolean {
    try {
      const cached = localStorage.getItem(STORAGE_KEY)
      if (!cached) return false
      config.value = JSON.parse(cached) as WikiConfig
      return true
    } catch {
      return false
    }
  }

  async function load(force = false) {
    if (!force && (config.value || loading.value)) return
    loading.value = true
    error.value = null
    try {
      const fresh = await pb
        .collection('wiki_config')
        .getFirstListItem<WikiConfig>('')
      config.value = fresh
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(fresh))
      } catch {
        /* ignore — private mode, quota, etc. */
      }
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  // Admin-only — caller must have role=admin or the update rule will reject.
  async function save(updates: Partial<WikiConfig>): Promise<WikiConfig> {
    if (!config.value) throw new Error('config not loaded yet')
    const updated = await pb
      .collection('wiki_config')
      .update<WikiConfig>(config.value.id, updates)
    config.value = updated
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
    } catch {
      /* ignore */
    }
    return updated
  }

  return { config, loading, error, load, save, loadFromStorage }
})
