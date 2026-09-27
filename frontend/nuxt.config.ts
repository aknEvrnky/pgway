// https://nuxt.com/docs/api/configuration/nuxt-config
import Aura from '@primeuix/themes/aura'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  // Static SPA for `nuxt generate` / go:embed. Local `nuxt dev` still works.
  ssr: false,

  css: ['~/assets/css/main.css'],

  modules: [
    '@primevue/nuxt-module',
    '@nuxtjs/tailwindcss',
  ],

  runtimeConfig: {
    public: {
      // Dev (split process): prefer localhost so httpOnly cookies stay same-site
      // with Nuxt on :3000. Embed/generate sets NUXT_PUBLIC_API_BASE="" for
      // same-origin relative /api/v1 calls.
      apiBase: process.env.NUXT_PUBLIC_API_BASE ?? 'http://localhost:8081',
    },
  },

  app: {
    head: {
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
      ],
    },
  },

  vite: {
    optimizeDeps: {
      include: [
        '@vue/devtools-core',
        '@vue/devtools-kit',
      ],
    },
  },

  primevue: {
    options: {
      theme: {
        preset: Aura,
        options: {
          darkModeSelector: '.dark',
        },
      },
    },
  },
})
