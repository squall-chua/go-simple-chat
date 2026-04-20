import {
  defineConfig,
  presetAttributify,
  presetIcons,
  presetTypography,
  presetUno,
  presetWebFonts,
  transformerDirectives,
  transformerVariantGroup,
} from 'unocss'

export default defineConfig({
  shortcuts: [
    ['btn-primary', 'bg-sky-700 hover:bg-sky-600 text-white font-bold py-2 px-4 rounded-xl transition-all duration-200 shadow-[0_0_20px_rgba(3,105,161,0.2)] active:scale-95 cursor-pointer flex items-center justify-center'],
    ['glass-card', 'bg-[#010409]/90 backdrop-blur-2xl border border-white/5 rounded-3xl shadow-[0_20px_50px_rgba(0,0,0,0.8)]'],
    ['input-field', 'bg-black/20 border border-white/5 rounded-xl px-4 py-3 text-slate-300 focus:border-sky-700/50 focus:bg-black/40 transition-all outline-none'],
  ],
  theme: {
    colors: {
      primary: '#0369A1',
      cta: '#075985',
      background: '#010409',
      surface: '#0d1117',
    },
    fontFamily: {
      sans: 'Inter, ui-sans-serif, system-ui, -apple-system, sans-serif',
      heading: 'Outfit, sans-serif',
    },
  },
  presets: [
    presetUno(),
    presetAttributify(),
    presetIcons({
      scale: 1.2,
    }),
    presetTypography(),
    presetWebFonts({
      fonts: {
        sans: 'Inter:400,500,600,700',
        heading: 'Outfit:600,700,800',
      },
    }),
  ],
  transformers: [
    transformerDirectives(),
    transformerVariantGroup(),
  ],
})
