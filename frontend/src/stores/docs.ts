import { defineStore } from 'pinia'
import { ref } from 'vue'

import { pb } from '@/lib/pb'
import type { DocumentRecord } from '@/lib/types'

// useDocsStore owns the lightweight `id,path,title` list that powers the
// sidebar tree and the in-memory search filter. Centralizing it lets
// mutating views (DocEdit save/delete) call `reload()` so the persistent
// Sidebar mounted at <App> level picks up changes without a page refresh.
//
// The list is reloaded on every auth change so that the per-user access
// filter in internal/hooks/documents.go is re-evaluated — otherwise an
// admin's full list would persist into a later viewer/anonymous session.
export const useDocsStore = defineStore('docs', () => {
  const list = ref<DocumentRecord[]>([])
  const loading = ref(false)
  const error = ref<unknown>(null)

  async function load(force = false) {
    if (!force && (list.value.length > 0 || loading.value)) return
    loading.value = true
    error.value = null
    try {
      list.value = await pb.collection('documents').getFullList<DocumentRecord>({
        sort: '+path',
        fields: 'id,path,title',
      })
    } catch (err) {
      error.value = err
    } finally {
      loading.value = false
    }
  }

  pb.authStore.onChange(() => {
    list.value = []
    void load(true)
  })

  return { list, loading, error, load, reload: () => load(true) }
})
