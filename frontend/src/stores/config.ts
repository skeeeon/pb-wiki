import { defineStore } from 'pinia'
import { ref } from 'vue'

import { pb } from '@/lib/pb'
import type { WikiConfig } from '@/lib/types'

// useConfigStore loads the wiki_config singleton row — viewable by anyone so
// the login page / unauth landing can render the wiki title.
export const useConfigStore = defineStore('config', () => {
  const config = ref<WikiConfig | null>(null)
  const loading = ref(false)
  const error = ref<unknown>(null)

  async function load(force = false) {
    if (!force && (config.value || loading.value)) return
    loading.value = true
    error.value = null
    try {
      config.value = await pb
        .collection('wiki_config')
        .getFirstListItem<WikiConfig>('')
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
    return updated
  }

  return { config, loading, error, load, save }
})
