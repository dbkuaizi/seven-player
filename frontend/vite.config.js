import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return
          }
          if (id.includes('vuetify') || id.includes('@mdi')) {
            return 'vuetify'
          }
          return 'vendor'
        },
      },
    },
  },
  server: {
    host: '127.0.0.1',
    port: 34115,
    strictPort: true,
  },
})
