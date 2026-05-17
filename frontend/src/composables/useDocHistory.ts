import { ref, watchEffect, type Ref } from 'vue'

import { pb } from '@/lib/pb'

// Mirrors the JSON returned by GET /api/wiki/history. The before/after fields
// are full snapshots of the documents record (path, title, body, updated_by,
// timestamps, etc.) but we don't tighten the type here — different revisions
// can disagree on shape if the schema has evolved over time, and the diff UI
// only reads body/title anyway.
export interface RevisionUser {
  id: string
  email: string
  name: string
}

export interface Revision {
  id: string
  timestamp: string
  event_type: 'create' | 'update'
  user: RevisionUser | null
  before: Record<string, unknown> | null
  after: Record<string, unknown> | null
}

// useDocHistory fetches the revision list for the document at `path` via the
// pb-wiki history endpoint. The endpoint also enforces access — a 404 means
// either the doc doesn't exist or the caller can't see it; we surface those
// the same way as useDoc.
export function useDocHistory(path: () => string): {
  revisions: Ref<Revision[]>
  loading: Ref<boolean>
  notFound: Ref<boolean>
  error: Ref<unknown>
  reload: () => Promise<void>
} {
  const revisions = ref<Revision[]>([])
  const loading = ref(false)
  const notFound = ref(false)
  const error = ref<unknown>(null)

  async function fetchAt(p: string) {
    loading.value = true
    notFound.value = false
    error.value = null
    try {
      const res = await pb.send<{ revisions: Revision[] }>('/api/wiki/history', {
        method: 'GET',
        query: { path: p },
      })
      revisions.value = res.revisions ?? []
    } catch (err: unknown) {
      if (isClientError(err) && err.status === 404) {
        notFound.value = true
        revisions.value = []
      } else {
        error.value = err
        revisions.value = []
      }
    } finally {
      loading.value = false
    }
  }

  watchEffect(() => {
    void fetchAt(path())
  })

  return {
    revisions,
    loading,
    notFound,
    error,
    reload: () => fetchAt(path()),
  }
}

function isClientError(err: unknown): err is { status: number } {
  return typeof err === 'object' && err !== null && 'status' in err
}
