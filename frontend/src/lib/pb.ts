import PocketBase from 'pocketbase'

// In dev (Vite on :5173), the /api and /_ paths are proxied to PocketBase on
// :8090 by vite.config.ts, so a same-origin client works in both dev and prod
// — the bundle never bakes in a hard-coded backend host.
//
// VITE_PB_URL is an escape hatch for environments where the SPA is hosted
// separately from the API (we don't ship that way, but the override is cheap).
const url = import.meta.env.VITE_PB_URL ?? window.location.origin

export const pb = new PocketBase(url)

// <img> requests cannot send the Authorization header, so the auth token is
// mirrored into a cookie that only /api/files/ receives. The assets download
// hook (internal/hooks/assets.go) reads it to decide whether an image on a
// private page may be served. SameSite=Strict keeps other sites from using it.
pb.authStore.onChange(() => {
  const secure = window.location.protocol === 'https:' ? '; Secure' : ''
  document.cookie = pb.authStore.isValid
    ? `pbwiki_auth=${pb.authStore.token}; Path=/api/files/; SameSite=Strict${secure}`
    : 'pbwiki_auth=; Path=/api/files/; Max-Age=0'
}, true)
