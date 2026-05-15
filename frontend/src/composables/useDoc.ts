import { ref, watchEffect, type Ref } from 'vue'

import { pb } from '@/lib/pb'
import type { DocumentRecord } from '@/lib/types'

// useDoc fetches the document at the given path. The path is reactive: when
// it changes, the doc is refetched. Returns refs that mirror the four states:
//   loading      — request in flight
//   notFound     — server returned 404 (no doc at that path yet)
//   error        — any other failure
//   doc          — the record on success
export function useDoc(path: () => string): {
  doc: Ref<DocumentRecord | null>
  loading: Ref<boolean>
  notFound: Ref<boolean>
  error: Ref<unknown>
  reload: () => Promise<void>
} {
  const doc = ref<DocumentRecord | null>(null)
  const loading = ref(false)
  const notFound = ref(false)
  const error = ref<unknown>(null)

  async function fetchAt(p: string) {
    loading.value = true
    notFound.value = false
    error.value = null
    try {
      doc.value = await pb
        .collection('documents')
        .getFirstListItem<DocumentRecord>(pb.filter('path = {:path}', { path: p }), {
          expand: 'updated_by',
        })
    } catch (err: unknown) {
      if (isClientError(err) && err.status === 404) {
        notFound.value = true
        doc.value = null
      } else {
        error.value = err
        doc.value = null
      }
    } finally {
      loading.value = false
    }
  }

  watchEffect(() => {
    void fetchAt(path())
  })

  return {
    doc,
    loading,
    notFound,
    error,
    reload: () => fetchAt(path()),
  }
}

function isClientError(err: unknown): err is { status: number } {
  return typeof err === 'object' && err !== null && 'status' in err
}
