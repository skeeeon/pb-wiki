import PocketBase from 'pocketbase'

// In dev (Vite on :5173), the /api and /_ paths are proxied to PocketBase on
// :8090 by vite.config.ts, so a same-origin client works in both dev and prod
// — the bundle never bakes in a hard-coded backend host.
//
// VITE_PB_URL is an escape hatch for environments where the SPA is hosted
// separately from the API (we don't ship that way, but the override is cheap).
const url =
  import.meta.env.VITE_PB_URL ??
  (import.meta.env.DEV ? 'http://127.0.0.1:5173' : window.location.origin)

export const pb = new PocketBase(url)
