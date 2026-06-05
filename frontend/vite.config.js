import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import wails from '@wailsio/runtime/plugins/vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [
    vue({
      template: {
        compilerOptions: {
          isCustomElement: (tag) => tag.startsWith('media-'),
        },
      },
    }),
    wails('bindings'),
  ],
  resolve: {
    alias: {
      bindings: fileURLToPath(new URL('./bindings', import.meta.url)),
    },
  },
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
