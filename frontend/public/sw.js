// Minimal service worker. It caches nothing and has no fetch listener, so
// every request goes straight to the network. Browsers no longer need a
// fetch listener to offer "Install app", and an empty one still makes the
// browser start the worker before every page load.
//
// The activate hook wipes any caches left behind by earlier
// workbox-generated workers, so users transitioning off the old offline
// setup don't carry stale shells. Once those clients have updated, this
// worker can go too.

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
