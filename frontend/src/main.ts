import { createApp } from 'vue'
import { createPinia } from 'pinia'

import './style.css'
import App from './App.vue'
import router from './router'
import { useConfigStore } from './stores/config'
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
