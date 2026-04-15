// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },
  modules: ['@unocss/nuxt'],
  unocss: {
    uno: true,
    icons: true,
    webFonts: {
      fonts: {
        heading: 'Fira Code:400,500,600,700',
        sans: 'Fira Sans:300,400,500,600,700',
      },
    },
    theme: {
      colors: {
        primary: '#3B82F6', // Trust Blue
        secondary: '#60A5FA',
        cta: '#F97316', // Orange
        background: '#0F172A', // OLED Slate-900
      },
    },
    shortcuts: {
      'glass-card': 'bg-white/5 backdrop-blur-md border border-white/10 rounded-xl px-6 py-4 shadow-xl',
      'btn-primary': 'bg-primary hover:bg-secondary text-white font-medium py-2 px-4 rounded-lg transition-all active:scale-95 cursor-pointer',
      'chat-bubble-self': 'bg-primary text-white rounded-2xl rounded-tr-none px-4 py-2 max-w-[80%] self-end shadow-sm',
      'chat-bubble-peer': 'bg-white/10 text-slate-100 rounded-2xl rounded-tl-none px-4 py-2 max-w-[80%] self-start shadow-sm',
    }
  },
  runtimeConfig: {
    public: {
      apiBase: 'https://localhost:8080',
      wsBase: 'wss://localhost:8080'
    }
  },
  ssr: false // Client-side demo
})
