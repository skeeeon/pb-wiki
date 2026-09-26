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

/** Shape of one hit in the /api/wiki/search response (see internal/api/search.go). */
interface BodySearchHit {
  id: string
  path: string
  title: string
  snippet: string
}

/**
 * useSearch combines two search strategies:
 *
 *   1. Instant client-side filter on the in-memory doc list (id, path, title
 *      only) — no network, runs on every keystroke. Covers title/path hits.
 *   2. Debounced server-side full-text query against /api/wiki/search, backed
 *      by a SQLite FTS5 index (porter stemming). Runs ~300ms after the user
 *      stops typing. Covers content hits and returns a server-built snippet
 *      around the match. The endpoint re-applies path-based access rules, so
 *      restricted docs never surface here.
 *
 * Results are deduped by id and ordered title/path first, body second — the
 * title-match signal beats a body match for the same doc.
 *
 * Each body query reuses the same SDK requestKey, so a newer keystroke
 * auto-cancels the previous in-flight request and rapid typing won't race.
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
    // A 1–2 char prefix matches almost everything under FTS5 (`a*`), so gate
    // on length. Title/path filtering still runs synchronously above, so the
    // user gets instant feedback either way.
    if (current.length < 3) {
      bodyResults.value = []
      bodyLoading.value = false
      return
    }
    bodyLoading.value = true
    timer = window.setTimeout(async () => {
      try {
        // /api/wiki/search does its own access filtering and snippet building,
        // so we just hand it the query. The shared requestKey makes the SDK
        // cancel any prior in-flight search when a newer keystroke arrives.
        const res = await pb.send<{ results: BodySearchHit[] }>('/api/wiki/search', {
          method: 'GET',
          query: { q: current },
          requestKey: 'wiki-search',
        })
        // Race-safety: if the query changed while we were waiting, drop this batch.
        if (current !== q.value) return
        bodyResults.value = res.results.map((h) => ({
          id: h.id,
          path: h.path,
          title: h.title,
          matchType: 'body' as const,
          snippet: h.snippet || undefined,
        }))
      } catch (err) {
        // Auto-cancellation throws — silently ignore those and surface real errors.
        if (!isAbortError(err)) console.error('body search failed', err)
        bodyResults.value = []
      } finally {
        bodyLoading.value = false
      }
    }, 300)
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
    '<mark class="bg-warning/25 text-inherit rounded px-0.5">$&</mark>',
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
