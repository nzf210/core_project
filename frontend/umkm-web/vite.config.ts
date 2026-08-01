import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3201,
    strictPort: true,
    host: '0.0.0.0',
    proxy: {
      // F060: Proxy /plans to API gateway (billing-service public endpoint)
      '/plans': {
        target: 'http://localhost:8010',
        changeOrigin: true,
        rewrite: (path) => path,
      },
      // F065: Proxy landing-configs to API gateway
      '/landing-configs': {
        target: 'http://localhost:8010',
        changeOrigin: true,
        rewrite: (path) => path,
      },
    },
  },
})
