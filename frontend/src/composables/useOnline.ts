import { ref } from 'vue'

// Single shared ref reflecting `navigator.onLine`. Components import `online`
// directly and bind it to UI (offline banner, status chips, etc.).
//
// `navigator.onLine` is heuristic: it tells you whether the device has *any*
// network interface, not whether the wiki is reachable. False positives
// (connected but the server is down) still exist, but for the "I disconnected
// wifi" UX this gives an immediate, accurate signal.
export const online = ref(typeof navigator !== 'undefined' ? navigator.onLine : true)

if (typeof window !== 'undefined') {
  window.addEventListener('online', () => {
    online.value = true
  })
  window.addEventListener('offline', () => {
    online.value = false
  })
}
