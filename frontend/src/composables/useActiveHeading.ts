import { ref, watch, onUnmounted, nextTick, type Ref } from 'vue'
import type { Heading } from '@/lib/markdown'

// useActiveHeading tracks which heading is currently "in view" as the user
// scrolls, using an IntersectionObserver. Headings count as active while the
// top of their element is within the upper 25% of the viewport; if multiple
// are in that band, the topmost wins. Used by the TOC sidebar to highlight
// the current section.
export function useActiveHeading(headings: () => Heading[]): Ref<string | null> {
  const active = ref<string | null>(null)
  let observer: IntersectionObserver | null = null
  const visible = new Set<string>()

  function disconnect() {
    observer?.disconnect()
    observer = null
    visible.clear()
  }

  watch(
    headings,
    async (hs) => {
      disconnect()
      if (!hs.length) {
        active.value = null
        return
      }
      // Wait for the markdown's new heading nodes to mount before observing.
      await nextTick()
      observer = new IntersectionObserver(
        (entries) => {
          for (const e of entries) {
            if (e.isIntersecting) visible.add(e.target.id)
            else visible.delete(e.target.id)
          }
          const first = hs.find((h) => visible.has(h.slug))
          if (first) active.value = first.slug
        },
        // rootMargin shrinks the viewport to its top 25%: a heading is
        // considered "active" only while it's in that band near the top.
        { rootMargin: '0px 0px -75% 0px', threshold: 0 },
      )
      for (const h of hs) {
        const el = document.getElementById(h.slug)
        if (el) observer.observe(el)
      }
    },
    { immediate: true, flush: 'post' },
  )

  onUnmounted(disconnect)
  return active
}
