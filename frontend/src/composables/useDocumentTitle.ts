import { watch } from 'vue'
import { useRoute } from 'vue-router'

import { useConfigStore } from '@/stores/config'

const FALLBACK = 'pb-wiki'

// Drives document.title from the wiki_config title plus an optional
// per-page title. App.vue calls this with no args as a baseline so every
// navigation resets to "<wiki title>"; route components (e.g. DocView)
// overlay a page-specific title with their own call.
//
// The route is tracked so the baseline re-fires on each navigation —
// otherwise a DocView's "Foo — Wiki" would stick around after we leave
// the page.
export function useDocumentTitle(pageTitle: () => string | null | undefined = () => null) {
  const config = useConfigStore()
  const route = useRoute()
  watch(
    [() => config.config?.title, pageTitle, () => route.fullPath],
    ([wiki, page]) => {
      const w = wiki || FALLBACK
      document.title = page ? `${page} — ${w}` : w
    },
    { immediate: true },
  )
}
