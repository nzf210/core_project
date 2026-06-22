import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3401,
    host: '0.0.0.0',
    proxy: {
      // Proxy /admin/* → api-gateway (billing-service endpoints)
      '/admin': {
        target: 'http://localhost:8000',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/admin/, '/api/superadmin/billing')
      },
      // Proxy /api/* → api-gateway
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
      // Proxy /auth/* → api-gateway (login endpoint)
      '/auth': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      },
    },
  },
})
