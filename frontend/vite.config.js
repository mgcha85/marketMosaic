import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/candle': 'http://localhost:8080',
      '/news': 'http://localhost:8080',
      '/dart': 'http://localhost:8080',
      '/judal': 'http://localhost:8080'
    }
  },
  // Force Rebuild Timestamp: 20260106
})
