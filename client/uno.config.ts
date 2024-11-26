import {
  defineConfig,
  presetAttributify,
  presetIcons,
  presetUno,
  presetWebFonts,
} from 'unocss'

export default defineConfig({
  shortcuts: [
    ['btn', 'rounded-2xl bg-white py-6 text-2xl text-[#0a0a2a] font-bold shadow-md transition-all duration-150 ease-in-out hover:bg-gray-200 disabled:opacity-50 disabled:bg-gray-200'],
    ['btn-sm', 'rounded-xl bg-white py-2 px-4 text-base text-[#0a0a2a] font-bold shadow-md transition-all duration-150 ease-in-out hover:bg-gray-200 disabled:opacity-50 disabled:bg-gray-200'],
    ['input', 'w-full border-[#3a3a6a] border-2 rounded-2xl bg-[#1a1a4a] py-6 text-center text-2xl text-white font-bold shadow-inner focus:border-[#3a3a6a] focus:outline-none focus:ring-0'],
  ],
  presets: [
    presetUno(),
    presetAttributify(),
    presetIcons({
      scale: 1.2,
      warn: true,
    }),
    presetWebFonts({
      fonts: {
        sans: 'DM Sans',
        serif: 'DM Serif Display',
        mono: 'DM Mono',
      },
    }),
  ],
})
