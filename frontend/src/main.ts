import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { config as mdEditorConfig } from 'md-editor-v3'

import './style.css'
import App from './App.vue'
import router from './router'
import { useConfigStore } from './stores/config'
import { applyCalloutContainers, applyMarkdownExtras, applyYoutubePlugin } from './lib/markdown'

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
  },
})
// Side-effect import: initializes the theme on `<html>` before the first
// render so there's no FOUC on first paint.
import './composables/useTheme'

const app = createApp(App)
app.use(createPinia())

// Eagerly load wiki_config before the router resolves the first route. The
// router's beforeEach guard reads `require_login` from the config store to
// decide whether to redirect anonymous users to /login — if the fetch ran
// lazily, the first paint could leak content from a locked-down wiki for a
// frame. wiki_config is world-readable specifically so this works for
// anonymous visitors. (load() swallows its own errors into config.error.)
await useConfigStore().load()

app.use(router)
app.mount('#app')
