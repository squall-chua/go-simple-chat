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
    ['btn-primary', 'bg-sky-500 hover:bg-sky-400 text-black font-bold py-2 px-4 rounded-xl transition-all duration-200 shadow-[0_0_20px_rgba(14,165,233,0.3)] hover:shadow-[0_0_30px_rgba(14,165,233,0.5)] active:scale-95 cursor-pointer flex items-center justify-center'],
    ['glass-card', 'bg-slate-900/90 backdrop-blur-2xl border border-white/20 rounded-3xl shadow-[0_20px_50px_rgba(0,0,0,0.5)]'],
    ['input-field', 'bg-black/40 border border-white/10 rounded-xl px-4 py-3 text-white focus:border-sky-500/50 focus:bg-black/60 transition-all outline-none'],
  ],
  theme: {
    colors: {
      primary: '#0EA5E9',
      cta: '#38BDF8',
      background: '#000000',
      surface: '#0F172A',
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
