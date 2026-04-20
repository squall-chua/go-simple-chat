// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },
  srcDir: 'app',
  modules: ['@unocss/nuxt'],
  app: {
    head: {
      title: 'Secure messaging',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Modern, secure, mTLS-based messaging platform.' }
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }
      ]
    }
  },
  runtimeConfig: {
    public: {
      apiBase: 'https://localhost:8080'
    }
  }
})
