import { ref, computed, watch, type Ref } from 'vue'

import { pb } from '@/lib/pb'
import type { DocumentRecord } from '@/lib/types'

export interface SearchResult {
  id: string
  path: string
  title: string
  /** 'title' or 'path' match: instant, client-side; no snippet. */
  /** 'body' match: server-side, includes a snippet around the first hit. */
  matchType: 'title-path' | 'body'
  snippet?: string
}

/**
 * useSearch combines two search strategies:
 *
 *   1. Instant client-side filter on the in-memory doc list (id, path, title
 *      only) — no network, runs on every keystroke. Covers title/path hits.
 *   2. Debounced server-side query via PocketBase's `~` (substring) filter on
 *      the body column. Runs ~200ms after the user stops typing. Covers
 *      content-only hits and produces a short snippet around the first match.
 *
 * Results are deduped by id and ordered title/path first, body second — the
 * title-match signal beats a body match for the same doc.
 *
 * PB's SDK already auto-cancels duplicate in-flight requests against the same
 * collection method, so rapid typing won't race the results.
 */
export function useSearch(
  query: Ref<string>,
  docsCache: Ref<DocumentRecord[]>,
) {
  const q = computed(() => query.value.trim())
  const isSearching = computed(() => q.value.length > 0)

  // ---- title / path matches: synchronous, in-memory ----
  const titlePathResults = computed<SearchResult[]>(() => {
    if (!q.value) return []
    const needle = q.value.toLowerCase()
    return docsCache.value
      .filter(
        (d) =>
          d.path.toLowerCase().includes(needle) ||
          (d.title ?? '').toLowerCase().includes(needle),
      )
      .map((d) => ({
        id: d.id,
        path: d.path,
        title: d.title,
        matchType: 'title-path' as const,
      }))
  })

  // ---- body matches: debounced server query ----
  const bodyResults = ref<SearchResult[]>([])
  const bodyLoading = ref(false)
  let timer: number | undefined

  watch(q, (current) => {
    if (timer !== undefined) {
      clearTimeout(timer)
      timer = undefined
    }
    if (!current) {
      bodyResults.value = []
      bodyLoading.value = false
      return
    }
    bodyLoading.value = true
    timer = window.setTimeout(async () => {
      try {
        const records = await pb.collection('documents').getList<DocumentRecord>(1, 50, {
          filter: pb.filter('body ~ {:q}', { q: current }),
          sort: '+path',
          fields: 'id,path,title,body',
        })
        // Race-safety: if the query changed while we were waiting, drop this batch.
        if (current !== q.value) return
        bodyResults.value = records.items.map((d) => ({
          id: d.id,
          path: d.path,
          title: d.title,
          matchType: 'body' as const,
          snippet: extractSnippet(d.body ?? '', current),
        }))
      } catch (err) {
        // Auto-cancellation throws — silently ignore those and surface real errors.
        if (!isAbortError(err)) console.error('body search failed', err)
        bodyResults.value = []
      } finally {
        bodyLoading.value = false
      }
    }, 200)
  })

  // Merge + dedup. title-path entries inserted first so they win when a doc
  // matches both.
  const results = computed<SearchResult[]>(() => {
    const seen = new Set<string>()
    const out: SearchResult[] = []
    for (const r of titlePathResults.value) {
      if (seen.has(r.id)) continue
      seen.add(r.id)
      out.push(r)
    }
    for (const r of bodyResults.value) {
      if (seen.has(r.id)) continue
      seen.add(r.id)
      out.push(r)
    }
    return out.slice(0, 50)
  })

  return { isSearching, results, bodyLoading, q }
}

/**
 * extractSnippet returns ~ `before` characters of context before the first
 * occurrence of `q` and `after` characters after it, with ellipses on either
 * side as appropriate. If somehow `q` isn't in `body` (shouldn't happen since
 * the server filter matched it, but defensive), returns the head of the body.
 */
function extractSnippet(body: string, q: string, before = 40, after = 100): string {
  const idx = body.toLowerCase().indexOf(q.toLowerCase())
  if (idx === -1) {
    return body.slice(0, before + after).replace(/\s+/g, ' ').trim()
  }
  const start = Math.max(0, idx - before)
  const end = Math.min(body.length, idx + q.length + after)
  const slice = body.slice(start, end).replace(/\s+/g, ' ').trim()
  return (start > 0 ? '…' : '') + slice + (end < body.length ? '…' : '')
}

/**
 * highlightMatch renders the snippet as HTML with the matched substring
 * wrapped in <mark>. HTML in the snippet is escaped first to prevent the
 * markdown body from injecting tags. Safe to v-html.
 */
export function highlightMatch(snippet: string, q: string): string {
  const escaped = escapeHTML(snippet)
  if (!q) return escaped
  const escapedQ = q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return escaped.replace(
    new RegExp(escapedQ, 'gi'),
    '<mark class="bg-brand-yellow/40 text-inherit rounded px-0.5">$&</mark>',
  )
}

function escapeHTML(s: string): string {
  return s.replace(/[&<>"']/g, (c) => {
    switch (c) {
      case '&':
        return '&amp;'
      case '<':
        return '&lt;'
      case '>':
        return '&gt;'
      case '"':
        return '&quot;'
      default:
        return '&#039;'
    }
  })
}

function isAbortError(err: unknown): boolean {
  if (typeof err !== 'object' || err === null) return false
  const e = err as { isAbort?: boolean; name?: string }
  return e.isAbort === true || e.name === 'AbortError'
}
