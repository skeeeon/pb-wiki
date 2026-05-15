import { createApp } from 'vue'
import { createPinia } from 'pinia'

import './style.css'
import App from './App.vue'
import router from './router'
// Side-effect import: initializes the theme on `<html>` before the first
// render so there's no FOUC on first paint.
import './composables/useTheme'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
