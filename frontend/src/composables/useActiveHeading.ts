import { ref, watch, onUnmounted, nextTick, type Ref } from 'vue'
import type { Heading } from '@/lib/markdown'

// useActiveHeading tracks which section the reader is in: the last heading
// whose top has scrolled above a line 25% down the viewport. Above the first
// heading nothing is active. At the bottom of the page the last heading on
// screen wins, since a short final section can never reach the line and
// picking it from the contents would otherwise leave the previous one lit.
// Used by the TOC sidebar and the phone top bar.
//
// Reading positions on scroll (once per frame) rather than watching headings
// cross a band with an IntersectionObserver: a fast flick or a jump can skip
// the band entirely, and nothing would be active.
export function useActiveHeading(headings: () => Heading[]): Ref<string | null> {
  const active = ref<string | null>(null)
  let els: HTMLElement[] = []
  let frame = 0

  function update() {
    frame = 0
    const line = window.innerHeight * 0.25
    let current: HTMLElement | null = null
    for (const el of els) {
      if (el.getBoundingClientRect().top > line) break
      current = el
    }
    // scrollY > 0: a page that fits on screen is not "at the bottom".
    const atBottom =
      window.scrollY > 0 &&
      window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 2
    if (atBottom) {
      for (const el of els) {
        if (el.getBoundingClientRect().top < window.innerHeight) current = el
      }
    }
    active.value = current?.id ?? null
  }

  function schedule() {
    if (!frame) frame = requestAnimationFrame(update)
  }

  watch(
    headings,
    async (hs) => {
      // Wait for the markdown's new heading nodes to mount before reading them.
      await nextTick()
      els = hs
        .map((h) => document.getElementById(h.slug))
        .filter((el): el is HTMLElement => !!el)
      update()
    },
    { immediate: true, flush: 'post' },
  )

  window.addEventListener('scroll', schedule, { passive: true })
  window.addEventListener('resize', schedule, { passive: true })
  onUnmounted(() => {
    window.removeEventListener('scroll', schedule)
    window.removeEventListener('resize', schedule)
    if (frame) cancelAnimationFrame(frame)
  })
  return active
}
