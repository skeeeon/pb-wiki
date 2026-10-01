import { onBeforeUnmount, shallowRef, watchEffect, type Ref } from 'vue'
import type { Heading } from '@/lib/markdown'

// The open page's table of contents, for the phone top bar in App.vue. The
// bar sits outside RouterView, so DocView publishes here instead of passing
// anything up through the router.
export interface PageToc {
  headings: Heading[]
  activeSlug: string | null
  pageTitle: string
  // Whether the inline "On this page" box is on screen. The bar offers the
  // contents only once that box has scrolled away, so both never show.
  inlineVisible: boolean
}

const current = shallowRef<PageToc | null>(null)

// publishPageToc keeps the shared TOC in step with `source` until the
// calling component unmounts. Vue unmounts the old route component before
// mounting the next, so a new page never has its TOC cleared by the old one.
export function publishPageToc(source: () => PageToc | null) {
  watchEffect(() => {
    current.value = source()
  })
  onBeforeUnmount(() => {
    current.value = null
  })
}

export function usePageToc(): Readonly<Ref<PageToc | null>> {
  return current
}
