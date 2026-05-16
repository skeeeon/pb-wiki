// Minimal service worker. Exists only to satisfy install-as-PWA criteria
// (browsers require a registered SW with a fetch listener). It caches
// nothing, intercepts nothing, and lets every request go to the network.
//
// The activate hook also wipes any caches left behind by earlier
// workbox-generated workers, so users transitioning off the old offline
// setup don't carry stale shells.

self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      await self.clients.claim()
      const keys = await caches.keys()
      await Promise.all(keys.map((k) => caches.delete(k)))
    })(),
  )
})

self.addEventListener('fetch', () => {
  // No-op. Required for installability; intentionally does not call
  // event.respondWith so the browser handles every request normally.
})
