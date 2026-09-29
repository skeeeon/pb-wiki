import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { config as mdEditorConfig } from 'md-editor-v3'

import './style.css'
import App from './App.vue'
import router from './router'
import { useConfigStore } from './stores/config'
import {
  applyCalloutContainers,
  applyFrontmatterPlugin,
  applyMarkdownExtras,
  applyYoutubePlugin,
} from './lib/markdown'

// Teach md-editor-v3's preview pane the same ::: callouts, YouTube embed,
// sub/sup/mark, and image-caption figures our renderer supports. The editor
// already ships anchors, task lists, and !!! admonitions; we only add what's
// missing so the preview matches the saved view.
//
// `md.options.html = false` mirrors the view renderer's posture (see
// lib/markdown.ts). md-editor-v3 instantiates its preview's markdown-it with
// `html: true`, which would let an editor inject <script>/onerror handlers
// that execute in another editor's session when they open the doc to review
// it — privilege escalation within the editor pool. Flipping the option
// here only suppresses raw-HTML *source* tokens; plugin-emitted HTML
// (callouts, figures, YouTube iframes, the editor's own admonitions/tables)
// is unaffected.
mdEditorConfig({
  markdownItConfig(md) {
    md.options.html = false
    applyCalloutContainers(md)
    applyYoutubePlugin(md)
    applyMarkdownExtras(md)
    applyFrontmatterPlugin(md)
  },
})
// Side-effect import: initializes the theme on `<html>` before the first
// render so there's no FOUC on first paint.
import './composables/useTheme'

// After a deploy, a tab opened on the old build still asks for the old
// content-hashed chunks, which the new binary no longer has (the server
// answers 404, see internal/static). Reload to pick up the new build. The
// timestamp stops a reload loop if a chunk is missing from the new build too.
window.addEventListener('vite:preloadError', () => {
  const key = 'pb-wiki:chunk-reload'
  try {
    const last = Number(sessionStorage.getItem(key))
    if (Date.now() - last < 10_000) return
    sessionStorage.setItem(key, String(Date.now()))
  } catch {
    /* storage unavailable — reload anyway */
  }
  window.location.reload()
})

const app = createApp(App)
app.use(createPinia())

// The router's beforeEach guard reads `require_login` from wiki_config to
// decide whether to redirect anonymous users to /login — so the config must
// be present before the first route resolves, or a locked-down wiki could
// leak a frame of content. We seed synchronously from localStorage when
// possible (covers every visit after the first) and only block on the
// network for a truly cold start. A seeded start still refreshes the cache
// in the background, so admin changes to wiki_config show up once that
// request lands (and the router guard sees them from the next navigation).
const config = useConfigStore()
if (config.loadFromStorage()) {
  void config.load(true)
} else {
  await config.load()
}

app.use(router)
app.mount('#app')

// Register a tiny service worker (public/sw.js). It does not cache or
// intercept anything; it only clears caches left by the old offline worker.
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch(() => {
    /* registration failed (e.g. http dev preview); not fatal. */
  })
}
