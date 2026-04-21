import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev: proxy /api to the Go backend so the SPA and API share an origin.
// host: true binds to 0.0.0.0 so phones / tablets on the same LAN can connect.
// Production: single-origin via Go's embed.FS (Phase 8).
export default defineConfig({
  plugins: [react()],
  server: {
    host: true, // listen on 0.0.0.0 (LAN accessible); Vite prints LAN URLs on start
    port: 5173,
    strictPort: true, // fail loudly instead of silently switching ports
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: false,
      },
    },
  },
})
