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
        // Precache the SPA shell so the app boots offline. Markdown bodies
        // and inline images are handled by the runtime rules below.
        globPatterns: ['**/*.{js,css,html,svg,woff2}'],
        // SPA fallback: deep links that aren't precached should return the
        // shell so vue-router can resolve them client-side.
        navigateFallback: '/index.html',
        // Don't intercept API/auth/realtime/admin paths with the SPA fallback.
        navigateFallbackDenylist: [/^\/api\//, /^\/_\//],
        runtimeCaching: [
          {
            // Document records — list + view. NetworkFirst so users get
            // fresh content online and a cached copy when offline.
            urlPattern: ({ url }) =>
              url.pathname.startsWith('/api/collections/documents/records'),
            handler: 'NetworkFirst',
            options: {
              cacheName: 'pb-wiki-documents',
              networkTimeoutSeconds: 4,
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
