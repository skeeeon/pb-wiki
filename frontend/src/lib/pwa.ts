import { ref } from 'vue'

// User-controlled offline cache. Opt-in: the SPA never registers a service
// worker unless the user flips the toggle on the account page. When disabled
// we both unregister any existing worker and delete its caches so a flip-off
// truly leaves the device clean.

const STORAGE_KEY = 'pb-wiki:pwa-cache-enabled'

// Reactive ref so the Account page can bind a checkbox to it.
export const pwaEnabled = ref<boolean>(readEnabled())

function readEnabled(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

function writeEnabled(on: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, on ? 'true' : 'false')
  } catch {
    /* ignore — private mode, etc. */
  }
}

// Lazily-loaded register function from virtual:pwa-register so we only pull
// the workbox client glue when the user has actually opted in.
let registerSWPromise: Promise<((opts?: { immediate?: boolean }) => Promise<void>) | null> | null = null

async function loadRegister() {
  if (!registerSWPromise) {
    registerSWPromise = (async () => {
      try {
        // @ts-expect-error — virtual module provided by vite-plugin-pwa at build time.
        const mod = await import('virtual:pwa-register')
        return mod.registerSW as (opts?: { immediate?: boolean }) => Promise<void>
      } catch {
        return null
      }
    })()
  }
  return registerSWPromise
}

export async function applyPwaState(on: boolean): Promise<void> {
  pwaEnabled.value = on
  writeEnabled(on)
  if (on) {
    const register = await loadRegister()
    register?.({ immediate: true })
  } else {
    await unregisterAll()
    await clearCache()
  }
}

export async function clearCache(): Promise<void> {
  if (!('caches' in globalThis)) return
  const keys = await caches.keys()
  await Promise.all(keys.map((k) => caches.delete(k)))
}

async function unregisterAll(): Promise<void> {
  if (!('serviceWorker' in navigator)) return
  const regs = await navigator.serviceWorker.getRegistrations()
  await Promise.all(regs.map((r) => r.unregister()))
}

// Called once at app startup. If the user previously opted in, register the
// SW so offline reads keep working across sessions; otherwise stay dormant.
export async function bootPwa(): Promise<void> {
  if (!pwaEnabled.value) return
  const register = await loadRegister()
  register?.()
}
