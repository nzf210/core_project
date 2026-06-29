import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { Unhead } from '@unhead/vue/bundler'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), Unhead().vite()],
})
