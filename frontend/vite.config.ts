import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    // PWA support — opt-in per user (see frontend/src/lib/pwa.ts).
    //   injectRegister: null  → we register the SW manually so the user's
    //                           account-page toggle controls activation.
    //   registerType: autoUpdate → new SW activates on next navigation.
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: null,
      manifest: {
        name: 'pb-wiki',
        short_name: 'pb-wiki',
        description: 'PocketBase-backed wiki',
        theme_color: '#F9423A',
        background_color: '#ffffff',
        display: 'standalone',
        start_url: '/',
        icons: [
          { src: '/favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' },
          { src: '/logo.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' },
        ],
      },
      workbox: {
        // Precache the SPA shell so the PWA opens offline at start_url '/'.
        // No navigateFallback — we don't intercept navigation requests, so
        // paths we don't own (/_, /api/*, future PB routes) pass straight
        // through to the network. The cost is that a browser refresh of a
        // deep link offline fails; the PWA launch and in-app navigation
        // (via vue-router, client-side) are unaffected.
        globPatterns: ['**/*.{js,css,html,svg,woff2}'],
        runtimeCaching: [
          {
            // Document records — list + view. StaleWhileRevalidate so a
            // cached response renders instantly while a background fetch
            // refreshes the cache. Mutations call invalidateDocsCache() so
            // the editor's own writes aren't served stale.
            urlPattern: ({ url }) =>
              url.pathname.startsWith('/api/collections/documents/records'),
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'pb-wiki-documents',
              expiration: { maxEntries: 200, maxAgeSeconds: 60 * 60 * 24 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            // Uploaded assets (images embedded in markdown).
            urlPattern: ({ url }) => url.pathname.startsWith('/api/files/'),
            handler: 'CacheFirst',
            options: {
              cacheName: 'pb-wiki-files',
              expiration: { maxEntries: 500, maxAgeSeconds: 60 * 60 * 24 * 7 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // PocketBase REST + realtime + file routes.
      '/api': 'http://127.0.0.1:8090',
      // PocketBase admin dashboard.
      '/_': 'http://127.0.0.1:8090',
    },
  },
})
